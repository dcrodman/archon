package ship

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/auth"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/shipgate"
)

const (
	// Menu "prefixes" that are OR'd with the menu IDs in order to
	// distinguish between the menus from which the client is selecting.
	shipListMenuType  = 0x10000000
	blockListMenuType = 0x20000000

	// BackMenuItem is the block ID reserved for returning to the ship select menu.
	BackMenuItem = 0xFF
)

var loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

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
	Name   string
	Config *core.Config
	Logger *logrus.Logger
	Blocks []Block

	shipgateClient shipgate.Shipgate
	shipListClient *shipgate.ShipRegistrationClient
}

func (s *Server) Identifier() string {
	return s.Name
}

// Init connects the ship to the shipgate and registers so that it
// can begin receiving players.
func (s *Server) Init(ctx context.Context) error {
	s.shipgateClient = shipgate.NewRPCClient(s.Config)
	s.shipListClient = &shipgate.ShipRegistrationClient{
		Logger:         s.Logger,
		ShipgateClient: s.shipgateClient,
	}

	// Register this ship with the shipgate so that it can start accepting players.
	if _, err := s.shipgateClient.RegisterShip(ctx, &shipgate.RegistrationRequest{
		Name:    s.Config.ShipServer.Name,
		Port:    strconv.Itoa(s.Config.ShipServer.Port),
		Address: s.Config.Hostname,
	}); err != nil {
		return fmt.Errorf("error registering with shipgate: %v", err)
	}
	// Start the loop that retrieves the ship list from the shipgate.
	if err := s.shipListClient.StartShipRefreshLoop(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Server) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
	c.DebugTags["server_type"] = "ship"
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
	bytes.StructFromBytes(data[:packets.BBHeaderSize], &header)

	var err error
	switch header.Type {
	case packets.LoginType:
		var loginPkt packets.Login
		bytes.StructFromBytes(data, &loginPkt)
		err = s.handleShipLogin(ctx, c, &loginPkt)
	case packets.MenuSelectType:
		var menuSelectPkt packets.MenuSelection
		bytes.StructFromBytes(data, &menuSelectPkt)
		err = s.handleMenuSelection(c, &menuSelectPkt)
	default:
		s.Logger.Infof("received unknown packet %02x from %s", header.Type, c.IPAddr())
	}
	return err
}

func (s *Server) handleShipLogin(ctx context.Context, c *client.Client, loginPkt *packets.Login) error {
	username := string(bytes.StripPadding(loginPkt.Username[:]))
	password := string(bytes.StripPadding(loginPkt.Password[:]))

	if _, err := s.shipListClient.AuthenticateAccount(ctx, username, password); err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			return s.sendSecurity(c, packets.BBLoginErrorPassword)
		case auth.ErrAccountBanned:
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

	return s.sendBlockList(c)
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

// send the client the block list on the selection screen.
func (s *Server) sendBlockList(c *client.Client) error {
	var blocks []packets.Block
	for _, blockCfg := range s.Blocks {
		block := packets.Block{
			Unknown: 0x12,
			BlockID: blockListMenuType | uint32(blockCfg.ID),
		}
		copy(block.BlockName[:], bytes.ConvertToUtf16(blockCfg.Name))
		blocks = append(blocks, block)
	}

	// The "back" menu item for returning to the ship select screen
	// is sent to the client as another (final) block selection option.
	blocks = append(blocks, packets.Block{
		Unknown: 0x08,
		BlockID: blockListMenuType | BackMenuItem,
	})
	copy(blocks[len(blocks)-1].BlockName[:], bytes.ConvertToUtf16("Ship Selection"))

	blockListPkt := &packets.BlockList{
		Header: packets.BBHeader{
			Type:  packets.BlockListType,
			Flags: uint32(len(blocks)),
		},
		Unknown: 0x08,
		Blocks:  blocks,
	}
	copy(blockListPkt.ShipName[:], []byte(s.Config.ShipServer.Name))

	return c.Send(blockListPkt)
}

func (s *Server) handleMenuSelection(c *client.Client, pkt *packets.MenuSelection) error {
	// They can be at either the ship or block selection menu, so make sure we have the right one.
	// Note: Should probably figure out what menuSelectPkt.MenuID is for (oandif that's the right name).
	var err error
	// Case if user gets back from block selection to ship selection
	if pkt.MenuID == 1 && pkt.ItemID == 1 {
		err = s.handleShipSelection(c, pkt.ItemID-1)
		if err != nil {
			return err
		}
	}
	switch pkt.ItemID & 0xFF000000 {
	case blockListMenuType:
		err = s.handleBlockSelection(c, pkt.ItemID^blockListMenuType)
	case shipListMenuType:
		err = s.handleShipSelection(c, pkt.ItemID^shipListMenuType)
	default:
		err = fmt.Errorf("unrecognized menu ID: %v", pkt.MenuID)
	}
	return err
}

func (s *Server) handleBlockSelection(c *client.Client, selection uint32) error {
	// The player selected a block to join from the menu. Redirect them to the block's address
	// if a block was chosen or send them the ship list to take them back to the ship selection
	// meny if "Ship List" was chosen.
	if selection == BackMenuItem {
		return s.sendShipList(c)
	} else if int(selection) > len(s.Blocks) || int(selection) < 0 {
		return fmt.Errorf("error selecting block: block ID %d out of range [0, %d]", selection, len(s.Blocks))
	}

	var err error
	for _, block := range s.Blocks {
		if block.ID == int(selection) {
			err = s.sendBlockRedirect(c, block)
			break
		}
	}
	return err
}

func (s *Server) sendShipList(c *client.Client) error {
	shipList := s.shipListClient.GetConnectedShipList()

	pkt := &packets.ShipList{
		Header: packets.BBHeader{
			Type:  packets.LoginShipListType,
			Flags: uint32(len(shipList)),
		},
		Unknown:     0x20,
		Unknown2:    0xFFFFFFF4,
		Unknown3:    0x04,
		ShipEntries: shipList,
	}
	copy(pkt.ServerName[:], bytes.ConvertToUtf16("Archon"))

	return c.Send(pkt)
}

// Player selected one of the items on the ship select screen.
func (s *Server) handleShipSelection(c *client.Client, selection uint32) error {
	ip, port, err := s.shipListClient.GetSelectedShipAddress(selection)
	if err != nil {
		return fmt.Errorf("could not get selected ship: %d", selection)
	}
	return c.Send(&packets.Redirect{
		Header: packets.BBHeader{Type: packets.RedirectType},
		IPAddr: [4]uint8{ip[0], ip[1], ip[2], ip[3]},
		Port:   uint16(port),
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
