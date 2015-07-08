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
* Starting point for the login server. Initializes the configuration
* package and sets up the workers listening on the necessary ports.
* Also provides top-level functions and other code shared between
* the two (found in login.go and character.go).
 */
package login_server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"libarchon/encryption"
	"libarchon/logger"
	"libarchon/server"
	"libarchon/util"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
)

var log *logger.Logger
var loginConnections *server.ConnectionList = server.NewClientList()
var charConnections *server.ConnectionList = server.NewClientList()
var shipConnections *server.ConnectionList = server.NewClientList()

var shipgateKey *rsa.PrivateKey
var sessionKey *cipher.BlockMode

var defaultShip ShipEntry
var shipList []ShipEntry
var shipListMutex sync.RWMutex

// Struct for holding client-specific data.
type LoginClient struct {
	c         *server.Client
	guildcard uint32
	teamId    uint32
	isGm      bool

	gcData     []byte
	gcDataSize uint16
	config     ClientConfig
	flag       uint32
}

func (lc LoginClient) Client() *server.Client { return lc.c }
func (lc LoginClient) IPAddr() string         { return lc.c.IpAddr }

// Struct for representing available ships in the ship selection menu.
type ShipEntry struct {
	Unknown  uint16 // Always 0x12
	Id       uint32
	Padding  uint16
	Shipname [23]byte
}

// A ship entry as defined by the Http response from the shipgate.
type ShipgateListEntry struct {
	Shipname   [23]byte
	Hostname   string
	Port       string
	NumPlayers int
}

type pktHandler func(p *LoginClient) error

// Handle account verification tasks common to both the login and character servers.
func verifyAccount(client *LoginClient) (*LoginPkt, error) {
	var loginPkt LoginPkt
	util.StructFromBytes(client.c.RecvData, &loginPkt)

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
	loginClient := new(LoginClient)
	client := new(server.Client)

	client.Conn = conn
	addr := strings.Split(conn.RemoteAddr().String(), ":")
	client.IpAddr = addr[0]
	client.Port = addr[1]

	client.ClientCrypt = encryption.NewCrypt()
	client.ServerCrypt = encryption.NewCrypt()
	client.ClientCrypt.CreateBBKeys()
	client.ServerCrypt.CreateBBKeys()
	client.RecvData = make([]byte, 512)

	loginClient.c = client

	var err error = nil
	if SendWelcome(loginClient) != 0 {
		err = errors.New("Error sending welcome packet to: " + client.IpAddr)
		loginClient = nil
	}
	return loginClient, err
}

// Handle communication with a particular client until the connection is
// closed or an error is encountered.
func handleClient(client *LoginClient, desc string, handler pktHandler, list *server.ConnectionList) {
	defer func() {
		if err := recover(); err != nil {
			errMsg := fmt.Sprintf("Error in client communication: %s: %s\n%s\n",
				client.IPAddr(), err, debug.Stack())
			log.Error(errMsg, logger.CriticalPriority)
		}
		client.c.Conn.Close()
		list.RemoveClient(client)
		log.Info("Disconnected "+desc+" client "+client.IPAddr(), logger.MediumPriority)
	}()

	log.Info("Accepted "+desc+" connection from "+client.IPAddr(), logger.MediumPriority)
	ec := server.Generate(client, BBHeaderSize)
	var err error
	for {
		if err = <-ec; err == io.EOF {
			client.c.Conn.Close()
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error(), logger.MediumPriority)
			break
		}

		if err = handler(client); err != nil {
			log.Info(err.Error(), logger.LowPriority)
			break
		}
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
				SendClientMessage(client, "Client is already connected to the server.")
				client.c.Conn.Close()
			} else {
				list.AddClient(client)
				go handleClient(client, id, handler, list)
			}
		}
	}
	wg.Done()
}

// Initialize the server's private PKCS1 key used for registering
// ships and generate a 16 byte key for an AES cipher to be used
// for the majority of ship communication.
func initKeys(dir string) {
	filename := dir + "/" + PrivateKeyFile
	fmt.Printf("Loading private key %s...", filename)
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

	bytes = make([]byte, 16)
	rand.Read(bytes)
	ac, err := aes.NewCipher(bytes)
	fmt.Printf("Doing something with aes %v\n", ac)
	// sessionKey = cipher.NewCBCEncrypter(b, iv)
}

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

func StartServer() {
	fmt.Println("Initializing Archon LOGIN and CHARACTER servers...")
	config := GetConfig()

	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", LoginConfigFile)
	err := config.InitFromFile(LoginConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+LoginConfigFile)
		err = config.InitFromFile(LoginConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("%s\n", err.Error())
			os.Exit(-1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Initialize the logger.
	log = logger.New(config.logWriter, config.LogLevel)

	loadParameterFiles()
	loadBaseStats()

	initKeys(config.KeysDir)

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

	// Open up our web port for retrieving player counts. If we're in debug mode, add a path
	// for dumping pprof output containing the stack traces of all running goroutines.
	http.HandleFunc("/list", handleShipCountRequest)
	if config.DebugMode {
		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			pprof.Lookup("goroutine").WriteTo(resp, 1)
		})
	}
	go http.ListenAndServe(":"+config.WebPort, nil)

	log.Info("Server Initialized", logger.CriticalPriority)
	// Create a WaitGroup so that main won't exit until the server threads have exited.
	var wg sync.WaitGroup
	wg.Add(3)
	go startWorker(&wg, "LOGIN", config.LoginPort, processLoginPacket, loginConnections)
	go startWorker(&wg, "CHARACTER", config.CharacterPort, processCharacterPacket, charConnections)
	go startWorker(&wg, "SHIPGATE", config.ShipgatePort, processShipgatePacket, shipConnections)
	wg.Wait()
}
