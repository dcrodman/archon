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
	"github.com/dcrodman/archon/util"
	"net"
)

// Info about the available block servers.
type Block struct {
	Unknown   uint16
	BlockId   uint32
	Padding   uint16
	BlockName [36]byte
}

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

	DebugLog("Sending Security Packet")
	return EncryptAndSend(client, pkt)
}

// Send the client the block list on the selection screen.
func (server *BlockServer) sendBlockList(client *Client) error {
	//data, size := util.BytesFromStruct(pkt)
	DebugLog("Sending Block Packet - NOT IMPLEMENTED")
	//return client.SendEncrypted(data, size)
	return nil
}

// Send the client the lobby list on the selection screen.
func (server *BlockServer) sendLobbyList(client *Client) error {
	DebugLog("Sending Lobby List Packet")
	return EncryptAndSend(client, server.lobbyPkt)
}
