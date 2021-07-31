package block

import (
	"context"
	"strings"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/auth"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server/client"
	"github.com/dcrodman/archon/internal/server/internal"
)

var loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

type Server struct {
	name string
}

func NewServer(name string) *Server {
	return &Server{
		name: name,
	}
}

func (s *Server) Name() string {
	return s.name
}

func (s *Server) Init(ctx context.Context) error {
	return nil
}

func (s *Server) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
	c.DebugTags["server_type"] = "block"
}

func (s *Server) Handshake(c *client.Client) error {
	pkt := &packets.Welcome{
		Header:       packets.BBHeader{Type: packets.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], loginCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(pkt)
}

func (s *Server) Handle(ctx context.Context, c *client.Client, data []byte) error {
	var packetHeader packets.BBHeader
	internal.StructFromBytes(data[:packets.BBHeaderSize], &packetHeader)

	var err error
	switch packetHeader.Type {
	case packets.LoginType:
		var loginPkt packets.Login
		internal.StructFromBytes(data, &loginPkt)
		err = s.handleLogin(c, &loginPkt)
	default:
		archon.Log.Infof("received unknown packet %x from %s", packetHeader.Type, c.IPAddr())
	}
	return err
}

func (s *Server) handleLogin(c *client.Client, loginPkt *packets.Login) error {
	username := string(internal.StripPadding(loginPkt.Username[:]))
	password := string(internal.StripPadding(loginPkt.Password[:]))

	if _, err := auth.VerifyAccount(username, password); err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			return s.sendSecurity(c, packets.BBLoginErrorPassword)
		case auth.ErrAccountBanned:
			return s.sendSecurity(c, packets.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, strings.Title(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}

	if err := s.sendSecurity(c, packets.BBLoginErrorNone); err != nil {
		return err
	}

	// TODO: Packets 0x83 and 0xE7
	return nil
}

func (s *Server) sendSecurity(c *client.Client, errorCode uint32) error {
	return c.Send(&packets.Security{
		Header:       packets.BBHeader{Type: packets.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamID:       c.TeamID,
		Config:       c.Config,
		Capabilities: 0x00000102,
	})
}

func (s *Server) sendMessage(c *client.Client, message string) error {
	return c.Send(&packets.LoginClientMessage{
		Header:   packets.BBHeader{Type: packets.LoginClientMessageType},
		Language: 0x00450009,
		Message:  internal.ConvertToUtf16(message),
	})
}
