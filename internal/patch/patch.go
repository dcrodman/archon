package patch

import (
	"context"
	"strconv"
	"sync"

	"github.com/spf13/viper"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/packets"
)

// Convert the welcome message to UTF-16LE and cache it. PSOBB expects this prefix to the message,
//not completely sure why. Language perhaps?

var (
	messageBytes []byte
	messageInit  sync.Once

	// Copyright message expected by the client for the patch welcome.
	copyright = []byte("Patch Server. Copyright SonicTeam, LTD. 2001")
)

func getWelcomeMessage() ([]byte, uint16) {
	messageInit.Do(func() {
		messageBytes = bytes.ConvertToUtf16(viper.GetString("patch_server.welcome_message"))

		if len(messageBytes) > (1 << 16) {
			archon.Log.Warn("patch server welcome message exceeds 65,000 characters")
			messageBytes = messageBytes[:1<<16-2]
		}
		// Set the unicode byte order mark appropriately since we use LE encoding.
		messageBytes = append([]byte{0xFF, 0xFE}, messageBytes...)
	})

	return messageBytes, uint16(len(messageBytes))
}

// Server is the PATCH server implementation. It is extremely simple and for the
// most part only exists as a first point of contact for the client, its chief
// responsibility being to send clients the address of the DATA server.
type Server struct {
	name string
	// Parsed representation of the login port.
	dataRedirectPort uint16
}

func NewServer(name, dataPort string) *Server {
	// Convert the data port to a BE uint for the redirect packet.
	parsedDataPort, _ := strconv.ParseUint(dataPort, 10, 16)
	dataRedirectPort := uint16((parsedDataPort >> 8) | (parsedDataPort << 8))

	return &Server{name: name, dataRedirectPort: dataRedirectPort}
}

func (s *Server) Name() string                   { return s.name }
func (s *Server) Init(ctx context.Context) error { return nil }

func (s *Server) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewPCCryptoSession()
	c.DebugTags["server_type"] = "patch"
}

func (s *Server) Handshake(c *client.Client) error {
	// Send the welcome packet to a client with the copyright message and encryption vectors.
	pkt := packets.PatchWelcome{
		Header: packets.PCHeader{Type: packets.PatchWelcomeType, Size: 0x4C},
	}
	copy(pkt.Copyright[:], copyright)
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())

	return c.SendRaw(pkt)
}

func (s *Server) Handle(ctx context.Context, c *client.Client, data []byte) error {
	var header packets.PCHeader
	bytes.StructFromBytes(data[:packets.PCHeaderSize], &header)

	var err error
	switch header.Type {
	case packets.PatchWelcomeType:
		err = s.sendWelcomeAck(c)
	case packets.PatchHandshakeType:
		if err = s.sendWelcomeMessage(c); err == nil {
			err = s.sendPatchRedirect(c)
		}
	default:
		archon.Log.Infof("received unknown packet %2x from %s", header.Type, c.IPAddr())
	}
	return err
}

func (s *Server) sendWelcomeAck(c *client.Client) error {
	// PatchHandshakeType is treated as an ack in this case.
	return c.Send(&packets.PCHeader{
		Size: 0x04,
		Type: packets.PatchHandshakeType,
	})
}

// Message displayed on the patch download screen.
func (s *Server) sendWelcomeMessage(c *client.Client) error {
	message, size := getWelcomeMessage()
	pkt := &packets.PatchWelcomeMessage{
		Header: packets.PCHeader{
			Size: packets.PCHeaderSize + size,
			Type: packets.PatchMessageType,
		},
		Message: message,
	}

	return c.Send(pkt)
}

// send the redirect packet, providing the IP and port of the next server.
func (s *Server) sendPatchRedirect(c *client.Client) error {
	pkt := packets.PatchRedirect{
		Header:  packets.PCHeader{Type: packets.PatchRedirectType},
		IPAddr:  [4]uint8{},
		Port:    s.dataRedirectPort,
		Padding: 0,
	}

	hostnameBytes := archon.BroadcastIP()
	copy(pkt.IPAddr[:], hostnameBytes[:])

	return c.Send(pkt)
}
