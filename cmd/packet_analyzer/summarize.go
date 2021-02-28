package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/pkg/errors"
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
		return "", errors.Wrap(err, "unable to parse session file")
	}
	filename := fmt.Sprintf("%s_summary.txt", strings.Replace(sessionFilename, ".session", "", 1))
	err = generateSummaryFile(filename, session)
	if err != nil {
		return "", errors.Wrap(err, "unable to generate summary file")
	}
	return filename, nil
}

func generateSummaryFile(filename string, session *SessionFile) error {
	f, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "Unable to create file "+filename)
	}

	for _, p := range session.Packets {
		if err := writePacketHeaderToFile(bufio.NewWriter(f), &p); err != nil {
			return errors.Wrap(err, "unable to write packet header to "+filename)
		}
	}
	return nil
}
