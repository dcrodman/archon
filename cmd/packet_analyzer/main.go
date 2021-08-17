// This utility stands up an HTTP server that receives packet data from a PSOBB server,
// persists it in a common format, and can perform some basic analysis. Primarily
// intended for comparing which packets are exchanged between the client and different
// server implementations for comparison with tools like diff.
//
// Note that this utility is mostly only useful in the context of local development,
// due if nothing else to the overhead incurred by the servers having to send every
// packet over an HTTP POST.
//
// Commands:
//     capture (default): starts a server waiting for packets to be submitted
//	   compact: generates a more human-readable version of session data (useful for tools like diff)
//	   summarize: similar to compact but only the packet types are included
//	   aggregate: combines multiple files into a nicely formatted Markdown file
//
// In order to use the capture utility, there must be a value set for packet_analyzer_address
// in Archon's config.yaml.
package main

import (
	"flag"
	"fmt"
	"time"
)

// Packet is this tool's representation of a packet received from a server.
type Packet struct {
	Source      string
	Destination string

	Type string
	Size string

	Contents          []int
	PrintableContents []string

	Timestamp time.Time
}

var (
	address  = flag.String("addr", "localhost", "Address and port on which to bind")
	httpPort = flag.Int("http", 8081, "Port on which the HTTP service should listen")
	tcpPort  = flag.Int("tcp", 8082, "Port on which the raw TCP service should listen")
	uiPort   = flag.Int("ui", 0, "Port on which HTTP UI server (disabled by default)")

	auto   = flag.Bool("auto", false, "Automatically runs both compact and summarize on generated session file")
	folder = flag.String("folder", "", "Folder to which the resulting session files will be written")
)

func main() {
	flag.Parse()

	if *uiPort > 0 {
		go startManageServer(*address, *uiPort)
	}

	command := "capture"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	switch command {
	case "capture":
		startCapturing(*address, *folder, *httpPort, *tcpPort, *auto)
	case "summarize":
		summarizeFiles()
	case "compact":
		compactFiles()
	case "aggregate":
		aggregateFiles()
	default:
		fmt.Printf("unrecognized command: %s", command)
	}
}
