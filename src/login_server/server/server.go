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
package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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

	guildcard int
	isGm      bool
}

func (lc LoginClient) Connection() *net.TCPConn { return lc.conn }
func (lc LoginClient) IPAddr() string           { return lc.ipAddr }

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
	row := GetConfig().Database().QueryRow("SELECT username, password, guildcard, is_gm, is_banned, "+
		"is_active from account_data  WHERE username = ? and password = ?", pktUername, pktPassword)
	err := row.Scan(&username, &password, &client.guildcard, &client.isGm, &isBanned, &isActive)
	switch {
	case err == sql.ErrNoRows:
		// TODO: Send E6, return better error
		fmt.Printf("Account doesn't exist\n")
		return nil, err
	case err != nil:
		// TODO: Send E6 for database error
		LogMsg(fmt.Sprintf("SQL Error: %s", err.Error()), LogTypeError, LogPriorityCritical)
		return nil, err
	case isBanned:
		// TODO: Send E6, return error
		fmt.Printf("Account banned\n")
	case !isActive:
		// TODO: Send E6, return error
		fmt.Printf("Account must be activated\n")
	}
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

	client.recvData = make([]byte, 1024)

	var err error = nil
	if SendWelcome(client) != 0 {
		err = util.ServerError{Message: "Error sending welcome packet to: " + client.ipAddr}
		client = nil
	}
	return client, err
}

func Start() {
	config := GetConfig()
	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", loginConfigFile)
	err := config.InitFromFile(loginConfigFile)
	if err != nil {
		path := util.ServerConfigDir + "/" + loginConfigFile
		fmt.Printf("Failed.\nLoading config from %v...", path)
		err = config.InitFromFile(path)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("%s\n", err.Error())
			os.Exit(-1)
		}
	}
	fmt.Printf("Done.\n--Configuration Parameters--\n%v\n\n", config.String())

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
