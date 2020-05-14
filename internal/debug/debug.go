package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/server"
	"github.com/spf13/viper"
	"net/http"
	"runtime/pprof"
	"time"
)

// Enabled returns whether or not the server was set to debug mode.
func Enabled() bool {
	return viper.GetBool("debug_mode")
}

// If the server was configured in debug mode, this function will launch an HTTP server
// that responds with pprof output containing the stack traces of all running goroutines.
func StartPprofServer() {
	webPort := viper.GetString("web.http_port")

	fmt.Println("opening debug port on " + webPort)
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		pprof.Lookup("goroutine").WriteTo(resp, 1)
	})

	http.ListenAndServe(":"+webPort, nil)
}

// SendServerPacketToAnalyzer makes an http request to a packet_analyzer
// instance with the packet data, reporting it as a server to client message.
func SendServerPacketToAnalyzer(c server.Client, packetBytes []byte, size uint16) {
	sendToPacketAnalyzer(c, packetBytes, int(size), "server", "client")
}

// SendServerPacketToAnalyzer makes an http request to a packet_analyzer
// instance with the packet data, reporting it as a client to server message.
func SendClientPacketToAnalyzer(c server.Client, packetBytes []byte, size uint16) {
	sendToPacketAnalyzer(c, packetBytes, int(size), "client", "server")
}

func sendToPacketAnalyzer(c server.Client, packetBytes []byte, size int, source, destination string) {
	if !viper.IsSet("packet_analyzer_address") {
		return
	}

	cbytes := make([]int, size)
	for i := 0; i < size; i++ {
		cbytes[i] = int(packetBytes[i])
	}

	serverType := c.DebugInfo()["server_type"].(string)

	packet := struct {
		ServerName  string
		SessionID   string
		Source      string
		Destination string
		Contents    []int
	}{
		"archon", serverType, source, destination, cbytes[:size],
	}

	reqBytes, _ := json.Marshal(&packet)
	httpClient := http.Client{Timeout: time.Second}

	// We don't care if the packets don't get through.
	r, err := httpClient.Post(
		"http://"+viper.GetString("packet_analyzer_address"),
		"application/json",
		bytes.NewBuffer(reqBytes),
	)

	if err != nil {
		archon.Log.Warn("failed to send packet to analyzer: ", err)
	} else if r.StatusCode != 200 {
		archon.Log.Warn("failed to send packet to analyzer: ", r.Body)
	}
}
