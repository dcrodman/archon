package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"unicode"
)

var (
	// Mapping of server names to channels of PacketRequests acting as queues.
	packetChannels = make(map[string]chan *PacketRequest)
	// Mapping of server names to the ordered packets.
	packetQueues = make(map[string][]Packet)
)

// startCapturing spins up an HTTP handler to await packet submissions from one
// or more running servers. On exit it will write the contents of each session
// to a file for you to do what you will.
func startCapturing() {
	// Register a signal handler to dump the packet lists before exiting.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	go captureExitHandler(signalChan)

	http.HandleFunc("/", packetHandler)

	serverAddr := fmt.Sprintf("%s:7000", viper.GetString("external_ip"))
	fmt.Println("starting session_server on", serverAddr)

	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		fmt.Println(err)
	}
}

// Write all of our current session information to files in the local directory.
func captureExitHandler(c chan os.Signal) {
	<-c
	fmt.Println("flushing session data to files...")

	for sessionName, packetList := range packetQueues {
		sessionFile := SessionFile{
			SessionID: sessionName,
			Packets:   packetList,
		}

		filename := sessionName + ".session"
		bytes, _ := json.MarshalIndent(sessionFile, "", "\t")

		if err := ioutil.WriteFile(filename, bytes, 0666); err != nil {
			fmt.Printf("failed to save session data: %s\n", err.Error())
			break
		}

		fmt.Println("wrote", filename)
	}

	os.Exit(0)
}

// Request handler responsible only for parsing the packet request and then
// throwing it onto a queue for async processing.
func packetHandler(w http.ResponseWriter, r *http.Request) {
	p := &PacketRequest{}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		fmt.Printf("error reading JSON from data: %s\n", err.Error())
		return
	}

	fmt.Println("received packet from", p.ServerName)

	channelKey := key(p.ServerName, p.SessionID)
	pc, ok := packetChannels[channelKey]

	if !ok {
		// Create the channel and start a goroutine to process it.
		pc = make(chan *PacketRequest, 50)
		packetChannels[channelKey] = pc
		go processPackets(pc)
	}

	pc <- p
}

// return a key composed of the server name and the optional session ID.
func key(serverName, sessionID string) string {
	if sessionID == "" {
		return serverName
	}
	return fmt.Sprintf("%s-%s", serverName, sessionID)
}

// Continuously spins on a channel, reading packets and appending them
// to the list of packets for the corresponding server.
func processPackets(pc chan *PacketRequest) {
	for {
		pr := <-pc
		if pr == nil {
			break
		}

		headerBytes := packetToBytes(pr.Contents[:4])
		pSize := int(binary.LittleEndian.Uint16(headerBytes[0:2]))
		pType := int(binary.LittleEndian.Uint16(headerBytes[2:4]))

		p := Packet{
			Source:            pr.Source,
			Destination:       pr.Destination,
			Type:              fmt.Sprintf("%04X", pType),
			Size:              fmt.Sprintf("%04X", pSize),
			Contents:          pr.Contents,
			PrintableContents: convertPrintableContents(pr.Contents),
		}

		channelKey := key(pr.ServerName, pr.SessionID)
		if _, ok := packetQueues[channelKey]; !ok {
			packetQueues[channelKey] = make([]Packet, 0)
		}

		packetQueues[channelKey] = append(packetQueues[channelKey], p)
	}
}

// Utility method that converts the packet contents to a slice of bytes
// since that's what the servers are actually sending.
func packetToBytes(packet []int) []byte {
	bytes := make([]byte, 0)

	for i := 0; i < len(packet); i++ {
		bytes = append(bytes, byte(packet[i]))
	}

	return bytes
}

// Convert all of the bytes in the packet to readable characters if possible.
func convertPrintableContents(packetBytes []int) []string {
	r := make([]string, len(packetBytes))

	for i, b := range packetBytes {
		if unicode.IsPrint(rune(packetBytes[i])) {
			r[i] = string(b)
		} else {
			r[i] = "."
		}
	}

	return r
}
