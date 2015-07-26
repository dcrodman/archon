/*
* Archon Ship Server
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
 */
package ship_server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"libarchon/util"
	"net"
	"os"
)

type Shipgate struct {
	conn   *net.TCPConn
	tlsCfg *tls.Config
}

func (s *Shipgate) Connect() error {
	cfg := GetConfig()

	conn, err := tls.Dial("tcp", cfg.ShipgateHost+":"+cfg.ShipgatePort, s.tlsCfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}

	authPkt := &ShipgateAuthPkt{
		Header: ShipgateHeader{
			Type: ShipgateAuthType,
			Size: 0x20,
			Id:   0x00,
		},
	}
	copy(authPkt.Name[:], cfg.Shipname)

	pkt, _ := util.BytesFromStruct(authPkt)
	if _, err = conn.Write(pkt); err != nil {
		log.Important("Failed to connect to shipgate: ", err.Error())
		return err
	}
	ack := make([]byte, 8)
	if _, err = conn.Read(ack); err != nil {
		log.Important("Shipgate connection error: ", err.Error())
		return err
	}
	var ackPkt ShipgateHeader
	util.StructFromBytes(ack, &ackPkt)
	if ackPkt.Type != ShipgateAuthAck {
		log.Important("Shipgate authentication failed")
		return errors.New("Auth failed")
	}
	return nil
}

// Returns request id
// func (s *Shipgate) makeRequest() int {

// }
