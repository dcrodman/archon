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

// Copyright message expected by the client for the patch welcome.
var copyright = []byte("Patch Server. Copyright SonicTeam, LTD. 2001")

// PatchAuthServer exists as a first point of contact for connecting clients. We don't
// care about actually enforcing auth here, so its chief responsibility being to send
// clients the address of the [PatchDataServer]. Other server implementations may refer to
// as the "PATCH" server.
type PatchAuthServer struct {
	Config *core.Config
	Logger *zap.SugaredLogger

	welcomeMessage []byte
}

func (s *PatchAuthServer) Identifier() string {
	return "PATCH:AUTH"
}

func (s *PatchAuthServer) Init(ctx context.Context) error {
	s.welcomeMessage = bytes.ConvertToUtf16(s.Config.PatchServer.WelcomeMessage)

	if len(s.welcomeMessage) > (1 << 16) {
		s.Logger.Warn("patch server welcome message exceeds 65,000 characters")
		s.welcomeMessage = s.welcomeMessage[:1<<16-2]
	}
	// Set the unicode byte order mark appropriately since we use LE encoding.
	s.welcomeMessage = append([]byte{0xFF, 0xFE}, s.welcomeMessage...)

	return nil
}

func (s *PatchAuthServer) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewPCCryptoSession()
	c.DebugTags[debug.SERVER_TYPE] = debug.PATCH_SERVER
}

func (s *PatchAuthServer) Handshake(c *client.Client) error {
	// Send the welcome packet to a client with the copyright message and encryption vectors.
	pkt := packets.PatchWelcome{
		Header: packets.PCHeader{Type: packets.PatchWelcomeType, Size: 0x4C},
	}
	copy(pkt.Copyright[:], copyright)
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())

	return c.SendRaw(pkt)
}

func (s *PatchAuthServer) Handle(ctx context.Context, c *client.Client, data []byte) error {
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

func (s *PatchAuthServer) sendWelcomeAck(c *client.Client) error {
	// PatchHandshakeType is treated as an ack in this case.
	return c.Send(&packets.PCHeader{
		Size: 0x04,
		Type: packets.PatchHandshakeType,
	})
}

// Message displayed on the patch download screen.
func (s *PatchAuthServer) sendWelcomeMessage(c *client.Client) error {
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
func (s *PatchAuthServer) sendPatchRedirect(c *client.Client) error {
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
