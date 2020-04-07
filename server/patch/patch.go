package patch

import (
	"fmt"
	"github.com/dcrodman/archon"
	crypto "github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/server/internal"
	"github.com/spf13/viper"
	"strconv"
	"sync"
)

// Convert the welcome message to UTF-16LE and cache it. PSOBB expects this prefix to the message,
//not completely sure why. Language perhaps?

var (
	messageBytes []byte
	messageInit  sync.Once
)

func GetWelcomeMessage() ([]byte, uint16) {
	messageInit.Do(func() {
		messageBytes = internal.ConvertToUtf16(viper.GetString("patch_server.welcome_message"))

		if len(messageBytes) > (1 << 16) {
			archon.Log.Warn("patch server welcome message exceeds 65,000 characters")
			messageBytes = messageBytes[:1<<16-2]
		}

		messageBytes = append([]byte{0xFF, 0xFE}, messageBytes...)
	})

	return messageBytes, uint16(len(messageBytes))
}

// Copyright message expected by the client for the patch welcome.
var copyright = []byte("Patch Server. Copyright SonicTeam, LTD. 2001")

// PatchServer is the sub-server that acts as the first point of contact for a client. Its
// only real job is to send the client a welcome message and then send the address of DataServer.
type PatchServer struct {
	name string
	port string
	// Parsed representation of the login port.
	dataRedirectPort uint16
}

func NewPatchServer(name, port, dataPort string) server.Server {
	// Convert the data port to a BE uint for the redirect packet.
	parsedDataPort, _ := strconv.ParseUint(dataPort, 10, 16)
	dataRedirectPort := uint16((parsedDataPort >> 8) | (parsedDataPort << 8))

	return &PatchServer{
		name:             name,
		port:             port,
		dataRedirectPort: dataRedirectPort,
	}
}

func (s *PatchServer) Name() string       { return s.name }
func (s *PatchServer) Port() string       { return s.port }
func (s *PatchServer) HeaderSize() uint16 { return archon.PCHeaderSize }

func (s *PatchServer) AcceptClient(cs *server.ConnectionState) (server.Client2, error) {
	c := &Client{
		cs:          cs,
		clientCrypt: crypto.NewPCCrypt(),
		serverCrypt: crypto.NewPCCrypt(),
	}

	if err := SendPCWelcome(c); err != nil {
		return nil, fmt.Errorf("error sending welcome packet to %s: %s", cs.IPAddr(), err)
	}
	return c, nil
}

// send the welcome packet to a client with the copyright message and encryption vectors.
func SendPCWelcome(client *Client) error {
	pkt := archon.PatchWelcomePkt{
		Header: archon.PCHeader{Type: archon.PatchWelcomeType, Size: 0x4C},
	}
	copy(pkt.Copyright[:], copyright)
	copy(pkt.ClientVector[:], client.clientVector())
	copy(pkt.ServerVector[:], client.serverVector())

	return client.sendRaw(pkt)
}

func (s *PatchServer) Handle(client server.Client2) error {
	c := client.(*Client)
	var header archon.PCHeader

	internal.StructFromBytes(c.ConnectionState().Data()[:archon.PCHeaderSize], &header)

	var err error
	switch header.Type {
	case archon.PatchWelcomeType:
		err = s.sendWelcomeAck(c)
	case archon.PatchHandshakeType:
		if err := s.sendWelcomeMessage(c); err == nil {
			err = s.sendPatchRedirect(c)
		}
	default:
		archon.Log.Infof("Received unknown packet %2x from %s", header.Type, c.ConnectionState().IPAddr())
	}
	return err
}

func (s *PatchServer) sendWelcomeAck(client *Client) error {
	// PatchHandshakeType is treated as an ack in this case.
	return client.send(&archon.PCHeader{
		Size: 0x04,
		Type: archon.PatchHandshakeType,
	})
}

// Message displayed on the patch download screen.
func (s *PatchServer) sendWelcomeMessage(client *Client) error {
	message, size := GetWelcomeMessage()
	pkt := &archon.PatchWelcomeMessage{
		Header: archon.PCHeader{
			Size: archon.PCHeaderSize + size,
			Type: archon.PatchMessageType,
		},
		Message: message,
	}

	return client.send(pkt)
}

// send the redirect packet, providing the IP and port of the next server.
func (s *PatchServer) sendPatchRedirect(client *Client) error {
	pkt := archon.PatchRedirectPacket{
		Header:  archon.PCHeader{Type: archon.PatchRedirectType},
		IPAddr:  [4]uint8{},
		Port:    s.dataRedirectPort,
		Padding: 0,
	}

	hostnameBytes := archon.BroadcastIP()
	copy(pkt.IPAddr[:], hostnameBytes[:])

	return client.send(pkt)
}
