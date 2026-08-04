package internal

import (
	"context"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/shipgate"
)

var GameCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

// AuthServer is the SHIP server implementation. This is similar to PATCH and LOGIN
// in that it really just exists to be a gateway. Is main responsibility is to
// provide the client with the block list and then send the address of the
// block that the user selects.
type AuthServer struct {
}

func (s *AuthServer) Identifier() string {
	return "SHIP:AUTH"
}

// Init connects the ship to the shipgate and registers so that it
// can begin receiving players.
func (s *AuthServer) Init(ctx context.Context) error {
	// Register this ship with the shipgate so that it can start accepting players.
	shipgate.Shipgate.RegisterShip(ctx, data.Ship{
		Name:    Config.ShipServer.Name,
		Address: Config.ExternalIP,
		Port:    Config.ShipServer.AuthPort,
	})
	return nil
}

func (s *AuthServer) SetUpClient(c *Client) {
	c.CryptoSession = encryption.NewBlueBurstCryptoSession()
}

func (s *AuthServer) Handshake(c *Client) error {
	pkt := &commands.Welcome{
		Header:       commands.BBHeader{Type: commands.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], GameCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(pkt)
}

func (s *AuthServer) Handle(ctx context.Context, c *Client, data []byte) error {
	var header commands.BBHeader
	UnmarshalStruct(data[:commands.BBHeaderSize], &header)

	var err error
	switch header.Type {
	case commands.LoginType:
		var loginPkt commands.Login
		UnmarshalStruct(data, &loginPkt)
		err = s.handleShipLogin(ctx, c, &loginPkt)
	default:
		Logger.Infof("received unknown command %02x from %s", header.Type, c.IPAddr)
	}
	return err
}

func (s *AuthServer) handleShipLogin(ctx context.Context, c *Client, loginPkt *commands.Login) error {
	username := string(StripPadding(loginPkt.Username[:]))
	password := string(StripPadding(loginPkt.Password[:]))

	_, err := shipgate.Shipgate.AuthenticateAccount(ctx, username, password)
	if err != nil {
		switch err {
		case shipgate.ErrInvalidCredentials:
			return SendSecurity(c, commands.BBLoginErrorPassword)
		case shipgate.ErrAccountBanned:
			return SendSecurity(c, commands.BBLoginErrorBanned)
		default:
			sendErr := SendMessage(c, cases.Title(language.English).String(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}

	if err := SendSecurity(c, commands.BBLoginErrorNone); err != nil {
		return err
	}

	return s.sendGameServerRedirect(c)
}

// Send the IP address and port of the character server to  which the client will
// connect after disconnecting from this server.
func (s *AuthServer) sendGameServerRedirect(c *Client) error {
	pkt := &commands.Redirect{
		Header: commands.BBHeader{Type: commands.RedirectType},
		Port:   uint16(Config.ShipServer.GamePort),
	}
	ip := Config.BroadcastIP()
	copy(pkt.IPAddr[:], ip[:])
	return c.Send(pkt)
}
