package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Generates a shortened, human-readable view of the session files",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		summarizeFiles(args)
	},
}

func init() {
	rootCmd.AddCommand(summarizeCmd)
}

func summarizeFiles(args []string) {
	for _, sessionFilename := range args {
		sum, err := summarizeSession(sessionFilename)
		if err != nil {
			fmt.Printf("unable to generate summary for session %s: %s\n", sessionFilename, err)
			return
		}
		fmt.Println("wrote", sum)
	}
}

func summarizeSession(sessionFilename string) (string, error) {
	session, err := parseSessionDataFromFile(sessionFilename)
	if err != nil {
		return "", fmt.Errorf("unable to parse session file: %v", err)
	}
	filename := fmt.Sprintf("%s_summary.txt", strings.Replace(sessionFilename, ".session", "", 1))
	err = generateSummaryFile(filename, session)
	if err != nil {
		return "", fmt.Errorf("unable to generate summary file %s: %v", filename, err)
	}
	return filename, nil
}

func generateSummaryFile(filename string, session *SessionFile) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", filename, err)
	}

	for _, p := range session.Packets {
		if err := writePacketHeaderToFile(bufio.NewWriter(f), &p); err != nil {
			return fmt.Errorf("unable to write packet header to %s: %v", filename, err)
		}
	}
	return nil
}
