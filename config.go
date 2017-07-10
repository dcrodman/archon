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
*
* Singleton package for handling the global server configuration
* and responsible for establishing a connection to the database
* to be maintained during execution.
 */
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dcrodman/archon/util"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

// Configuration structure that can be shared between sub servers.
// The fields are intentionally exported to cut down on verbosity
// with the intent that they be considered immutable.
type Config struct {
	Hostname string
	ExternalIP string
	// Patch ports.
	PatchPort string
	DataPort  string
	// Login ports.
	LoginPort     string
	CharacterPort string
	// Shipgate ports.
	ShipgatePort string
	WebPort      string
	// Ship ports.
	ShipPort string

	// Number of blocks to open on the ship server.
	NumBlocks int
	// Number of lobbies available per block.
	NumLobbies     int
	MaxConnections int

	// Patch server welcome message.
	WelcomeMessage string
	// Scrolling message on ship select.
	ScrollMessage string
	MessageBytes  []byte
	MessageSize   uint16

	PatchDir      string
	ParametersDir string
	KeysDir       string

	// Database parameters.
	database   *sql.DB
	DBHost     string
	DBPort     string
	DBName     string
	DBUsername string
	DBPassword string

	Logfile   string
	LogLevel  string
	DebugMode bool

	// Ship server config.
	ShipName string

	cachedHostBytes [4]byte
	cachedScrollMsg []byte
}

// Singleton instance. Provides reasonable default values so
// that some configurations can remain simpler.
var config *Config = &Config{
	Hostname:       "127.0.0.1",
	ExternalIP:     "127.0.0.1",
	PatchPort:      "11000",
	DataPort:       "11001",
	LoginPort:      "12000",
	CharacterPort:  "12001",
	ShipgatePort:   "13000",
	WebPort:        "14000",
	ShipPort:       "15000",
	NumBlocks:      2,
	NumLobbies:     15,
	MaxConnections: 30000,

	ShipName:       "Unconfigured",
	WelcomeMessage: "Unconfigured Welcome Message",
	ScrollMessage:  "Add a welcome message here",

	PatchDir:      "patches/",
	ParametersDir: "parameters/",
	KeysDir:       "keys/",

	DBHost: "127.0.0.1",
	DBPort: "3306",
	DBName: "archondb",

	Logfile:   "",
	LogLevel:  "warn",
	DebugMode: false,
}

func GetConfig() *Config { return config }

// Populate config with the contents of a JSON file at path fileName. Config parameters
// in the file must match the above fields exactly in order to be read.
func (config *Config) InitFromFile(fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	json.Unmarshal(data, config)

	// Convert the welcome message to UTF-16LE and cache it.
	config.MessageBytes = util.ConvertToUtf16(config.WelcomeMessage)
	// PSOBB expects this prefix to the message, not completely sure why. Language perhaps?
	config.MessageBytes = append([]byte{0xFF, 0xFE}, config.MessageBytes...)
	msgLen := len(config.MessageBytes)
	if msgLen > (1<<16 - 16) {
		return errors.New("Message length must be less than 65,000 characters")
	}
	config.MessageSize = uint16(msgLen)

	config.cachedScrollMsg = util.ConvertToUtf16(config.ScrollMessage)

	// Strip the trailing slash if needed.
	if strings.HasSuffix(config.PatchDir, "/") {
		config.PatchDir = filepath.Dir(config.PatchDir)
	}
	return nil
}

// Establish a connection to the database and ping it to verify.
func (config *Config) InitDb() error {
	dbName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.DBUsername,
		config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	var err error
	config.database, err = sql.Open("mysql", dbName)
	if err == nil {
		err = config.database.Ping()
	}
	return err
}

func (config *Config) CloseDB() {
	config.database.Close()
}

// Returns a reference to the database so that it can remain
// encapsulated and any consistency checks can be centralized.
func (config *Config) DB() *sql.DB {
	if config.database == nil {
		// Don't implicitly initialize the database - if there's an error or other action that causes
		// the reference to become nil then we're probably leaking a connection.
		panic("Attempt to reference uninitialized database")
	}
	return config.database
}

// Convert the hostname string into 4 bytes to be used with the redirect packet.
func (config *Config) HostnameBytes() [4]byte {
	// Hacky, but chances are the IP address isn't going to start with 0 and a
	// fixed-length array can't be null.
	if config.cachedHostBytes[0] == 0x00 {
		parts := strings.Split(config.ExternalIP, ".")
		for i := 0; i < 4; i++ {
			tmp, _ := strconv.ParseUint(parts[i], 10, 8)
			config.cachedHostBytes[i] = uint8(tmp)
		}
	}
	return config.cachedHostBytes
}

// Returns the configured scroll message for the login server.
func (config *Config) ScrollMessageBytes() []byte {
	return config.cachedScrollMsg[:]
}

func (config *Config) String() string {
	outfile := config.Logfile
	if outfile == "" {
		outfile = "Standard Out"
	}
	return "Hostname: " + config.Hostname + "\n" +
		"Patch Port: " + config.PatchPort + "\n" +
		"Data Port: " + config.DataPort + "\n" +
		"Login Port: " + config.LoginPort + "\n" +
		"Character Port: " + config.CharacterPort + "\n" +
		"Shipgate Port: " + config.ShipgatePort + "\n" +
		"Web Port: " + config.WebPort + "\n" +
		"Ship Port: " + config.ShipPort + "\n" +
		"Num Ship Blocks: " + strconv.FormatInt(int64(config.NumBlocks), 10) + "\n" +
		"Num Lobbies: " + strconv.FormatInt(int64(config.NumLobbies), 10) + "\n" +
		"Max Connections: " + strconv.FormatInt(int64(config.MaxConnections), 10) + "\n" +
		"Ship Name: " + config.ShipName + "\n" +
		"Welcome Message: " + config.WelcomeMessage + "\n" +
		"Parameters Directory: " + config.ParametersDir + "\n" +
		"Patch Directory: " + config.PatchDir + "\n" +
		"Keys Directory: " + config.KeysDir + "\n" +
		"Database Host: " + config.DBHost + "\n" +
		"Database Port: " + config.DBPort + "\n" +
		"Database Name: " + config.DBName + "\n" +
		"Database Username: " + config.DBUsername + "\n" +
		"Database Password: " + config.DBPassword + "\n" +
		"Output Logged To: " + outfile + "\n" +
		"Logging Level: " + config.LogLevel + "\n" +
		"Debug Mode Enabled: " + strconv.FormatBool(config.DebugMode)
}
