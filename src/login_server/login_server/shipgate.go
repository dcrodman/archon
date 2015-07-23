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
* Handles the connection initialization and management for connected
* ships. This module handles all of its own connection logic since the
* shipgate protocol differs from the way game clients are processed.
 */
package login_server

import (
	// "crypto/rsa"
	"crypto/tls"
	// "errors"
	"fmt"
	// "io/ioutil"
	"libarchon/logger"
	// "libarchon/server"
	"libarchon/util"
	"net"
	"os"
	// "runtime/debug"
	"sync"
)

type Ship struct {
	Conn   *net.TCPConn
	IpAddr string
	Port   string

	RecvData   []byte
	RecvSize   int
	PacketSize uint16
}

func (s *Ship) IPAddr() string { return s.IpAddr }

// Loop for the life of the server, pinging the shipgate every 30
// seconds to update the list of available ships.
func fetchShipList() {
	// config := GetConfig()
	// errorInterval, pingInterval := time.Second*5, time.Second*60
	// shipgateUrl := fmt.Sprintf("http://%s:%s/list", config.ShipgateHost, config.ShipgatePort)
	// for {
	// 	resp, err := http.Get(shipgateUrl)
	// 	if err != nil {
	// 		log.Error("Failed to connect to shipgate: "+err.Error(), logger.CriticalPriority)
	// 		// Sleep for a shorter interval since we want to know as soon
	// 		// as the shipgate is back online.
	// 		time.Sleep(errorInterval)
	// 	} else {
	// 		ships := make([]ShipgateListEntry, 1)
	// 		// Extract the Http response and convert it from JSON.
	// 		shipData := make([]byte, 100)
	// 		resp.Body.Read(shipData)
	// 		if err = json.Unmarshal(util.StripPadding(shipData), &ships); err != nil {
	// 			log.Error("Error parsing JSON response from shipgate: "+err.Error(),
	// 				logger.MediumPriority)
	// 			time.Sleep(errorInterval)
	// 			continue
	// 		}

	// 		// Taking the easy way out and just reallocating the entire slice
	// 		// to make the GC do the hard part. If this becomes an issue for
	// 		// memory footprint then the list should be overwritten in-place.
	// 		shipListMutex.Lock()
	// 		if len(ships) < 1 {
	// 			shipList = []ShipEntry{defaultShip}
	// 		} else {
	// 			shipList = make([]ShipEntry, len(shipList))
	// 			for i := range ships {
	// 				ship := shipList[i]
	// 				ship.Unknown = 0x12
	// 				// TODO: Does this have any actual significance? Will the possibility
	// 				// of a ship id changing for the same ship break things?
	// 				ship.Id = uint32(i)
	// 				ship.Shipname = ships[i].Shipname
	// 			}
	// 		}
	// 		shipListMutex.Unlock()
	// 		log.Info("Updated ship list", logger.LowPriority)
	// 		time.Sleep(pingInterval)
	// 	}
	// }
}

func processShipgatePacket(ship *LoginClient) error {
	return nil
}

// Per-ship connection loop.
func handleShipConnection(conn *net.Conn) {
	// ship, err := authenticate(conn)
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		errMsg := fmt.Sprintf("Error in ship communication: %s: %s\n%s\n",
	// 			ship.IPAddr(), err, debug.Stack())
	// 		log.Error(errMsg, logger.CriticalPriority)
	// 	}
	// 	conn.Close()
	// 	log.Info("Disconnected ship "+ship.IPAddr(), logger.CriticalPriority)
	// 	// TODO: Remove from ship list
	// }()
	// if err != nil {
	// 	log.Warn("Failed to authenticate ship: "+err.Error(), logger.CriticalPriority)
	// 	return
	// }
	// // sessionKey, err := aes.NewCipher(bytes)
	// // sessionKey = cipher.NewCBCEncrypter(b, iv)
}

// Wait for ship connections and spin off goroutines to handle them.
func handleShipgateConnections(cfg *configuration) {
	for {
		connection, err := socket.Accept()
		if err != nil {
			log.Warn("Failed to accept connection: %s", err.Error())
			continue
		}
		// TODO: Add to ship list
		data := make([]byte, 50)
		b, err := connection.Read(data)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(-1)
		}
		fmt.Println(b)
		fmt.Println(connection.RemoteAddr())
		util.PrintPayload(data, b)

		connection.Close()
	}
}

func startShipgate(wg *sync.WaitGroup) {
	cfg := GetConfig()

	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	socket, err := tls.Listen("tcp", cfg.Hostname+":"+cfg.ShipgatePort, tlsCfg)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	fmt.Printf("Waiting for SHIPGATE connections on %s:%s...\n",
		cfg.Hostname, cfg.ShipgatePort)

	handleShipgateConnections(cfg)
	wg.Done()
}
