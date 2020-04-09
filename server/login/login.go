package login

import (
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/auth"
	crypto "github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/server/internal"
	"strconv"
	"strings"
)

// Copyright message expected by the client when connecting.
var loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

// LoginServer is the client's first point of contact with the account/character
// part of the flow. Its main purpose is to authenticate the client and send them
// on to the CHARACTER server.
type LoginServer struct {
	name             string
	port             string
	charRedirectPort uint16
}

func NewServer(name, port, characterPort string) server.Server {
	charPort, _ := strconv.ParseUint(characterPort, 10, 16)
	return &LoginServer{name: name, port: port, charRedirectPort: uint16(charPort)}
}

func (s *LoginServer) Name() string       { return s.name }
func (s *LoginServer) Port() string       { return s.port }
func (s *LoginServer) HeaderSize() uint16 { return archon.BBHeaderSize }

func (s *LoginServer) AcceptClient(cs *server.ConnectionState) (server.Client2, error) {
	c := &Client{
		cs:          cs,
		serverCrypt: crypto.NewBBCrypt(),
		clientCrypt: crypto.NewBBCrypt(),
	}

	if err := s.SendWelcome(c); err != nil {
		return nil, fmt.Errorf("error sending welcome packet to %s: %s", cs.IPAddr(), err)
	}
	return c, nil
}

func (s *LoginServer) SendWelcome(c *Client) error {
	pkt := &archon.WelcomePkt{
		Header:       archon.BBHeader{Type: archon.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], loginCopyright)
	copy(pkt.ServerVector[:], c.serverVector())
	copy(pkt.ClientVector[:], c.clientVector())

	return c.sendRaw(pkt)
}

func (s *LoginServer) Handle(client server.Client2) error {
	c := client.(*Client)
	var hdr archon.BBHeader
	internal.StructFromBytes(c.ConnectionState().Data()[:archon.BBHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case archon.LoginType:
		err = s.handleLogin(c)
	case archon.DisconnectType:
		// Just wait until we recv 0 from the client to disconnect.
		break
	default:
		archon.Log.Infof("Received unknown packet %x from %s", hdr.Type, c.ConnectionState().IPAddr())
	}

	return err
}

func (s *LoginServer) handleLogin(c *Client) error {
	var loginPkt archon.LoginPkt
	internal.StructFromBytes(c.ConnectionState().Data(), &loginPkt)

	username := string(internal.StripPadding(loginPkt.Username[:]))
	password := string(internal.StripPadding(loginPkt.Password[:]))

	if _, err := auth.VerifyAccount(username, password); err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			return s.sendSecurity(c, archon.BBLoginErrorPassword)
		case auth.ErrAccountBanned:
			return s.sendSecurity(c, archon.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, strings.Title(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}

	// The first time we receive this packet the client will have included the
	// version string in the security data; check it.
	//if ClientVersionString != string(util.StripPadding(loginPkt.Security[:])) {
	//	SendSecurity(client, BBLoginErrorPatch, 0, 0)
	//	return errors.New("Incorrect version string")
	//}

	// Copy over the config, to indicate they've passed initial authentication.
	internal.StructFromBytes(loginPkt.Security[:], &c.Config)
	// Newserv sets this field when the client first connects. I think this is
	// used to indicate that the client has made it through the LOGIN server,
	// but for now we'll just set it and leave it alone.
	c.Config.Magic = 0x48615467

	if err := s.sendSecurity(c, archon.BBLoginErrorNone); err != nil {
		return err
	}
	return s.sendCharacterRedirect(c)
}

// sendSecurity transmits initialization packet with information about the user's
// authentication status.
func (s *LoginServer) sendSecurity(c *Client, errorCode uint32) error {
	// Constants set according to how Newserv does it.
	return c.send(&archon.SecurityPacket{
		Header:       archon.BBHeader{Type: archon.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamId:       c.TeamId,
		Config:       &c.Config,
		Capabilities: 0x00000102,
	})
}

// Sends a message to the client. In this case whatever message is sent
// here will be displayed in a dialog box after the patch screen.
func (s *LoginServer) sendMessage(c *Client, message string) error {
	return c.send(&archon.LoginClientMessagePacket{
		Header:   archon.BBHeader{Type: archon.LoginClientMessageType},
		Language: 0x00450009,
		Message:  internal.ConvertToUtf16(message),
	})
}

// Send the IP address and port of the character server to  which the client will
// connect after disconnecting from this server.
func (s *LoginServer) sendCharacterRedirect(c *Client) error {
	pkt := &archon.RedirectPacket{
		Header: archon.BBHeader{Type: archon.RedirectType},
		IPAddr: [4]uint8{},
		Port:   s.charRedirectPort,
	}
	ip := archon.BroadcastIP()
	copy(pkt.IPAddr[:], ip[:])

	return c.send(pkt)
}
