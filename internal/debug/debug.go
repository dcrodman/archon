package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/spf13/viper"

	"github.com/dcrodman/archon"
)

type packetAnalyzerRequest struct {
	ServerName  string
	SessionID   string
	Source      string
	Destination string
	Contents    []int
}

var packetAnalyzerChan = make(chan packetAnalyzerRequest, 10)

// Enabled returns whether or not the server was set to debug mode.
func Enabled() bool {
	return viper.GetBool("debugging.enabled")
}

// StartUtilities spins off the services associated with debug mode.
func StartUtilities() {
	startPprofServer()

	if packetAnalyzerEnabled() {
		go startAnalyzerExporter()
	}
}

func packetAnalyzerEnabled() bool {
	return PacketAnalyzerAddress() != ""
}

func PacketAnalyzerAddress() string {
	return viper.GetString("debugging.packet_analyzer_address")
}

func startAnalyzerExporter() {
	for {
		packet := <-packetAnalyzerChan

		reqBytes, _ := json.Marshal(&packet)
		httpClient := http.Client{Timeout: time.Second}

		// We don't care if the packets don't get through.
		r, err := httpClient.Post(
			"http://"+PacketAnalyzerAddress(),
			"application/json",
			bytes.NewBuffer(reqBytes),
		)

		if err != nil {
			archon.Log.Warn("failed to send packet to analyzer: ", err)
		} else if r.StatusCode != 200 {
			archon.Log.Warn("failed to send packet to analyzer: ", r.Body)
		}
	}
}

// This function starts the default pprof HTTP server that can be accessed via localhost
// to get runtime information about archon. See https://golang.org/pkg/net/http/pprof/
func startPprofServer() {
	listenerAddr := fmt.Sprintf("localhost:%s", viper.GetString("debugging.pprof_port"))
	archon.Log.Infof("starting pprof server on %s", listenerAddr)

	go func() {
		if err := http.ListenAndServe(listenerAddr, nil); err != nil {
			archon.Log.Infof("error starting pprof server: %s", err)
		}
	}()
}

// SendServerPacketToAnalyzer makes an http request to a packet_analyzer
// instance with the packet data, reporting it as a server to client message.
func SendServerPacketToAnalyzer(debugInfo map[string]interface{}, packetBytes []byte, size uint16) {
	sendToPacketAnalyzer(debugInfo, packetBytes, int(size), "server", "client")
}

// SendServerPacketToAnalyzer makes an http request to a packet_analyzer
// instance with the packet data, reporting it as a client to server message.
func SendClientPacketToAnalyzer(debugInfo map[string]interface{}, packetBytes []byte, size uint16) {
	sendToPacketAnalyzer(debugInfo, packetBytes, int(size), "client", "server")
}

func sendToPacketAnalyzer(debugInfo map[string]interface{}, packetBytes []byte, size int, source, destination string) {
	if !packetAnalyzerEnabled() {
		return
	}

	cbytes := make([]int, size)
	for i := 0; i < size; i++ {
		cbytes[i] = int(packetBytes[i])
	}

	serverName := debugInfo["server_type"].(string)

	packetAnalyzerChan <- packetAnalyzerRequest{
		"archon", serverName, source, destination, cbytes[:size],
	}
}
