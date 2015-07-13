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
	// "crypto/aes"
	// "crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	// "libarchon/logger"
	// "libarchon/server"
	"net"
	"net/http"
	"os"
	"strconv"
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

const ShipgateHeaderSize = 0x04

var shipgateKey *rsa.PrivateKey
var sessionKey []byte

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

// Distributing a shared symmetric key is insecure, so in order to
// allow symmetric encryption an initial handshake is performed
// using PKCS1 and known keys. The shipgate keeps all ship public
// keys (along with its private key) locally and assumes that
// connecting ships in turn have its public key. This doubles as a
// registration mechanism since we only allow ships whose public keys
// we have stored to connect.
func authenticateClient(c *net.TCPConn) {
}

func processShipgatePacket(ship *LoginClient) error {
	return nil
}

// Initialize the server's private PKCS1 key used for registering
// ships and generate a 16 byte key for an AES cipher to be used
// for the majority of ship communication.
func initKeys(dir string) {
	filename := dir + "/" + PrivateKeyFile
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("\nError loading private key: %s\n", err.Error())
		os.Exit(-1)
	}

	block, _ := pem.Decode(bytes)
	shipgateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Printf("\nError parsing private key: %s\n", err.Error())
		os.Exit(-1)
	}
	fmt.Printf("Done\n")

	sessionKey = make([]byte, 16)
	rand.Read(sessionKey)
}

// Silently spin off a registration port for ships in the background.
// Ships connect to this port to authenticate and obtain the symmetric
// key that's used for encrypted comms via the SHIPGATE port.
func handleShipRegistration(port string) {
	// socket, err := server.OpenSocket(cfg.Hostname, port)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }
	// for {
	// 	conn, err := socket.Accept()
	// 	if err != nil {
	// 		log.Warn("Failed to accept ship connection", logger.HighPriority)
	// 	}
	// 	go authenticateClient(conn)
	// }
	// socket.Close()
}

func handleShipgateConnections(cfg *configuration) {
	// socket, err := server.OpenSocket(cfg.Hostname, cfg.ShipgatePort)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(-1)
	// }
	// fmt.Printf("Waiting for SHIPGATE connections on %s:%s...\n", cfg.Hostname, cfg.ShipgatePort)
	// connection, err := socket.AcceptTCP()
	// if err != nil {
	// 	log.Error("Failed to accept connection: "+err.Error(), logger.HighPriority)
	// 	continue
	// }
	// 	sessionKey, err := aes.NewCipher(bytes)
	// sessionKey = cipher.NewCBCEncrypter(b, iv)
}

func startShipgate(wg *sync.WaitGroup) {
	cfg := GetConfig()
	initKeys(cfg.KeysDir)

	// The registration port is always the shipgate port + 1.
	regPort, _ := strconv.ParseInt(cfg.ShipgatePort, 10, 0)
	go handleShipRegistration(strconv.FormatInt(regPort+1, 10))
	handleShipgateConnections(cfg)
	wg.Done()
}
