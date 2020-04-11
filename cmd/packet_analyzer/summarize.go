package main

import (
	"flag"
	"fmt"
	"os"
)

func summarizeFiles() {
	if flag.NArg() == 1 {
		fmt.Println("usage: summarize [file1.session] [file2.session]")
		return
	}

	for i := 1; i < flag.NArg(); i++ {
		session, err := parseSessionDataFromFile(flag.Arg(i))
		if err != nil {
			fmt.Printf("unable read file %s: %s\n", flag.Arg(i), err)
			os.Exit(1)
		}

		filename := generateSummaryFile(session)

		fmt.Println("wrote", filename)
	}
}

func generateSummaryFile(session *SessionFile) string {
	filename := fmt.Sprintf("%s_summary.txt", session.SessionID)

	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("unable to write to %s: %s\n", filename, err)
		os.Exit(1)
	}

	for _, p := range session.Packets {
		if err := writePacketHeaderToFile(f, &p); err != nil {
			fmt.Printf("unable to write packet header to %s: %s\n", filename, err)
			os.Exit(1)
		}
	}

	return filename
}
