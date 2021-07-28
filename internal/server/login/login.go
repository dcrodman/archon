package login

import (
	"context"
	"strconv"
	"strings"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/auth"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server/client"
	"github.com/dcrodman/archon/internal/server/internal"
)

// Copyright message expected by the client when connecting.
var loginCopyright = []byte("Phantasy Star Online Blue Burst Game Backend. Copyright 1999-2004 SONICTEAM.")

// Server is the LOGIN server implementation. Clients connect to this server
// after going through the DATA server, its main responsibility is to authenticate
// the client's username/password and set some initial state on the client before
// redirecting them to the CHARACTER server.
type Server struct {
	name                  string
	characterRedirectPort uint16
}

func NewServer(name, characterPort string) *Server {
	charPort, _ := strconv.ParseUint(characterPort, 10, 16)
	return &Server{name: name, characterRedirectPort: uint16(charPort)}
}

func (s *Server) Name() string                   { return s.name }
func (s *Server) Init(ctx context.Context) error { return nil }

func (s *Server) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
	c.DebugTags["server_type"] = "login"
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
	var header packets.BBHeader
	internal.StructFromBytes(data[:packets.BBHeaderSize], &header)

	var err error
	switch header.Type {
	case packets.LoginType:
		var loginPkt packets.Login
		internal.StructFromBytes(data, &loginPkt)
		err = s.handleLogin(c, &loginPkt)
	case packets.DisconnectType:
		// Just wait until we recv 0 from the client to disconnect.
		break
	default:
		archon.Log.Infof("Received unknown packet %x from %s", header.Type, c.IPAddr())
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

	// The first time we receive this packet the loginClientExtension will have included the
	// version string in the security data; check it.
	//if ClientVersionString != string(util.StripPadding(loginPkt.Security[:])) {
	//	SendSecurity(loginClientExtension, BBLoginErrorPatch, 0, 0)
	//	return errors.New("Incorrect version string")
	//}

	// Copy over the config, to indicate they've passed initial authentication.
	internal.StructFromBytes(loginPkt.Security[:], &c.Config)
	// Newserv sets this field when the login client first connects. I think this is
	// used to indicate that the client has made it through the LOGIN server,
	// but for now we'll just set it and leave it alone.
	c.Config.Magic = 0x48615467

	if err := s.sendSecurity(c, packets.BBLoginErrorNone); err != nil {
		return err
	}
	return s.sendCharacterRedirect(c)
}

// sendSecurity transmits initialization packet with information about the user's
// authentication status.
func (s *Server) sendSecurity(c *client.Client, errorCode uint32) error {
	// Constants set according to how Newserv does it.
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

// Sends a message to the client. In this case whatever message is sent
// here will be displayed in a dialog box after the patch screen.
func (s *Server) sendMessage(c *client.Client, message string) error {
	return c.Send(&packets.LoginClientMessage{
		Header:   packets.BBHeader{Type: packets.LoginClientMessageType},
		Language: 0x00450009,
		Message:  internal.ConvertToUtf16(message),
	})
}

// Send the IP address and port of the character server to  which the client will
// connect after disconnecting from this server.
func (s *Server) sendCharacterRedirect(c *client.Client) error {
	pkt := &packets.Redirect{
		Header: packets.BBHeader{Type: packets.RedirectType},
		IPAddr: [4]uint8{},
		Port:   s.characterRedirectPort,
	}
	ip := archon.BroadcastIP()
	copy(pkt.IPAddr[:], ip[:])

	return c.Send(pkt)
}
