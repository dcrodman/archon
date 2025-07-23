package patch

import (
	"context"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/core/debug"
	"github.com/dcrodman/archon/internal/packets"
	"go.uber.org/zap"
)

// Convert the welcome message to UTF-16LE and cache it. PSOBB expects this prefix to the message,
//not completely sure why. Language perhaps?

// Copyright message expected by the client for the patch welcome.
var copyright = []byte("Patch Server. Copyright SonicTeam, LTD. 2001")

// Server is the PATCH server implementation. It is extremely simple and for the
// most part only exists as a first point of contact for the client, its chief
// responsibility being to send clients the address of the DATA server.
type Server struct {
	Name   string
	Config *core.Config
	Logger *zap.SugaredLogger

	welcomeMessage []byte
}

func (s *Server) Identifier() string {
	return s.Name
}

func (s *Server) Init(ctx context.Context) error {
	s.welcomeMessage = bytes.ConvertToUtf16(s.Config.PatchServer.WelcomeMessage)

	if len(s.welcomeMessage) > (1 << 16) {
		s.Logger.Warn("patch server welcome message exceeds 65,000 characters")
		s.welcomeMessage = s.welcomeMessage[:1<<16-2]
	}
	// Set the unicode byte order mark appropriately since we use LE encoding.
	s.welcomeMessage = append([]byte{0xFF, 0xFE}, s.welcomeMessage...)

	return nil
}

func (s *Server) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewPCCryptoSession()
	c.DebugTags[debug.SERVER_TYPE] = debug.PATCH_SERVER
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
		s.Logger.Infof("received unknown packet %2x from %s", header.Type, c.IPAddr())
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
	pkt := &packets.PatchWelcomeMessage{
		Header: packets.PCHeader{
			Size: packets.PCHeaderSize + uint16(len(s.welcomeMessage)),
			Type: packets.PatchMessageType,
		},
		Message: s.welcomeMessage,
	}

	return c.Send(pkt)
}

// send the redirect packet, providing the IP and port of the next server.
func (s *Server) sendPatchRedirect(c *client.Client) error {
	pkt := packets.PatchRedirect{
		Header: packets.PCHeader{Type: packets.PatchRedirectType},
		IPAddr: [4]uint8{},
		// Convert the data port to a BE uint for the redirect packet.
		Port:    uint16((s.Config.PatchServer.DataPort >> 8) | (s.Config.PatchServer.DataPort << 8)),
		Padding: 0,
	}

	hostnameBytes := s.Config.BroadcastIP()
	copy(pkt.IPAddr[:], hostnameBytes[:])

	return c.Send(pkt)
}
