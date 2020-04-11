package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const PacketLineLength = 16

func compactFiles() {
	if flag.NArg() == 1 {
		fmt.Println("usage: diff [file1.session] [file2.session]")
		return
	}

	for i := 1; i < flag.NArg(); i++ {
		session, err := parseSessionDataFromFile(flag.Arg(i))
		if err != nil {
			fmt.Printf("unable read file %s: %s\n", flag.Arg(i), err)
			os.Exit(1)
		}

		filename := generateCompactedFile(session)

		fmt.Println("wrote", filename)
	}
}

func parseSessionDataFromFile(filename string) (*SessionFile, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var s SessionFile
	if err := json.Unmarshal(bytes, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func generateCompactedFile(s1 *SessionFile) string {
	filename := fmt.Sprintf("%s.diff", s1.SessionID)

	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("unable to write to %s: %s\n", filename, err)
		os.Exit(1)
	}

	for _, p := range s1.Packets {
		if err := writePacketToFile(f, &p); err != nil {
			fmt.Printf("unable to write packet to %s: %s\n", filename, err)
			os.Exit(1)
		}
	}

	return filename
}

func writePacketToFile(f *os.File, p *Packet) error {
	data := packetToBytes(p.Contents)
	pktLen := len(p.Contents)

	size, _ := strconv.ParseInt(p.Size, 10, 16)
	header := fmt.Sprintf("Type: %s\nSize: %s (%d) bytes\n", p.Type, p.Size, size)

	if _, err := f.WriteString(header); err != nil {
		return err
	}

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
			line.WriteString("  ")
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
