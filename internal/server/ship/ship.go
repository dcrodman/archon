// The ship server logic.
package ship

import (
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/auth"
	crypto "github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server"
	"github.com/dcrodman/archon/internal/server/block"
	"github.com/dcrodman/archon/internal/server/character"
	"github.com/dcrodman/archon/internal/server/internal"
	"strings"
)

// BackMenuItem is the block ID reserved for returning to the ship select menu.
const BackMenuItem = 0xFF

var loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

// ShipServer defines the operations for the gameplay servers.
type ShipServer struct {
	name string
	port string

	// Precomputed block packet.
	blockListPkt *packets.BlockList
}

func NewServer(name, port string, blockServers []block.BlockServer) *ShipServer {
	// The block list packet is recomputed since it's mildly expensive and
	// (at least for now) shouldn't be changing without a restart.
	blocks := make([]packets.Block, 0)
	for i, blockServer := range blockServers {
		b := packets.Block{
			Unknown: 0x12,
			BlockId: uint32(i + 1),
		}
		copy(b.BlockName[:], internal.ConvertToUtf16(blockServer.Name()))
		blocks = append(blocks, b)
	}

	// The "back" menu item for returning to the ship select screen
	// is sent to the client as another (final) block selection option.
	blocks = append(blocks, packets.Block{
		Unknown: 0x08,
		BlockId: BackMenuItem,
	})
	copy(blocks[len(blocks)-1].BlockName[:], internal.ConvertToUtf16("Ship Selection"))

	blockListPkt := &packets.BlockList{
		Header: packets.BBHeader{
			Type:  packets.BlockListType,
			Flags: uint32(len(blockServers)),
		},
		Unknown: 0x08,
		Blocks:  blocks,
	}
	copy(blockListPkt.ShipName[:], name)

	return &ShipServer{
		name:         name,
		port:         port,
		blockListPkt: blockListPkt,
	}
}

func (s *ShipServer) Name() string       { return s.name }
func (s *ShipServer) Port() string       { return s.port }
func (s *ShipServer) HeaderSize() uint16 { return packets.BBHeaderSize }

func (s *ShipServer) AcceptClient(cs *server.ConnectionState) (server.Client, error) {
	c := &Client{
		cs:          cs,
		clientCrypt: crypto.NewBBCrypt(),
		serverCrypt: crypto.NewBBCrypt(),
	}

	if err := s.SendWelcome(c); err != nil {
		return nil, fmt.Errorf("error sending welcome packet to %s: %s", cs.IPAddr(), err)
	}
	return c, nil
}

func (s *ShipServer) SendWelcome(c *Client) error {
	pkt := &packets.Welcome{
		Header:       packets.BBHeader{Type: packets.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], loginCopyright)
	copy(pkt.ServerVector[:], c.serverVector())
	copy(pkt.ClientVector[:], c.clientVector())

	return c.sendRaw(pkt)
}

func (s *ShipServer) Handle(client server.Client) error {
	c := client.(*Client)
	packetData := c.ConnectionState().Data()

	var header packets.BBHeader
	internal.StructFromBytes(packetData[:packets.BBHeaderSize], &header)

	var err error
	switch header.Type {
	case packets.LoginType:
		err = s.handleShipLogin(c)
	case packets.MenuSelectType:
		var pkt packets.MenuSelection
		internal.StructFromBytes(packetData, &pkt)
		// They can be at either the ship or block selection menu, so make sure we have the right one.
		if pkt.MenuId == character.ShipSelectionMenuId {
			// TODO: Hack for now, but this coupling on the login server logic needs to go away.
			err = s.handleShipSelection(c)
		} else {
			err = s.handleBlockSelection(c, pkt)
		}
	default:
		archon.Log.Infof("received unknown packet %02x from %s", header.Type, c.ConnectionState().IPAddr())
	}
	return err
}

func (s *ShipServer) handleShipLogin(c *Client) error {
	var loginPkt packets.Login
	internal.StructFromBytes(c.ConnectionState().Data(), &loginPkt)

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
	return s.sendBlockList(c)
}

func (s *ShipServer) sendSecurity(c *Client, errorCode uint32) error {
	return c.send(&packets.Security{
		Header:       packets.BBHeader{Type: packets.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamId:       c.TeamId,
		Config:       c.Config,
		Capabilities: 0x00000102,
	})
}

func (s *ShipServer) sendMessage(c *Client, message string) error {
	return c.send(&packets.LoginClientMessage{
		Header:   packets.BBHeader{Type: packets.LoginClientMessageType},
		Language: 0x00450009,
		Message:  internal.ConvertToUtf16(message),
	})
}

// send the client the block list on the selection screen.
func (s *ShipServer) sendBlockList(c *Client) error {
	return c.send(s.blockListPkt)
}

// Player selected one of the items on the ship select screen.
func (s *ShipServer) handleShipSelection(client *Client) error {
	//var pkt packets.MenuSelection
	//internal.StructFromBytes(client.ConnectionState().Data(), &pkt)
	//
	//selectedShip := pkt.ItemId - 1
	//
	//if selectedShip < 0 || selectedShip >= uint32(len(character.shipList)) {
	//	return errors.New("Invalid ship selection: " + string(selectedShip))
	//}
	//s := &character.shipList[selectedShip]
	//
	//return archon.SendRedirect(client, s.ipAddr[:], s.port)
	return nil
}

// Send the IP address and port of the character server to  which the client will
// connect after disconnecting from this server.
func (s *ShipServer) sendBlockRedirect(c *Client) error {
	//pkt := &packets.Redirect{
	//	Header: packets.BBHeader{Type: packets.RedirectType},
	//	IPAddr: [4]uint8{},
	//	Port:   s.charRedirectPort,
	//}
	//ip := archon.BroadcastIP()
	//copy(pkt.IPAddr[:], ip[:])
	//
	//return c.send(pkt)
	return nil
}

// The player selected a block to join from the menu.
func (s *ShipServer) handleBlockSelection(c *Client, pkt packets.MenuSelection) error {
	// Grab the chosen block and redirect them to the selected block server.
	//port, _ := strconv.ParseInt(s.port, 10, 16)
	//selectedBlock := pkt.ItemId
	//
	//if selectedBlock == BackMenuItem {
	//	s.SendShipList(sc, character.shipList)
	//} else if int(selectedBlock) > archon.Config.ShipServer.NumBlocks {
	//	return fmt.Errorf("Block selection %v out of range %v", selectedBlock, archon.Config.ShipServer.NumBlocks)
	//}
	//
	//ipAddr := archon.BroadcastIP()
	//return archon.SendRedirect(sc, ipAddr[:], uint16(uint32(port)+selectedBlock))
	return nil
}
