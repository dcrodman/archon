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
	"fmt"
	"io"
	"net"
	"server/util"
)

type ShipClient struct {
	c *PSOClient
}

func (sc ShipClient) IPAddr() string     { return sc.c.IPAddr() }
func (sc ShipClient) Client() Client     { return sc.c }
func (sc ShipClient) Data() []byte       { return sc.c.Data() }
func (sc ShipClient) HeaderSize() uint16 { return BBHeaderSize }

func NewShipClient(conn *net.TCPConn) (ClientWrapper, error) {
	sc := &ShipClient{c: NewPSOClient(conn, BBHeaderSize)}
	return *sc, nil
}

func BlockHandler(cw ClientWrapper) {
	pc := cw.(ShipClient)
	var pktHeader PCPktHeader
	for {
		err := pc.c.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		util.StructFromBytes(pc.c.Data()[:PCHeaderSize], &pktHeader)
		if config.DebugMode {
			fmt.Printf("BLOCK: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(pc.c.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		}
	}
}

func ShipHandler(cw ClientWrapper) {
	pc := cw.(ShipClient)
	var pktHeader PCPktHeader
	for {
		err := pc.c.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		util.StructFromBytes(pc.c.Data()[:PCHeaderSize], &pktHeader)
		if config.DebugMode {
			fmt.Printf("SHIP: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(pc.c.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		}
	}
}

func InitShip() {
	// Create our ship entry for the built-in ship server.
	defaultShip.Unknown = 0x12
	defaultShip.Id = 1
	copy(defaultShip.Shipname[:], util.ConvertToUtf16(config.ShipName))
	shipList = append(shipList, defaultShip)
}
