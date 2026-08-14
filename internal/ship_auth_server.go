package internal

import (
	"context"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/shipgate"
)

var GameCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

// GameAuthServer is the SHIP server implementation. This is similar to PATCH and LOGIN
// in that it really just exists to be a gateway. Is main responsibility is to
// provide the client with the block list and then send the address of the
// block that the user selects.
type GameAuthServer struct {
}

func (s *GameAuthServer) Identifier() string {
	return "SHIP:AUTH"
}

// Init connects the ship to the shipgate and registers so that it
// can begin receiving players.
func (s *GameAuthServer) Init(ctx context.Context) error {
	// Register this ship with the shipgate so that it can start accepting players.
	shipgate.Shipgate.RegisterShip(ctx, shipgate.Ship{
		Name:    Config.ShipServer.Name,
		Address: Config.ExternalIP,
		Port:    Config.ShipServer.AuthPort,
	})
	return nil
}

func (s *GameAuthServer) Handshake(ctx context.Context, c *Client) error {
	c.CryptoSession = encryption.NewBlueBurstCryptoSession()

	pkt := &commands.Welcome{
		Header:       commands.BBHeader{Type: commands.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], GameCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(ctx, pkt)
}

func (s *GameAuthServer) Handle(ctx context.Context, c *Client, data []byte) error {
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

func (s *GameAuthServer) handleShipLogin(ctx context.Context, c *Client, loginPkt *commands.Login) error {
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

	return s.sendGameServerRedirect(ctx, c)
}

// Send the IP address and port of the character server to  which the client will
// connect after disconnecting from this server.
func (s *GameAuthServer) sendGameServerRedirect(ctx context.Context, c *Client) error {
	pkt := &commands.Redirect{
		Header: commands.BBHeader{Type: commands.RedirectType},
		Port:   uint16(Config.ShipServer.GamePort),
	}
	ip := Config.BroadcastIP()
	copy(pkt.IPAddr[:], ip[:])
	return c.Send(ctx, pkt)
}
