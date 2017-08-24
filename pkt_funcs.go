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
* Packet types, defintitions, and sending functions.
 */
package main

import (
	"fmt"
	"github.com/dcrodman/archon/util"
)

const (
	// Copyright messages the client expects.
	patchCopyright = "Patch Server. Copyright SonicTeam, LTD. 2001"
	loginCopyright = "Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM."
)

var (
	patchCopyrightBytes []byte
	loginCopyrightBytes []byte
	serverName          = util.ConvertToUtf16("Archon")
)

// Deprecated
func sendPacket(c *Client, pkt []byte, length uint16) int {
	if err := c.SendRaw(pkt[:length], length); err != nil {
		log.Info("Error sending to client %v: %s", c.IPAddr(), err.Error())
		return -1
	}
	return 0
}

// Deprecated
func sendEncrypted(c *Client, data []byte, length uint16) int {
	if err := c.SendEncrypted(data, int(length)); err != nil {
		return -1
	}
	return 0
}

// Send the welcome packet to a client with the copyright message and encryption vectors.
func (client *Client) SendWelcome() int {
	pkt := new(WelcomePkt)
	pkt.Header.Type = LoginWelcomeType
	pkt.Header.Size = 0xC8
	copy(pkt.Copyright[:], loginCopyrightBytes)
	copy(pkt.ClientVector[:], client.ClientVector())
	copy(pkt.ServerVector[:], client.ServerVector())

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Packet")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return sendPacket(client, data, uint16(size))
}

// Send the security initialization packet with information about the user's
// authentication status.
func (client *Client) SendSecurity(errorCode BBLoginError,
	guildcard uint32, teamId uint32) int {

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
	return sendEncrypted(client, data, uint16(size))
}

// Send the redirect packet, providing the IP and port of the next server.
func (client *Client) SendRedirect(port uint16, ipAddr [4]byte) int {
	pkt := new(RedirectPacket)
	pkt.Header.Type = RedirectType
	copy(pkt.IPAddr[:], ipAddr[:])
	pkt.Port = port

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Redirect Packet")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Send an error message to the client, usually used before disconnecting.
func (client *Client) SendClientMessage(message string) int {
	pkt := &LoginClientMessagePacket{
		Header: BBHeader{Type: LoginClientMessageType},
		// English? Tethealla sets this.
		Language: 0x00450009,
		Message:  util.ConvertToUtf16(message),
	}
	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Client Message Packet")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Send the menu items for the ship select screen.
func (client *Client) SendShipList(ships []Ship) int {
	pkt := &ShipListPacket{
		Header:      BBHeader{Type: LoginShipListType, Flags: 0x01},
		Unknown:     0x02,
		Unknown2:    0xFFFFFFF4,
		Unknown3:    0x04,
		ShipEntries: make([]ShipMenuEntry, len(ships)),
	}
	copy(pkt.ServerName[:], serverName)

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
	return sendEncrypted(client, data, uint16(size))
}

// Send the client the block list on the selection screen.
func (client *Client) SendBlockList(pkt *BlockListPacket) int {
	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Block Packet")
	}
	return sendEncrypted(client, data, uint16(size))
}

// Send the client the lobby list on the selection screen.
func (client *Client) SendLobbyList(pkt *LobbyListPacket) int {
	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Lobby List Packet")
	}
	return sendEncrypted(client, data, uint16(size))
}

func init() {
	patchCopyrightBytes = []byte(patchCopyright)
	loginCopyrightBytes = []byte(loginCopyright)
}
