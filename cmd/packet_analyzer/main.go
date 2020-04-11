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
//
// In order to use the capture utility, there must be a value set for packet_analyzer_address
// in Archon's config.yaml.
package main

import (
	"flag"
	_ "github.com/dcrodman/archon"
)

// Requests from the server implementations containing the packet data.
type PacketRequest struct {
	// The server from which this request has originated.
	ServerName string
	// (Optional) identifier to append for this session.
	SessionID string
	// The origin of the packet. This will usually be "client" or one of the server names.
	Source string
	// The destination of the packet. This will usually be "client" or one of the server names.
	Destination string
	// The contents of the packet.
	Contents []int
}

// This tool's representation of a packet received from a server.
type Packet struct {
	Source      string
	Destination string

	Type string
	Size string

	Contents          []int
	PrintableContents []string
}

// File format of the persisted session data.
type SessionFile struct {
	SessionID string
	Packets   []Packet
}

func main() {
	flag.Parse()

	command := ""
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	switch command {
	case "compact":
		compactFiles()
	case "summarize":
		summarizeFiles()
	default:
		startCapturing()
	}
}
