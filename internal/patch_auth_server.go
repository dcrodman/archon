package internal

import (
	"context"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/encryption"
)

// PatchCopyright is the message expected by the client for the patch welcome.
var PatchCopyright = []byte("Patch Server. Copyright SonicTeam, LTD. 2001")

// PatchAuthServer exists as a first point of contact for connecting clients. We don't
// care about actually enforcing auth here, so its chief responsibility being to send
// clients the address of the [PatchServer]. Other server implementations may refer to
// as the "PATCH" server.
type PatchAuthServer struct {
	welcomeMessage []byte
}

func (s *PatchAuthServer) Identifier() string {
	return "PATCH:AUTH"
}

func (s *PatchAuthServer) Init(ctx context.Context) error {
	s.welcomeMessage = ConvertToUtf16(Config.PatchServer.WelcomeMessage)

	if len(s.welcomeMessage) > (1 << 16) {
		Logger.Warn("patch server welcome message exceeds 65,000 characters")
		s.welcomeMessage = s.welcomeMessage[:1<<16-2]
	}
	// Set the unicode byte order mark appropriately since we use LE encoding.
	s.welcomeMessage = append([]byte{0xFF, 0xFE}, s.welcomeMessage...)

	return nil
}

func (s *PatchAuthServer) Handshake(c *Client) error {
	// Initialize new encryption vectors for this session.
	c.CryptoSession = encryption.NewPCCryptoSession()

	// Send the welcome command to a client with the copyright message and encryption vectors.
	pkt := commands.PatchWelcome{
		Header: commands.PCHeader{Type: commands.PatchWelcomeType, Size: 0x4C},
	}
	copy(pkt.Copyright[:], PatchCopyright)
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())

	return c.SendRaw(pkt)
}

func (s *PatchAuthServer) Handle(ctx context.Context, c *Client, data []byte) error {
	var header commands.PCHeader
	UnmarshalStruct(data[:commands.PCHeaderSize], &header)

	var err error
	switch header.Type {
	case commands.PatchWelcomeType:
		err = SendPatchWelcomeAck(c)
	case commands.PatchHandshakeType:
		err = SendPatchWelcomeMessage(c, s.welcomeMessage)
		if err == nil {
			err = SendPatchRedirect(c)
		}
	default:
		Logger.Infof("received unknown command %2x from %s", header.Type, c.IPAddr)
	}
	return err
}

func SendPatchWelcomeAck(c *Client) error {
	// PatchHandshakeType is treated as an ack in this case.
	return c.Send(&commands.PCHeader{
		Size: 0x04,
		Type: commands.PatchHandshakeType,
	})
}

func SendPatchWelcomeMessage(c *Client, m []byte) error {
	pkt := &commands.PatchWelcomeMessage{
		Header: commands.PCHeader{
			Size: commands.PCHeaderSize + uint16(len(m)),
			Type: commands.PatchMessageType,
		},
		Message: m,
	}

	return c.Send(pkt)
}

func SendPatchRedirect(c *Client) error {
	pkt := commands.PatchRedirect{
		Header: commands.PCHeader{Type: commands.PatchRedirectType},
		IPAddr: [4]uint8{},
		// Convert the data port to a BE uint for the redirect command.
		Port:    uint16((Config.PatchServer.DataPort >> 8) | (Config.PatchServer.DataPort << 8)),
		Padding: 0,
	}

	hostnameBytes := Config.BroadcastIP()
	copy(pkt.IPAddr[:], hostnameBytes[:])

	return c.Send(pkt)
}
