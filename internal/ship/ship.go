package ship

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/core/debug"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/shipgate"
)

var copyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

type Block struct {
	Name    string
	Address string
	ID      int
}

// Server is the SHIP server implementation. This is similar to PATCH and LOGIN
// in that it really just exists to be a gateway. Is main responsibility is to
// provide the client with the block list and then send the address of the
// block that the user selects.
type Server struct {
	Config *core.Config
	Logger *zap.SugaredLogger
	Blocks []Block

	shipgateClient shipgate.Shipgate
}

func (s *Server) Identifier() string {
	return "SHIP"
}

// Init connects the ship to the shipgate and registers so that it
// can begin receiving players.
func (s *Server) Init(ctx context.Context) error {
	s.shipgateClient = shipgate.NewRPCClient(s.Config)

	// Register this ship with the shipgate so that it can start accepting players.
	if _, err := s.shipgateClient.RegisterShip(ctx, &shipgate.RegisterShipRequest{
		Name:    s.Config.ShipServer.Name,
		Address: s.Config.ExternalIP,
		Port:    strconv.Itoa(s.Config.ShipServer.Port),
	}); err != nil {
		return fmt.Errorf("error registering with shipgate: %v", err)
	}
	return nil
}

func (s *Server) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
	c.DebugTags[debug.SERVER_TYPE] = debug.SHIP_SERVER
}

func (s *Server) Handshake(c *client.Client) error {
	pkt := &packets.Welcome{
		Header:       packets.BBHeader{Type: packets.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], copyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(pkt)
}

func (s *Server) Handle(ctx context.Context, c *client.Client, data []byte) error {
	var header packets.BBHeader
	bytes.StructFromBytes(data[:packets.BBHeaderSize], &header)

	var err error
	switch header.Type {
	case packets.LoginType:
		var loginPkt packets.Login
		bytes.StructFromBytes(data, &loginPkt)
		err = s.handleShipLogin(ctx, c, &loginPkt)
	default:
		s.Logger.Infof("received unknown packet %02x from %s", header.Type, c.IPAddr())
	}
	return err
}

func (s *Server) handleShipLogin(ctx context.Context, c *client.Client, loginPkt *packets.Login) error {
	username := string(bytes.StripPadding(loginPkt.Username[:]))
	password := string(bytes.StripPadding(loginPkt.Password[:]))

	_, err := s.shipgateClient.AuthenticateAccount(ctx, &shipgate.AuthenticateAccountRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		switch err {
		case shipgate.ErrInvalidCredentials:
			return s.sendSecurity(c, packets.BBLoginErrorPassword)
		case shipgate.ErrAccountBanned:
			return s.sendSecurity(c, packets.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, cases.Title(language.English).String(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}

	if err := s.sendSecurity(c, packets.BBLoginErrorNone); err != nil {
		return err
	}
	// Tethealla sends

	// s.sendBlockList(c)
	return s.sendBlockRedirect(c, s.Blocks[0])
}

func (s *Server) sendSecurity(c *client.Client, errorCode uint32) error {
	cfg := packets.ClientConfig{
		Magic:        c.Config.Magic,
		CharSelected: c.Config.CharSelected,
		SlotNum:      c.Config.SlotNum,
		Flags:        c.Config.Flags,
	}
	copy(cfg.Ports[:], c.Config.Ports[:])
	copy(cfg.Unused[:], c.Config.Unused[:])
	copy(cfg.Unused2[:], c.Config.Unused2[:])

	return c.Send(&packets.Security{
		Header:       packets.BBHeader{Type: packets.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamID:       c.TeamID,
		Config:       cfg,
		Capabilities: 0x00000102,
	})
}

func (s *Server) sendMessage(c *client.Client, message string) error {
	return c.Send(&packets.LoginClientMessage{
		Header:   packets.BBHeader{Type: packets.LoginClientMessageType},
		Language: 0x00450009,
		Message:  bytes.ConvertToUtf16(message),
	})
}

// Send the IP address and port of the character server to  which the client will
// connect after disconnecting from this server.
func (s *Server) sendBlockRedirect(c *client.Client, block Block) error {
	addressParts := strings.Split(block.Address, ":")
	blockIP := net.ParseIP(addressParts[0]).To4()
	port, err := strconv.Atoi(addressParts[1])
	if err != nil {
		return fmt.Errorf("error parsing port from block address: %v", block.Address)
	}

	pkt := &packets.Redirect{
		Header: packets.BBHeader{Type: packets.RedirectType},
		Port:   uint16(port),
	}
	copy(pkt.IPAddr[:], blockIP)
	return c.Send(pkt)
}
