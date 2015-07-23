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
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
)

type Shipgate struct {
	ip        *net.TCPAddr
	conn      *net.TCPConn
	publicKey *crypto.PublicKey
}

func InitShipgate() (*Shipgate, error) {
	cfg := GetConfig()

	pool := x509.NewCertPool()
	certData, err := ioutil.ReadFile("cert.pem")
	pool.AppendCertsFromPEM(certData)
	tlsCfg := &tls.Config{RootCAs: pool}

	conn, err := tls.Dial("tcp", cfg.ShipgateHost+":"+cfg.ShipgatePort, tlsCfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	b, err := conn.Write([]byte("oh herrow"))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	fmt.Println(b)

	shipgate := new(Shipgate)

	return shipgate, nil
}

func (s *Shipgate) Connect() error {
	// var err error
	// s.conn, err = net.DialTCP("tcp", cfg.ShipgateHost+":"+cfg.ShipgatePort, nil)
	// if err != nil {
	// 	return errors.New("Error connecting to shipgate: " + err.Error())
	// }
	// Authenticate and load symmetric key
	return nil
}

// Returns request id
// func (s *Shipgate) makeRequest() int {

// }
