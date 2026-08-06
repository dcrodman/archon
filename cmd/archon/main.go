package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	archon "github.com/dcrodman/archon/internal"
	"github.com/spf13/cobra"
)

var ConfigFlag string

func main() {
	rootCmd := &cobra.Command{
		Use:   "archon",
		Short: "Archon PSOBB server and related tools",
		Run:   ServerCommand,
	}
	rootCmd.PersistentFlags().StringVarP(&ConfigFlag, "config", "c", "", "Path to the server config/data directory")

	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountDeleteCmd)
	accountDeleteCmd.Flags().BoolVar(&PermanentFlag, "permanent", false, "Permanently delete the account (as opposed to a soft delete)")

	rootCmd.AddCommand(accountCmd)

	patchCmd.Flags().StringVarP(&NewAddressFlag, "address", "a", "127.0.0.1", "The new address or IPv4 address")
	patchCmd.Flags().StringVarP(&ExeVersionFlag, "version", "v", "TethVer12513", "Version of the PSOBB client")

	rootCmd.AddCommand(patchCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

// The server command is the main entrypoint for running archon. It takes
// care of initializing everything as well as running as many servers are
// needed for a fully functional server backend.
func ServerCommand(cmd *cobra.Command, args []string) {
	archon.InitConfig(ConfigFlag)
	fmt.Println("loaded configuration from", archon.Config.FilePath)

	// Change to the same directory as the config file so that any relative
	// paths in the config file will resolve.
	if ConfigFlag != "" {
		if err := os.Chdir(ConfigFlag); err != nil {
			fmt.Println("error changing to config directory:", err)
			os.Exit(1)
		}
	}

	// Start any debug utilities if we're configured to do so.
	if archon.Config.Debugging.PprofEnabled {
		StartPprofServer(archon.Config.Debugging.PprofPort)
	}

	// Set up the logger that will be used for server activity.
	if err := archon.InitLogger(); err != nil {
		fmt.Println("exiting; failed to initialize logger:", err)
		return
	}

	// Bind the Controller to one top-level server context so that we can shut down cleanly.
	ctx, cancel := context.WithCancel(context.Background())

	// Register a SIGTERM handler so that Ctrl-C will shut the servers down gracefully.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go exitHandler(cancel, c)

	// Start up the controller to handle all of the resources and server init.
	fmt.Println("starting server")
	archon.Start(ctx)
	fmt.Println("shut down")
}

var receivedSignal atomic.Bool

func exitHandler(cancelFn func(), c chan os.Signal) {
	for {
		<-c
		// Give the server a chance to shut down after the first ctrl-c.
		if !receivedSignal.Load() {
			fmt.Println("waiting to shut down gracefully...")
			cancelFn()
			receivedSignal.Store(true)
			continue
		}

		// If the user does it again, just exit without waiting.
		fmt.Println("hard exiting (killed)")
		os.Exit(0)
	}
}

// This function starts the default pprof HTTP server that can be accessed via localhost
// to get runtime information about archon. See https://golang.org/pkg/net/http/pprof/
func StartPprofServer(pprofPort int) {
	listenerAddr := fmt.Sprintf("localhost:%d", pprofPort)
	fmt.Println("starting pprof server on", listenerAddr)

	go func() {
		if err := http.ListenAndServe(listenerAddr, nil); err != nil {
			fmt.Println("error starting pprof server:")
		}
	}()
}
