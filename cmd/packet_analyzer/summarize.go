package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func summarizeFiles() {
	if flag.NArg() == 0 {
		fmt.Println("usage: -summarize [file.session...]")
		return
	}

	for i := 0; i < flag.NArg(); i++ {
		sessionFilename := flag.Arg(i)
		session, err := parseSessionDataFromFile(sessionFilename)
		if err != nil {
			fmt.Printf("unable read file %s: %s\n", sessionFilename, err)
			os.Exit(1)
		}

		filename := fmt.Sprintf("%s_summary.txt", strings.Replace(sessionFilename, ".session", "", 1))
		generateSummaryFile(filename, session)

		fmt.Println("wrote", filename)
	}
}

func generateSummaryFile(filename string, session *SessionFile) {
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
}
