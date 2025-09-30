package character

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/core/debug"
	"github.com/dcrodman/archon/internal/shipgate"
)

// Copyright message expected by the client when connecting.
var authCopyright = []byte("Phantasy Star Online Blue Burst Game Backend. Copyright 1999-2004 SONICTEAM.")

// AuthServer is the first connection point of the second "phase" for a client.
// Its main responsibility is to authenticate the client's username/password and
// set some initial state on the client before redirecting them to [SERVER]. Other
// server implementations might call this the "LOGIN" server.
type AuthServer struct {
	Config *core.Config
	Logger *zap.SugaredLogger

	shipgateClient shipgate.Shipgate
}

func (s *AuthServer) Identifier() string {
	return "CHARACTER:AUTH"
}

func (s *AuthServer) Init(_ context.Context) error {
	s.shipgateClient = shipgate.NewClient(s.Config)
	return nil
}

func (s *AuthServer) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
	c.DebugTags[debug.SERVER_TYPE] = debug.LOGIN_SERVER
}

func (s *AuthServer) Handshake(c *client.Client) error {
	pkt := &commands.Welcome{
		Header:       commands.BBHeader{Type: commands.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], authCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(pkt)
}

func (s *AuthServer) Handle(ctx context.Context, c *client.Client, data []byte) error {
	var header commands.BBHeader
	bytes.StructFromBytes(data[:commands.BBHeaderSize], &header)

	var err error
	switch header.Type {
	case commands.LoginType:
		var loginPkt commands.Login
		bytes.StructFromBytes(data, &loginPkt)
		err = s.handleLogin(ctx, c, &loginPkt)
	case commands.DisconnectType:
		// Just wait until we recv 0 from the client to disconnect.
		break
	default:
		s.Logger.Infof("received unknown command %x from %s", header.Type, c.IPAddr())
	}

	return err
}

func (s *AuthServer) handleLogin(ctx context.Context, c *client.Client, loginPkt *commands.Login) error {
	username := string(bytes.StripPadding(loginPkt.Username[:]))
	password := string(bytes.StripPadding(loginPkt.Password[:]))

	_, err := s.shipgateClient.AuthenticateAccount(ctx, &shipgate.AuthenticateAccountRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		switch err {
		case shipgate.ErrInvalidCredentials:
			return s.sendSecurity(c, commands.BBLoginErrorPassword)
		case shipgate.ErrAccountBanned:
			return s.sendSecurity(c, commands.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, cases.Title(language.English).String(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}

	if err := s.sendSecurity(c, commands.BBLoginErrorNone); err != nil {
		return err
	}
	// The first time we receive this command the loginClientExtension will have included the
	// version string in the security data; check it.
	//if ClientVersionString != string(util.StripPadding(loginPkt.Security[:])) {
	//	SendSecurity(loginClientExtension, BBLoginErrorPatch, 0, 0)
	//	return errors.New("Incorrect version string")
	//}

	// Copy over the config, to indicate they've passed initial authentication.
	bytes.StructFromBytes(loginPkt.Security[:], &c.Config)
	// Newserv sets this field when the login client first connects. I think this is
	// used to indicate that the client has made it through the LOGIN server,
	// but for now we'll just set it and leave it alone.
	c.Config.Magic = 0x48615467

	return s.sendDataServerRedirect(c)
}

// send the security initialization command with information about the user's
// authentication status.
func (s *AuthServer) sendSecurity(c *client.Client, errorCode uint32) error {
	cfg := commands.ClientConfig{
		Magic:        c.Config.Magic,
		CharSelected: c.Config.CharSelected,
		SlotNum:      c.Config.SlotNum,
		Flags:        c.Config.Flags,
	}
	copy(cfg.Ports[:], c.Config.Ports[:])
	copy(cfg.Unused[:], c.Config.Unused[:])
	copy(cfg.Unused2[:], c.Config.Unused2[:])

	// Constants set according to how Newserv does it.
	return c.Send(&commands.Security{
		Header:       commands.BBHeader{Type: commands.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamID:       c.TeamID,
		Config:       cfg,
		Capabilities: 0x00000102,
	})
}

// Sends a message to the client. In this case whatever message is sent
// here will be displayed in a dialog box after the patch screen.
func (s *AuthServer) sendMessage(c *client.Client, message string) error {
	return c.Send(&commands.LoginClientMessage{
		Header:   commands.BBHeader{Type: commands.LoginClientMessageType},
		Language: 0x00450009,
		Message:  bytes.ConvertToUtf16(message),
	})
}

// Send the IP address and port of the character server to which the client will
// connect after disconnecting from this server.
func (s *AuthServer) sendDataServerRedirect(c *client.Client) error {
	pkt := &commands.Redirect{
		Header: commands.BBHeader{Type: commands.RedirectType},
		IPAddr: [4]uint8{},
		Port:   uint16(s.Config.CharacterServer.DataPort),
	}
	ip := s.Config.BroadcastIP()
	copy(pkt.IPAddr[:], ip[:])

	return c.Send(pkt)
}
