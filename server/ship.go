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
package server

import (
	"errors"
	"fmt"
	"github.com/dcrodman/archon/server/util"
	"io"
	"net"
)

func NewShipClient(conn *net.TCPConn) (*Client, error) {
	sc := NewClient(conn, BBHeaderSize)
	err := error(nil)
	if sc.SendWelcome() != 0 {
		err = errors.New("Error sending welcome packet to: " + sc.IPAddr())
		sc = nil
	}
	return sc, err
}

func verifyAccount(c *Client) error {
	// TODO: Check for bans based on the login info
	if _, err := VerifyAccount(c); err != nil {
		return err
	}
	return nil
}

func handleShipLogin(c *Client) error {
	if err := verifyAccount(c); err != nil {
		return err
	}
	c.SendSecurity(BBLoginErrorNone, c.guildcard, c.teamId)
	return nil
}

func BlockHandler(sc *Client) {
	var pktHeader PCPktHeader
	for {
		err := sc.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		util.StructFromBytes(sc.Data()[:PCHeaderSize], &pktHeader)
		if config.DebugMode {
			fmt.Printf("BLOCK: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(sc.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		}
	}
}

func ShipHandler(sc *Client) {
	var pktHeader PCPktHeader
	for {
		err := sc.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		util.StructFromBytes(sc.Data()[:PCHeaderSize], &pktHeader)
		if config.DebugMode {
			fmt.Printf("SHIP: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(sc.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		case LoginType:
			err = handleShipLogin(sc)
		}

		if err != nil {
			log.Warn("Error in client communication: " + err.Error())
			return
		}
	}
}

func InitShip() {

}
