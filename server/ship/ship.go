// The BLOCK and SHIP server logic.
package ship

//
//import (
//	"errors"
//	"fmt"
//	"github.com/dcrodman/archon"
//	"github.com/dcrodman/archon/server"
//	"github.com/dcrodman/archon/server/character"
//	"github.com/dcrodman/archon/server/shipgate"
//	"net"
//	"strconv"
//
//	"github.com/dcrodman/archon/util"
//	crypto "github.com/dcrodman/archon/util/encryption"
//)
//
//// BackMenuItem is the block ID reserved for returning to the ship select menu.
//const BackMenuItem = 0xFF
//
//// ShipServer defines the operations for the gameplay servers.
//type ShipServer struct {
//	// Precomputed block packet.
//	blockListPkt *archon.BlockListPacket
//}
//
//func NewServer() server.Server {
//	return &ShipServer{}
//}
//
//func (server *ShipServer) Name() string { return "SHIP" }
//
//func (server *ShipServer) Port() string { return archon.Config.ShipServer.Port }
//
//func (server *ShipServer) Init() error {
//	// Precompute the block list packet since it's not going to change.
//	numBlocks := archon.Config.ShipServer.NumBlocks
//	ship := character.shipList[0]
//
//	server.blockListPkt = &archon.BlockListPacket{
//		Header:  archon.BBHeader{Type: archon.BlockListType, Flags: uint32(numBlocks + 1)},
//		Unknown: 0x08,
//		Blocks:  make([]archon.Block, numBlocks+1),
//	}
//	shipName := fmt.Sprintf("%d:%s", ship.id, ship.name)
//	copy(server.blockListPkt.ShipName[:], util.ConvertToUtf16(shipName))
//
//	for i := 0; i < numBlocks; i++ {
//		b := &server.blockListPkt.Blocks[i]
//		b.Unknown = 0x12
//		b.BlockId = uint32(i + 1)
//		blockName := fmt.Sprintf("BLOCK %02d", i+1)
//		copy(b.BlockName[:], util.ConvertToUtf16(blockName))
//	}
//
//	// Always append a menu item for returning to the ship select screen.
//	b := &server.blockListPkt.Blocks[numBlocks]
//	b.Unknown = 0x12
//	b.BlockId = BackMenuItem
//	copy(b.BlockName[:], util.ConvertToUtf16("Ship Selection"))
//	return nil
//}
//
//func (server *ShipServer) NewClient(conn *net.TCPConn) (*server.Client, error) {
//	return NewShipClient(conn)
//}
//
//func NewShipClient(conn *net.TCPConn) (*server.Client, error) {
//	cCrypt := crypto.NewBBCrypt()
//	sCrypt := crypto.NewBBCrypt()
//	sc := server.NewClient(conn, archon.BBHeaderSize, cCrypt, sCrypt)
//
//	err := error(nil)
//	if archon.SendWelcome(sc) != nil {
//		err = errors.New("Error sending welcome packet to: " + sc.IPAddr())
//		sc = nil
//	}
//	return sc, err
//}
//
//func (server *ShipServer) Handle(c *server.Client) error {
//	var hdr archon.BBHeader
//	util.StructFromBytes(c.Data()[:archon.BBHeaderSize], &hdr)
//
//	var err error
//	switch hdr.Type {
//	case archon.LoginType:
//		err = server.HandleShipLogin(c)
//	case archon.MenuSelectType:
//		var pkt archon.MenuSelectionPacket
//		util.StructFromBytes(c.Data(), &pkt)
//		// They can be at either the ship or block selection menu, so make sure we have the right one.
//		if pkt.MenuId == character.ShipSelectionMenuId {
//			// TODO: Hack for now, but this coupling on the login server logic needs to go away.
//			err = server.HandleShipSelection(c)
//		} else {
//			err = server.HandleBlockSelection(c, pkt)
//		}
//	default:
//		archon.Log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
//	}
//	return err
//}
//
//func (server *ShipServer) HandleShipLogin(sc *server.Client) error {
//	if _, err := archon.VerifyAccount(sc); err != nil {
//		return err
//	}
//	if err := server.sendSecurity(sc, archon.BBLoginErrorNone, sc.guildcard, sc.teamId); err != nil {
//		return err
//	}
//	return server.sendBlockList(sc)
//}
//
//// send the security initialization packet with information about the user's
//// authentication status.
//func (server *ShipServer) sendSecurity(client *server.Client, errorCode archon.BBLoginError,
//	guildcard uint32, teamId uint32) error {
//
//	// Constants set according to how Newserv does it.
//	pkt := &archon.SecurityPacket{
//		Header:       archon.BBHeader{Type: archon.LoginSecurityType},
//		ErrorCode:    uint32(errorCode),
//		PlayerTag:    0x00010000,
//		Guildcard:    guildcard,
//		TeamId:       teamId,
//		Config:       &client.config,
//		Capabilities: 0x00000102,
//	}
//	archon.Log.Debug("Sending Security Packet")
//	return archon.EncryptAndSend(client, pkt)
//}
//
//// send the client the block list on the selection screen.
//func (server *ShipServer) sendBlockList(client *server.Client) error {
//	archon.Log.Debug("Sending Block List Packet")
//	return archon.EncryptAndSend(client, server.blockListPkt)
//}
//
//// Player selected one of the items on the ship select screen.
//func (server *ShipServer) HandleShipSelection(client *server.Client) error {
//	var pkt archon.MenuSelectionPacket
//	util.StructFromBytes(client.Data(), &pkt)
//	selectedShip := pkt.ItemId - 1
//	if selectedShip < 0 || selectedShip >= uint32(len(character.shipList)) {
//		return errors.New("Invalid ship selection: " + string(selectedShip))
//	}
//	s := &character.shipList[selectedShip]
//	return archon.SendRedirect(client, s.ipAddr[:], s.port)
//}
//
//// The player selected a block to join from the menu.
//func (server *ShipServer) HandleBlockSelection(sc *server.Client, pkt archon.MenuSelectionPacket) error {
//	// Grab the chosen block and redirect them to the selected block server.
//	port, _ := strconv.ParseInt(archon.Config.ShipServer.Port, 10, 16)
//	selectedBlock := pkt.ItemId
//
//	if selectedBlock == BackMenuItem {
//		server.SendShipList(sc, character.shipList)
//	} else if int(selectedBlock) > archon.Config.ShipServer.NumBlocks {
//		return fmt.Errorf("Block selection %v out of range %v", selectedBlock, archon.Config.ShipServer.NumBlocks)
//	}
//
//	ipAddr := archon.BroadcastIP()
//	return archon.SendRedirect(sc, ipAddr[:], uint16(uint32(port)+selectedBlock))
//}
//
//// send the menu items for the ship select screen.
//func (server *ShipServer) SendShipList(client *server.Client, ships []shipgate.Ship) error {
//	pkt := &archon.ShipListPacket{
//		Header:      archon.BBHeader{Type: archon.LoginShipListType, Flags: 0x01},
//		Unknown:     0x02,
//		Unknown2:    0xFFFFFFF4,
//		Unknown3:    0x04,
//		ShipEntries: make([]character.ShipMenuEntry, len(ships)),
//	}
//	copy(pkt.ServerName[:], "Archon")
//
//	// TODO: Will eventually need a mutex for read.
//	for i, ship := range ships {
//		item := &pkt.ShipEntries[i]
//		item.MenuId = character.ShipSelectionMenuId
//		item.ShipId = ship.id
//		copy(item.Shipname[:], util.ConvertToUtf16(string(ship.name[:])))
//	}
//
//	archon.Log.Debug("Sending Ship List Packet")
//	return archon.EncryptAndSend(client, pkt)
//}
