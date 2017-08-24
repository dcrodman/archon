/*
* Archon PSO Server
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
* ---------------------------------------------------------------------
* The BLOCK and SHIP server logic.
 */
package main

import (
	"errors"
	"fmt"
	crypto "github.com/dcrodman/archon/encryption"
	"github.com/dcrodman/archon/util"
	"net"
	"strconv"
)

// Block ID reserved for returning to the ship select menu.
const BackMenuItem = 0xFF

// Info about the available block servers.
type Block struct {
	Unknown   uint16
	BlockId   uint32
	Padding   uint16
	BlockName [36]byte
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

type ShipServer struct {
	// Precomputed block packet.
	blockPkt *BlockListPacket
}

func (server *ShipServer) Name() string { return "SHIP" }

func (server *ShipServer) Port() string { return config.ShipPort }

func (server *ShipServer) Init() error {
	// Precompute the block list packet since it's not going to change.
	numBlocks := config.NumBlocks
	ship := shipList[0]

	server.blockPkt = &BlockListPacket{
		Header:  BBHeader{Type: BlockListType, Flags: uint32(numBlocks + 1)},
		Unknown: 0x08,
		Blocks:  make([]Block, numBlocks+1),
	}
	shipName := fmt.Sprintf("%d:%s", ship.id, ship.name)
	copy(server.blockPkt.ShipName[:], util.ConvertToUtf16(shipName))

	for i := 0; i < numBlocks; i++ {
		b := &server.blockPkt.Blocks[i]
		b.Unknown = 0x12
		b.BlockId = uint32(i + 1)
		blockName := fmt.Sprintf("BLOCK %02d", i+1)
		copy(b.BlockName[:], util.ConvertToUtf16(blockName))
	}
	// Always append a menu item for returning to the ship select screen.
	b := &server.blockPkt.Blocks[numBlocks]
	b.Unknown = 0x12
	b.BlockId = BackMenuItem
	copy(b.BlockName[:], util.ConvertToUtf16("Ship Selection"))
	return nil
}

func (server *ShipServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewShipClient(conn)
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
		log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
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

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Security Packet")
	}
	return client.SendEncrypted(data, size)
}

// Send the client the block list on the selection screen.
func (server *ShipServer) sendBlockList(client *Client) error {
	data, size := util.BytesFromStruct(server.blockPkt)
	if config.DebugMode {
		fmt.Println("Sending Block Packet")
	}
	return client.SendEncrypted(data, size)
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
	port, _ := strconv.ParseInt(config.ShipPort, 10, 16)
	selectedBlock := pkt.ItemId
	if selectedBlock == BackMenuItem {
		server.SendShipList(sc, shipList)
	} else if int(selectedBlock) > config.NumBlocks {
		return errors.New(fmt.Sprintf("Block selection %v out of range %v", selectedBlock, config.NumBlocks))
	}
	ipAddr := config.HostnameBytes()
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

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Ship List Packet")
	}
	return client.SendEncrypted(data, size)
}

// Block sub-server definition.
type BlockServer struct {
	name string
	port string

	lobbyPkt LobbyListPacket
}

func (server *BlockServer) Name() string { return server.name }

func (server *BlockServer) Port() string { return server.port }

func (server *BlockServer) Init() error {
	// Precompute our lobby list since this won't change once the server has started.
	server.lobbyPkt.Header.Size = BBHeaderSize
	server.lobbyPkt.Header.Type = LobbyListType
	server.lobbyPkt.Header.Flags = uint32(config.NumLobbies)
	for i := 0; i <= config.NumLobbies; i++ {
		server.lobbyPkt.Lobbies = append(server.lobbyPkt.Lobbies, struct {
			MenuId  uint32
			LobbyId uint32
			Padding uint32
		}{
			MenuId:  0x1A0001,
			LobbyId: uint32(i),
			Padding: 0,
		})
		server.lobbyPkt.Header.Size += 12
	}
	return nil
}

func (server *BlockServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewShipClient(conn)
}

func (server *BlockServer) Handle(c *Client) error {
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case LoginType:
		err = server.HandleShipLogin(c)
	default:
		log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *BlockServer) HandleShipLogin(c *Client) error {
	if _, err := VerifyAccount(c); err != nil {
		return err
	}
	if err := server.sendSecurity(c, BBLoginErrorNone, c.guildcard, c.teamId); err != nil {
		return err
	}
	if err := server.sendBlockList(c); err != nil {
		return err
	}
	return server.sendLobbyList(c)
}

func (server *BlockServer) sendSecurity(client *Client, errorCode BBLoginError,
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

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Security Packet")
	}
	return client.SendEncrypted(data, size)
}

// Send the client the block list on the selection screen.
func (server *BlockServer) sendBlockList(client *Client) error {
	//data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Block Packet - NOT IMPLEMENTED")
	}
	//return client.SendEncrypted(data, size)
	return nil
}

// Send the client the lobby list on the selection screen.
func (server *BlockServer) sendLobbyList(client *Client) error {
	data, size := util.BytesFromStruct(server.lobbyPkt)
	if config.DebugMode {
		fmt.Println("Sending Lobby List Packet")
	}
	return client.SendEncrypted(data, size)
}
