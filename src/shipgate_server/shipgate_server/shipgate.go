/*
* Archon Shipgate Server
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
* The Shipgate server logic.
 */

package shipgate_server

import (
	"fmt"
	"libarchon/logger"
	"libarchon/util"
)

// Process packets sent to the LOGIN port by sending them off to another handler or by
// taking some brief action.
func processShipgatePacket(client *ShipgateClient) error {
	var pktHeader ShipgateHeader
	util.StructFromBytes(client.recvData[:ShipgateHeaderSize], &pktHeader)

	if GetConfig().DebugMode {
		fmt.Printf("Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.recvData, int(pktHeader.Size))
		fmt.Println()
	}

	var err error = nil
	switch pktHeader.Type {
	default:
		msg := fmt.Sprintf("Received unknown packet %x from %s", pktHeader.Type, client.ipAddr)
		log.Info(msg, logger.MediumPriority)
	}
	return err
}
