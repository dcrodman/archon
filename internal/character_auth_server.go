package internal

import (
	"context"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/shipgate"
)

// Copyright message expected by the client when connecting.
var authCopyright = []byte("Phantasy Star Online Blue Burst Game Backend. Copyright 1999-2004 SONICTEAM.")

// CharacterAuthServer is the first connection point of the second "phase" for a
// Its main responsibility is to authenticate the client's username/password and
// set some initial state on the client before redirecting them to [SERVER]. Other
// server implementations might call this the "LOGIN" server.
type CharacterAuthServer struct{}

func (s *CharacterAuthServer) Identifier() string {
	return "CHARACTER:AUTH"
}

func (s *CharacterAuthServer) Init(_ context.Context) error {
	return nil
}

func (s *CharacterAuthServer) Handshake(ctx context.Context, c *Client) error {
	c.CryptoSession = encryption.NewBlueBurstCryptoSession()

	pkt := &commands.Welcome{
		Header:       commands.BBHeader{Type: commands.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], authCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(ctx, pkt)
}

func (s *CharacterAuthServer) Handle(ctx context.Context, c *Client, data []byte) error {
	var header commands.BBHeader
	UnmarshalStruct(data[:commands.BBHeaderSize], &header)

	var err error
	switch header.Type {
	case commands.LoginType:
		var loginPkt commands.Login
		UnmarshalStruct(data, &loginPkt)
		err = s.handleLogin(ctx, c, &loginPkt)
	case commands.DisconnectType:
		// Just wait until we recv 0 from the client to disconnect.
		break
	default:
		Logger.Infof("received unknown command %x from %s", header.Type, c.IPAddr)
	}

	return err
}

func (s *CharacterAuthServer) handleLogin(ctx context.Context, c *Client, loginPkt *commands.Login) error {
	username := string(StripPadding(loginPkt.Username[:]))
	password := string(StripPadding(loginPkt.Password[:]))

	account, err := shipgate.Shipgate.AuthenticateAccount(ctx, username, password)
	if err != nil {
		switch err {
		case shipgate.ErrInvalidCredentials:
			return SendSecurity(ctx, c, commands.BBLoginErrorPassword)
		case shipgate.ErrAccountBanned:
			return SendSecurity(ctx, c, commands.BBLoginErrorBanned)
		default:
			sendErr := SendMessage(ctx, c, cases.Title(language.English).String(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}
	c.Account = account

	if err := SendSecurity(ctx, c, commands.BBLoginErrorNone); err != nil {
		return err
	}
	// The first time we receive this command the loginClientExtension will have included the
	// version string in the security data; check it.
	//if ClientVersionString != string(util.StripPadding(loginPkt.Security[:])) {
	//	SendSecurity(loginClientExtension, BBLoginErrorPatch, 0, 0)
	//	return errors.New("Incorrect version string")
	//}

	// Copy over the config, to indicate they've passed initial authentication.
	UnmarshalStruct(loginPkt.Security[:], &c.Config)
	// Newserv sets this field when the login client first connects. I think this is
	// used to indicate that the client has made it through the LOGIN server,
	// but for now we'll just set it and leave it alone.
	c.Config.Magic = 0x48615467

	// Send the IP address and port of the character server to which the client will
	// connect after disconnecting from this server.
	return SendRedirect(ctx, c, Config.BroadcastIP(), uint16(Config.CharacterServer.DataPort))
}

// send the security initialization command with information about the user's
// authentication status. Requires that c.Account is set before calling.
func SendSecurity(ctx context.Context, c *Client, errorCode uint32) error {
	cfg := commands.ClientConfig{
		Magic:        c.Config.Magic,
		CharSelected: c.Config.CharSelected,
		SlotNum:      c.Config.SlotNum,
		Flags:        c.Config.Flags,
	}
	copy(cfg.Ports[:], c.Config.Ports[:])
	copy(cfg.Unused[:], c.Config.Unused[:])
	copy(cfg.Unused2[:], c.Config.Unused2[:])

	cmd := &commands.Security{
		Header:       commands.BBHeader{Type: commands.SecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Config:       cfg,
		Capabilities: 0x00000102,
	}
	if c.Account != nil {
		cmd.Guildcard = uint32(c.Account.Guildcard)
		cmd.TeamID = uint32(c.Account.TeamID)
	}

	// Constants set according to how Newserv does it.
	return c.Send(ctx, cmd)
}

func SendRedirect(ctx context.Context, c *Client, ip [4]uint8, port uint16) error {
	cmd := &commands.Redirect{
		Header: commands.BBHeader{Type: commands.RedirectType},
		IPAddr: ip,
		Port:   port,
	}
	return c.Send(ctx, cmd)
}
