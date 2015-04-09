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
	"libarchon/encryption"
	"libarchon/util"
	"net"
	"os"
	"sync"
)

// Struct for holding client-specific data.
type LoginClient struct {
	conn   *net.TCPConn
	ipAddr string

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

// Helper for logging SQL errors and creating an Error instance.
func DBError(err error) error {
	errMsg := fmt.Sprintf("SQL Error: %s", err.Error())
	LogMsg(errMsg, LogTypeError, LogPriorityCritical)
	return &util.ServerError{Message: errMsg}
}

// Handle account verification tasks common to both the login and character servers.
func VerifyAccount(client *LoginClient) (*LoginPkt, error) {
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
		// TODO: Send error message (1A)
		SendSecurity(client, BBLoginErrorUnknown, 0, 0)
		return nil, DBError(err)
	// Is the account banned?
	case isBanned:
		SendSecurity(client, BBLoginErrorBanned, 0, 0)
		return nil, errors.New("Account banned: " + username)
	// Has the account been activated?
	case !isActive:
		// TODO: Send message (1A)
		SendSecurity(client, BBLoginErrorUnregistered, 0, 0)
		return nil, errors.New("Account must be activated for username: " + username)
	}
	// Copy over the config, which should indicate how far they are in the login flow.
	util.StructFromBytes(loginPkt.Security[:], &client.config)

	// TODO: Hardware ban check.
	return &loginPkt, nil
}

// Create and initialize a new struct to hold client information.
func NewClient(conn *net.TCPConn) (*LoginClient, error) {
	client := new(LoginClient)
	client.conn = conn
	client.ipAddr = conn.RemoteAddr().String()

	client.clientCrypt = encryption.NewCrypt()
	client.serverCrypt = encryption.NewCrypt()
	client.clientCrypt.CreateKeys()
	client.serverCrypt.CreateKeys()

	client.recvData = make([]byte, 2048)

	var err error = nil
	if SendWelcome(client) != 0 {
		err = util.ServerError{Message: "Error sending welcome packet to: " + client.ipAddr}
		client = nil
	}
	return client, err
}

func Start() {
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

	// Create a WaitGroup so that main won't exit until the server threads have exited.
	var wg sync.WaitGroup
	wg.Add(2)
	go StartLogin(&wg)
	go StartCharacter(&wg)
	wg.Wait()
}
