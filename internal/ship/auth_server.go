package ship

import (
	"context"
	"fmt"
	"strconv"

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

var copyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

type Block struct {
	Name    string
	Address string
	ID      int
}

// AuthServer is the SHIP server implementation. This is similar to PATCH and LOGIN
// in that it really just exists to be a gateway. Is main responsibility is to
// provide the client with the block list and then send the address of the
// block that the user selects.
type AuthServer struct {
	Config *core.Config
	Logger *zap.SugaredLogger

	shipgateClient shipgate.Shipgate
}

func (s *AuthServer) Identifier() string {
	return "SHIP:AUTH"
}

// Init connects the ship to the shipgate and registers so that it
// can begin receiving players.
func (s *AuthServer) Init(ctx context.Context) error {
	s.shipgateClient = shipgate.NewClient(s.Config)

	// Register this ship with the shipgate so that it can start accepting players.
	if _, err := s.shipgateClient.RegisterShip(ctx, &shipgate.RegisterShipRequest{
		Name:    s.Config.ShipServer.Name,
		Address: s.Config.ExternalIP,
		Port:    strconv.Itoa(s.Config.ShipServer.GamePort),
	}); err != nil {
		return fmt.Errorf("error registering with shipgate: %v", err)
	}
	return nil
}

func (s *AuthServer) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
	c.DebugTags[debug.SERVER_TYPE] = debug.SHIP_SERVER
}

func (s *AuthServer) Handshake(c *client.Client) error {
	pkt := &commands.Welcome{
		Header:       commands.BBHeader{Type: commands.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], copyright)
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
		err = s.handleShipLogin(ctx, c, &loginPkt)
	default:
		s.Logger.Infof("received unknown command %02x from %s", header.Type, c.IPAddr())
	}
	return err
}

func (s *AuthServer) handleShipLogin(ctx context.Context, c *client.Client, loginPkt *commands.Login) error {
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

	return s.sendGameServerRedirect(c)
}

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

func (s *AuthServer) sendMessage(c *client.Client, message string) error {
	return c.Send(&commands.LoginClientMessage{
		Header:   commands.BBHeader{Type: commands.LoginClientMessageType},
		Language: 0x00450009,
		Message:  bytes.ConvertToUtf16(message),
	})
}

// Send the IP address and port of the character server to  which the client will
// connect after disconnecting from this server.
func (s *AuthServer) sendGameServerRedirect(c *client.Client) error {
	pkt := &commands.Redirect{
		Header: commands.BBHeader{Type: commands.RedirectType},
		Port:   uint16(s.Config.ShipServer.GamePort),
	}
	ip := s.Config.BroadcastIP()
	copy(pkt.IPAddr[:], ip[:])
	return c.Send(pkt)
}
