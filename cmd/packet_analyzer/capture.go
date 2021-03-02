package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
	"unicode"
)

// HTTP request from the server implementations containing the packet data.
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
	// Packet timestamp
	Timestamp time.Time
}

var (
	// Mapping of server names to channels of PacketRequests acting as queues.
	packetChannels = make(map[string]chan *PacketRequest)
	// Mapping of server names to the ordered packets.
	packetQueues = make(map[string][]Packet)
)

// startCapturing spins up an HTTP handler to await packet submissions from one
// or more running servers. On exit it will write the contents of each session
// to a file for you to do what you will.
func startCapturing(serverAddr, folder string, httpPort, tcpPort, managePort int, auto bool) {
	// Register a signal handler to dump the packet lists before exiting.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGCONT, syscall.SIGTERM)
	go captureExitHandler(signalChan, folder, auto)

	go listenForTCPPackets(serverAddr, tcpPort)
	if managePort != 0 {
		go startManageServer(serverAddr, managePort)
	}
	listenForHTTPPackets(serverAddr, httpPort)
}

func startManageServer(serverAddr string, managePort int) {
	addr := fmt.Sprintf("%s:%d", serverAddr, managePort)
	http.HandleFunc("/manage", manage)
	http.HandleFunc("/manage/ui", ui)
	fmt.Println("manage API is listening on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Println(err)
	}
}

func manage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		sessionName := r.FormValue("session")
		if sessionName != "" {
			for _, p := range packetQueues[sessionName] {
				if err := writePacketToFile(bufio.NewWriter(w), &p); err != nil {
					fmt.Printf("unable to write packet to %s: %s\n", sessionName, err)
				}
			}
		} else {
			var sessionNames []string
			for k := range packetQueues {
				sessionNames = append(sessionNames, k)
			}
			names, _ := json.Marshal(sessionNames)
			w.Write(names)
		}
	case "DELETE":
		packetQueues = make(map[string][]Packet)
	default:
		fmt.Fprintf(w, "Sorry, only GET and DELETE methods are supported.")
	}
}

func ui(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("./templates/index.html")
	var sessionNames []string
	for k := range packetQueues {
		sessionNames = append(sessionNames, k)
	}
	err := tmpl.Execute(w, sessionNames)
	if err != nil {
		fmt.Printf("unable to execute template %v", err)
	}
}

// Write all of our current session information to files in the local directory.
func captureExitHandler(c chan os.Signal, folder string, auto bool) {
	<-c
	fmt.Println("flushing session data to files...")

	for sessionName, packetList := range packetQueues {
		sessionFile := SessionFile{
			SessionID: sessionName,
			Packets:   packetList,
		}

		filename := path.Join(".", folder, sessionName) + ".session"
		b, _ := json.MarshalIndent(sessionFile, "", "\t")

		if err := ioutil.WriteFile(filename, b, 0666); err != nil {
			fmt.Printf("failed to save session data: %s\n", err.Error())
			break
		}

		fmt.Println("wrote", filename)

		if auto {
			if _, err := summarizeSession(filename); err != nil {
				fmt.Printf("unable to generate summary for session %s: %s\n", filename, err)
			}
			if _, err = compactSession(filename); err != nil {
				fmt.Printf("unable to compact session %s: %s\n", filename, err)
			}
		}
	}

	os.Exit(0)
}

func listenForTCPPackets(serverAddr string, tcpPort int) {
	tcpAddr := fmt.Sprintf("%s:%d", serverAddr, tcpPort)

	fmt.Println("listening for packets via TCP on", tcpAddr)

	l, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		fmt.Println("failed to start TCP listener: ", err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("failed to accept connection:", err.Error())
			continue
		}

		// Sloppily grab the data out of the packet and convert it into a
		// PacketRequest so that the handler paths converge.
		packet, err := ioutil.ReadAll(conn)
		if err != nil {
			fmt.Println("failed to read from client:", err.Error())
			break
		}

		packetContents := packet[40:]
		contents := make([]int, len(packetContents))

		for i := 0; i < len(packetContents); i++ {
			contents[i] = int(packetContents[i])
		}

		recordPacket(&PacketRequest{
			ServerName:  readStringFromData(packet[0:10]),
			SessionID:   readStringFromData(packet[10:20]),
			Source:      readStringFromData(packet[20:30]),
			Destination: readStringFromData(packet[30:40]),
			Contents:    contents,
			Timestamp:   time.Now(),
		})
		_ = conn.Close()
	}
}

// Returns the byte-encoded string without the zero padding.
func readStringFromData(data []byte) string {
	return strings.TrimRight(string(data), "�")
}

func listenForHTTPPackets(serverAddr string, httpPort int) {
	addr := fmt.Sprintf("%s:%d", serverAddr, httpPort)

	fmt.Println("listening for packets via HTTP on", addr)

	http.HandleFunc("/", packetHandler)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Println(err)
	}
}

// Request handler responsible only for parsing the packet request and then
// throwing it onto a queue for async processing.
func packetHandler(w http.ResponseWriter, r *http.Request) {
	p := &PacketRequest{}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		fmt.Printf("error reading JSON from data: %s\n", err.Error())
		return
	}
	p.Timestamp = time.Now()
	recordPacket(p)
}

func recordPacket(p *PacketRequest) {
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
			Timestamp:         pr.Timestamp,
		}

		fmt.Printf("received packet %s from %s\n", p.Type, pr.ServerName)

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
