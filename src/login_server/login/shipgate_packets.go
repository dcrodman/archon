/*
* Archon Login Server
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
*
* Packet constants and structures for the packets exchanged between
* the central shipgate server and connected ships.
 */
package login

import (
	"fmt"
	"libarchon/util"
)

// Packet types for the shipgate. These can overlap since they aren't
// processed by the same set of handlers as the client ones.
const (
	ShipgateHeaderSize  = 8
	ShipgateAuthType    = 0x01
	ShipgateAuthAckType = 0x02
	ShipgatePingType    = 0x03
)

type ShipgateHeader struct {
	Size uint16
	Type uint16
	// Used to distinguish between requests.
	Id uint32
}

// Initial auth request sent to the shipgate.
type ShipgateAuthPkt struct {
	Header ShipgateHeader
	Name   [24]byte
}

// Send the packet serialized (or otherwise contained) in pkt to a ship.
func SendShipPacket(ship *Ship, pkt []byte, length uint16) int {
	if err := ship.Send(pkt[:length]); err != nil {
		log.Warn("Error sending to ship %v: %s", ship.IPAddr(), err.Error())
		return -1
	}
	return 0
}

// Ship name acknowledgement.
func SendAuthAck(ship *Ship) int {
	pkt := &ShipgateHeader{
		Size: ShipgateHeaderSize,
		Type: ShipgateAuthAckType,
		Id:   0,
	}
	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Auth Ack")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return SendShipPacket(ship, data, uint16(size))
}

// Liveliness check.
func SendPing(ship *Ship) int {
	pkt := &ShipgateHeader{
		Size: ShipgateHeaderSize,
		Type: ShipgatePingType,
		Id:   0,
	}
	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Ping")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return SendShipPacket(ship, data, uint16(size))
}
