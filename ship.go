// The BLOCK and SHIP server logic.
package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	crypto "github.com/dcrodman/archon/encryption"
	"github.com/dcrodman/archon/util"
)

// BackMenuItem is the block ID reserved for returning to the ship select menu.
const BackMenuItem = 0xFF

// ShipServer defines the operations for the gameplay servers.
type ShipServer struct {
	// Precomputed block packet.
	blockListPkt *BlockListPacket
}

func (server *ShipServer) Name() string { return "SHIP" }

func (server *ShipServer) Port() string { return Config.ShipServer.Port }

func (server *ShipServer) Init() error {
	// Precompute the block list packet since it's not going to change.
	numBlocks := Config.ShipServer.NumBlocks
	ship := shipList[0]

	server.blockListPkt = &BlockListPacket{
		Header:  BBHeader{Type: BlockListType, Flags: uint32(numBlocks + 1)},
		Unknown: 0x08,
		Blocks:  make([]Block, numBlocks+1),
	}
	shipName := fmt.Sprintf("%d:%s", ship.id, ship.name)
	copy(server.blockListPkt.ShipName[:], util.ConvertToUtf16(shipName))

	for i := 0; i < numBlocks; i++ {
		b := &server.blockListPkt.Blocks[i]
		b.Unknown = 0x12
		b.BlockId = uint32(i + 1)
		blockName := fmt.Sprintf("BLOCK %02d", i+1)
		copy(b.BlockName[:], util.ConvertToUtf16(blockName))
	}

	// Always append a menu item for returning to the ship select screen.
	b := &server.blockListPkt.Blocks[numBlocks]
	b.Unknown = 0x12
	b.BlockId = BackMenuItem
	copy(b.BlockName[:], util.ConvertToUtf16("Ship Selection"))
	return nil
}

func (server *ShipServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewShipClient(conn)
}

func NewShipClient(conn *net.TCPConn) (*Client, error) {
	cCrypt := crypto.NewBBCrypt()
	sCrypt := crypto.NewBBCrypt()
	sc := NewClient(conn, BBHeaderSize, cCrypt, sCrypt)

	err := error(nil)
	if SendWelcome(sc) != nil {
		err = errors.New("Error sending welcome packet to: " + sc.IPAddr())
		sc = nil
	}
	return sc, err
}

func (server *ShipServer) Handle(c *Client) error {
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case LoginType:
		err = server.HandleShipLogin(c)
	case MenuSelectType:
		var pkt MenuSelectionPacket
		util.StructFromBytes(c.Data(), &pkt)
		// They can be at either the ship or block selection menu, so make sure we have the right one.
		if pkt.MenuId == ShipSelectionMenuId {
			// TODO: Hack for now, but this coupling on the login server logic needs to go away.
			err = server.HandleShipSelection(c)
		} else {
			err = server.HandleBlockSelection(c, pkt)
		}
	default:
		Log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *ShipServer) HandleShipLogin(sc *Client) error {
	if _, err := VerifyAccount(sc); err != nil {
		return err
	}
	if err := server.sendSecurity(sc, BBLoginErrorNone, sc.guildcard, sc.teamId); err != nil {
		return err
	}
	return server.sendBlockList(sc)
}

// Send the security initialization packet with information about the user's
// authentication status.
func (server *ShipServer) sendSecurity(client *Client, errorCode BBLoginError,
	guildcard uint32, teamId uint32) error {

	// Constants set according to how Newserv does it.
	pkt := &SecurityPacket{
		Header:       BBHeader{Type: LoginSecurityType},
		ErrorCode:    uint32(errorCode),
		PlayerTag:    0x00010000,
		Guildcard:    guildcard,
		TeamId:       teamId,
		Config:       &client.config,
		Capabilities: 0x00000102,
	}
	Log.Debug("Sending Security Packet")
	return EncryptAndSend(client, pkt)
}

// Send the client the block list on the selection screen.
func (server *ShipServer) sendBlockList(client *Client) error {
	Log.Debug("Sending Block List Packet")
	return EncryptAndSend(client, server.blockListPkt)
}

// Player selected one of the items on the ship select screen.
func (server *ShipServer) HandleShipSelection(client *Client) error {
	var pkt MenuSelectionPacket
	util.StructFromBytes(client.Data(), &pkt)
	selectedShip := pkt.ItemId - 1
	if selectedShip < 0 || selectedShip >= uint32(len(shipList)) {
		return errors.New("Invalid ship selection: " + string(selectedShip))
	}
	s := &shipList[selectedShip]
	return SendRedirect(client, s.ipAddr[:], s.port)
}

// The player selected a block to join from the menu.
func (server *ShipServer) HandleBlockSelection(sc *Client, pkt MenuSelectionPacket) error {
	// Grab the chosen block and redirect them to the selected block server.
	port, _ := strconv.ParseInt(Config.ShipServer.Port, 10, 16)
	selectedBlock := pkt.ItemId

	if selectedBlock == BackMenuItem {
		server.SendShipList(sc, shipList)
	} else if int(selectedBlock) > Config.ShipServer.NumBlocks {
		return fmt.Errorf("Block selection %v out of range %v", selectedBlock, Config.ShipServer.NumBlocks)
	}

	ipAddr := BroadcastIP()
	return SendRedirect(sc, ipAddr[:], uint16(uint32(port)+selectedBlock))
}

// Send the menu items for the ship select screen.
func (server *ShipServer) SendShipList(client *Client, ships []Ship) error {
	pkt := &ShipListPacket{
		Header:      BBHeader{Type: LoginShipListType, Flags: 0x01},
		Unknown:     0x02,
		Unknown2:    0xFFFFFFF4,
		Unknown3:    0x04,
		ShipEntries: make([]ShipMenuEntry, len(ships)),
	}
	copy(pkt.ServerName[:], "Archon")

	// TODO: Will eventually need a mutex for read.
	for i, ship := range ships {
		item := &pkt.ShipEntries[i]
		item.MenuId = ShipSelectionMenuId
		item.ShipId = ship.id
		copy(item.Shipname[:], util.ConvertToUtf16(string(ship.name[:])))
	}

	Log.Debug("Sending Ship List Packet")
	return EncryptAndSend(client, pkt)
}
