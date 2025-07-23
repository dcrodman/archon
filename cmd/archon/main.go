package main

import (
	"fmt"

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

	patchCmd.Flags().StringVarP(&NewAddressFlag, "address", "a", "127.0.0.1", "The new address or IPv4 address")
	patchCmd.Flags().StringVarP(&ExeVersionFlag, "version", "v", "TethVer12513", "Version of the PSOBB client")

	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(patchCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
