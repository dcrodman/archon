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
* Starting point for the login server. Initializes the configuration package and takes care of
* launching the LOGIN and CHARACTER servers. Also provides top-level functions and other code
* shared between the two (found in login.go and character.go).
 */
package login_server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"libarchon/encryption"
	"libarchon/logger"
	"libarchon/server"
	"libarchon/util"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
)

var log *logger.Logger
var loginConnections *server.ConnectionList = server.NewClientList()
var charConnections *server.ConnectionList = server.NewClientList()

var defaultShip ShipEntry
var shipList []ShipEntry

// Struct for holding client-specific data.
type LoginClient struct {
	conn   *net.TCPConn
	ipAddr string
	port   string

	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt

	recvData   []byte
	recvSize   int
	packetSize uint16

	guildcard uint32
	teamId    uint32
	isGm      bool

	gcData     []byte
	gcDataSize uint16
	config     ClientConfig
	flag       uint32
}

func (lc LoginClient) Connection() *net.TCPConn { return lc.conn }
func (lc LoginClient) IPAddr() string           { return lc.ipAddr }

type ShipEntry struct {
	Unknown  uint16 // Always 0x12
	Id       uint32
	Padding  uint16
	Shipname [23]byte
}

type pktHandler func(p *LoginClient) error

// Handle account verification tasks common to both the login and character servers.
func verifyAccount(client *LoginClient) (*LoginPkt, error) {
	var loginPkt LoginPkt
	util.StructFromBytes(client.recvData, &loginPkt)

	// Passwords are stored as sha256 hashes, so hash what the client sent us for the query.
	hasher := sha256.New()
	hasher.Write(util.StripPadding(loginPkt.Password[:]))
	pktUername := string(util.StripPadding(loginPkt.Username[:]))
	pktPassword := hex.EncodeToString(hasher.Sum(nil)[:])

	var username, password string
	var isBanned, isActive bool
	row := GetConfig().Database().QueryRow("SELECT username, password, "+
		"guildcard, is_gm, is_banned, is_active, team_id from account_data "+
		"WHERE username = ? and password = ?", pktUername, pktPassword)
	err := row.Scan(&username, &password, &client.guildcard,
		&client.isGm, &isBanned, &isActive, &client.teamId)
	switch {
	// Check if we have a valid username/combination.
	case err == sql.ErrNoRows:
		// The same error is returned for invalid passwords as attempts to log in
		// with a nonexistent username as some measure of account security. Note
		// that if this is changed to query by username and add a password check,
		// the index on account_data will need to be modified.
		SendSecurity(client, BBLoginErrorPassword, 0, 0)
		return nil, errors.New("Account does not exist for username: " + username)
	// Database error?
	case err != nil:
		SendClientMessage(client, "Encountered an unexpected error while accessing the "+
			"database.\n\nPlease contact your server administrator.")
		log.DBError(err.Error())
		return nil, err
	// Is the account banned?
	case isBanned:
		SendSecurity(client, BBLoginErrorBanned, 0, 0)
		return nil, errors.New("Account banned: " + username)
	// Has the account been activated?
	case !isActive:
		SendClientMessage(client, "Encountered an unexpected error while accessing the "+
			"database.\n\nPlease contact your server administrator.")
		return nil, errors.New("Account must be activated for username: " + username)
	}
	// Copy over the config, which should indicate how far they are in the login flow.
	util.StructFromBytes(loginPkt.Security[:], &client.config)

	// TODO: Hardware ban check.
	return &loginPkt, nil
}

// Create and initialize a new struct to hold client information.
func newClient(conn *net.TCPConn) (*LoginClient, error) {
	client := new(LoginClient)
	client.conn = conn
	addr := strings.Split(conn.RemoteAddr().String(), ":")
	client.ipAddr = addr[0]
	client.port = addr[1]

	client.clientCrypt = encryption.NewCrypt()
	client.serverCrypt = encryption.NewCrypt()
	client.clientCrypt.CreateBBKeys()
	client.serverCrypt.CreateBBKeys()

	client.recvData = make([]byte, 512)

	var err error = nil
	if SendWelcome(client) != 0 {
		err = errors.New("Error sending welcome packet to: " + client.ipAddr)
		client = nil
	}
	return client, err
}

// Handle communication with a particular client until the connection is
// closed or an error is encountered.
func handleClient(client *LoginClient, desc string, handler pktHandler, list *server.ConnectionList) {
	defer func() {
		if err := recover(); err != nil {
			errMsg := fmt.Sprintf("Error in client communication: %s: %s\n%s\n",
				client.ipAddr, err, debug.Stack())
			log.Error(errMsg, logger.CriticalPriority)
		}
		client.conn.Close()
		list.RemoveClient(client)
		log.Info("Disconnected "+desc+" client "+client.ipAddr, logger.MediumPriority)
	}()

	log.Info("Accepted "+desc+" connection from "+client.ipAddr, logger.MediumPriority)
	// We're running inside a goroutine at this point, so we can block on this connection
	// and not interfere with any other clients.
	for {
		// Wait for the packet header.
		for client.recvSize < BBHeaderSize {
			bytes, err := client.conn.Read(client.recvData[client.recvSize:BBHeaderSize])
			if bytes == 0 || err == io.EOF {
				// The client disconnected, we're done.
				client.conn.Close()
				return
			} else if err != nil {
				// Socket error, nothing we can do now
				log.Warn("Socket Error ("+client.ipAddr+") "+err.Error(),
					logger.MediumPriority)
				return
			}
			client.recvSize += bytes

			if client.recvSize >= BBHeaderSize {
				// We have our header; decrypt it.
				client.clientCrypt.Decrypt(client.recvData[:BBHeaderSize], BBHeaderSize)
				client.packetSize, err = util.GetPacketSize(client.recvData[:2])
				if err != nil {
					// Something is seriously wrong if this causes an error. Bail.
					panic(err.Error())
				}
				// PSO likes to occasionally send us packets that are longer than their
				// declared size. Adjust the expected length just in case in order to
				// avoid leaving stray bytes in the buffer.
				for client.packetSize%BBHeaderSize != 0 {
					client.packetSize++
				}
			}
		}
		pktSize := int(client.packetSize)
		// Grow the client's receive buffer if they send us a packet bigger
		// than its current capacity.
		if pktSize > cap(client.recvData) {
			newSize := pktSize + len(client.recvData)
			newBuf := make([]byte, newSize)
			copy(newBuf, client.recvData)
			client.recvData = newBuf
			msg := fmt.Sprintf("Reallocated buffer to %v bytes", newSize)
			log.Info(msg, logger.LowPriority)
		}

		// Read in the rest of the packet.
		for client.recvSize < pktSize {
			remaining := pktSize - client.recvSize
			bytes, err := client.conn.Read(
				client.recvData[client.recvSize : client.recvSize+remaining])
			if err != nil {
				log.Warn("Socket Error ("+client.ipAddr+") "+err.Error(),
					logger.MediumPriority)
				return
			}
			client.recvSize += bytes
		}

		// We have the whole thing; decrypt the rest of it if needed and pass it along.
		if client.packetSize > BBHeaderSize {
			client.clientCrypt.Decrypt(
				client.recvData[BBHeaderSize:client.packetSize],
				uint32(client.packetSize-BBHeaderSize))
		}
		if err := handler(client); err != nil {
			log.Info(err.Error(), logger.LowPriority)
			break
		}

		// Extra bytes left in the buffer will just be ignored.
		client.recvSize = 0
		client.packetSize = 0
	}
}

// Creates the socket and starts listening for connections on the specified
// port, spawning off goroutines to handle communications for each client.
func startWorker(wg *sync.WaitGroup, id, port string, handler pktHandler, list *server.ConnectionList) {
	cfg := GetConfig()
	socket, err := server.OpenSocket(cfg.Hostname, port)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for %s connections on %s:%s...\n", id, cfg.Hostname, port)
	for {
		// Poll until we can accept more clients.
		for list.Count() < cfg.MaxConnections {
			connection, err := socket.AcceptTCP()
			if err != nil {
				log.Error("Failed to accept connection: "+err.Error(), logger.HighPriority)
				continue
			}
			client, err := newClient(connection)
			if err != nil {
				continue
			}
			if list.HasClient(client) {
				SendClientMessage(client, "This computer is already connected to the server.")
				client.conn.Close()
			} else {
				list.AddClient(client)
				go handleClient(client, id, handler, list)
			}
		}
	}
	wg.Done()
}

func StartServer() {
	fmt.Println("Initializing Archon LOGIN and CHARACTER servers...")
	config := GetConfig()

	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", loginConfigFile)
	err := config.InitFromFile(loginConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+loginConfigFile)
		err = config.InitFromFile(loginConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("%s\n", err.Error())
			os.Exit(-1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Initialize the logger.
	log = logger.New(config.logWriter, config.LogLevel)
	log.Info("Server Initialized", logger.CriticalPriority)

	loadParameterFiles()
	loadBaseStats()

	// Create our "No Ships" item to indicate the absence of any ship servers.
	defaultShip.Unknown = 0x12
	defaultShip.Id = 1
	copy(defaultShip.Shipname[:], util.ConvertToUtf16("No Ships"))
	shipList = append(shipList, defaultShip)

	// Initialize the database.
	fmt.Printf("Connecting to MySQL database %s:%s...", config.DBHost, config.DBPort)
	err = config.InitDb()
	if err != nil {
		fmt.Println("Failed.\nPlease make sure the database connection parameters are correct.")
		fmt.Printf("Error: %s\n", err)
		os.Exit(-1)
	}
	fmt.Println("Done.")
	defer config.CloseDb()

	if config.DebugMode {
		go server.CreateStackTraceServer("127.0.0.1:8081", "/")
	}

	// Create a WaitGroup so that main won't exit until the server threads have exited.
	var wg sync.WaitGroup
	wg.Add(2)
	go startWorker(&wg, "LOGIN", config.LoginPort, processLoginPacket, loginConnections)
	go startWorker(&wg, "CHARACTER", config.CharacterPort, processCharacterPacket, charConnections)
	wg.Wait()
}
