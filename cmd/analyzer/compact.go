package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var compactCmd = &cobra.Command{
	Use:   "compact",
	Short: "Generates a human-readable view of the session files",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		compactFiles(args)
	},
}

func init() {
	rootCmd.AddCommand(compactCmd)
}

const PacketLineLength = 16

func compactFiles(args []string) {
	for _, sessionFile := range args {
		compact, err := compactSession(sessionFile)
		if err != nil {
			fmt.Printf("unable to compact session %s: %s\n", sessionFile, err)
			return
		}
		fmt.Println("wrote", compact)
	}
}

func compactSession(sessionFilename string) (string, error) {
	session, err := parseSessionDataFromFile(sessionFilename)
	if err != nil {
		return "", fmt.Errorf("unable to read file: %v", err)
	}
	filename := fmt.Sprintf("%s_compact.txt", strings.Replace(sessionFilename, ".session", "", 1))
	err = generateCompactedFile(filename, session)
	if err != nil {
		return "", fmt.Errorf("unable to generate compact file: %v", err)
	}
	return filename, nil
}

func parseSessionDataFromFile(filename string) (*SessionFile, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var s SessionFile
	if err := json.Unmarshal(bytes, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func generateCompactedFile(filename string, session *SessionFile) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", filename, err)
	}

	for _, p := range session.Packets {
		if err := writePacketToFile(bufio.NewWriter(f), &p); err != nil {
			return fmt.Errorf("unable to write packet to file %s: %v", filename, err)
		}
	}
	return nil
}

func writePacketToFile(f *bufio.Writer, p *Packet) error {
	if err := writePacketHeaderToFile(f, p); err != nil {
		return err
	}

	if err := writePacketBodyToFile(f, p); err != nil {
		return err
	}

	_, _ = f.WriteString("\n\n")
	return f.Flush()
}

func writePacketHeaderToFile(f *bufio.Writer, p *Packet) error {
	size, _ := strconv.ParseInt(p.Size, 16, 16)
	header := fmt.Sprintf(
		"%s -> %s\nType: %s\nSize: %s (%d) bytes\n",
		p.Source, p.Destination, p.Type, p.Size, size,
	)

	_, err := f.WriteString(header)
	return err
}

func writePacketBodyToFile(f *bufio.Writer, p *Packet) error {
	data := packetToBytes(p.Contents)
	pktLen := len(p.Contents)

	for rem, offset := pktLen, 0; rem > 0; rem -= PacketLineLength {
		var line string

		if rem < PacketLineLength {
			line = buildPacketLine(data[(pktLen-rem):pktLen], rem, offset)
		} else {
			line = buildPacketLine(data[offset:offset+PacketLineLength], PacketLineLength, offset)
		}
		offset += PacketLineLength

		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

// Build one line of formatted packet data.
func buildPacketLine(data []uint8, length int, offset int) string {
	var line strings.Builder

	line.WriteString(fmt.Sprintf("(%04X) ", offset))

	for i, j := 0, 0; i < length; i++ {
		if j == 8 {
			// Visual aid - spacing between groups of 8 bytes.
			j = 0
			// line.WriteString("  ")
		}

		line.WriteString(fmt.Sprintf("%02x ", data[i]))
		j++
	}

	// Fill in rest of the line gap if we don't have enough bytes.
	for i := length; i < PacketLineLength; i++ {
		if i == 8 {
			line.WriteString("  ")
		}
		line.WriteString("   ")
	}
	line.WriteString("    ")

	// Display the print characters as-is, others as periods.
	for i := 0; i < length; i++ {
		c := data[i]

		if strconv.IsPrint(rune(c)) {
			line.WriteString(string(data[i]))
		} else {
			line.WriteString(".")
		}
	}

	line.WriteString("\n")
	return line.String()
}
