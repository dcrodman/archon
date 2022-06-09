// Commands:
//     capture (default): starts a server waiting for packets to be submitted
// 	   compact: generates a more human-readable version of session data (useful for tools like diff)
// 	   summarize: similar to compact but only the packet types are included
// 	   aggregate: combines multiple files into a nicely formatted Markdown file

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "analyzer",
	Short: "Packet sniffer for PSOBB servers",
	Long: `This utility stands up an HTTP server that receives packet data from a PSOBB server,
	persists it in a common format, and can perform some basic analysis. Primarily
	intended for comparing which packets are exchanged between the client and different
	server implementations for comparison with tools like diff.
	
	Note that this utility is mostly only useful in the context of local development
	due to the overhead incurred by the servers having to send every packet over an HTTP POST.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
