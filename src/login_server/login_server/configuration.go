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
* Singleton package for handling the login and character server configuration. Also
* responsible for establishing a connection to the database to be maintained
* during execution.
 */
package login_server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "go-sql-driver"
	"io/ioutil"
	"libarchon/logger"
	"libarchon/util"
	"strconv"
	"strings"
)

const (
	ServerConfigDir = "/usr/local/share/archon"
	LoginConfigFile = "login_config.json"
	CertificateFile = "certificate.pem"
	KeyFile         = "key.pem"
)

// Configuration structure that can be shared between the Login and Character servers.
type configuration struct {
	Hostname       string
	LoginPort      string
	CharacterPort  string
	ShipgatePort   string
	WebPort        string
	WelcomeMessage string

	MaxConnections int
	ParametersDir  string
	KeysDir        string
	DBHost         string
	DBPort         string
	DBName         string
	DBUsername     string
	DBPassword     string
	Logfile        string
	LogLevel       logger.LogPriority
	DebugMode      bool

	database         *sql.DB
	cachedHostBytes  [4]byte
	cachedWelcomeMsg []byte
	redirectPort     uint16
}

// Singleton instance.
var loginConfig *configuration = nil

// This function should be used to get access to the server config instead of directly
// referencing the loginConfig pointer.
func GetConfig() *configuration {
	if loginConfig == nil {
		loginConfig = new(configuration)
	}
	return loginConfig
}

// Populate config with the contents of a JSON file at path fileName. Config parameters
// in the file must match the above fields exactly in order to be read.
func (config *configuration) InitFromFile(fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	// Provide default values for fields that are optional or critical.
	config.Hostname = "127.0.0.1"
	config.LoginPort = "12000"
	config.CharacterPort = "12001"
	config.ShipgatePort = "13000"
	config.WebPort = "14000"
	config.WelcomeMessage = "Add a welcome message here"
	config.MaxConnections = 30000
	config.ParametersDir = "parameters"
	config.KeysDir = "keys"
	config.DBHost = "127.0.0.1"
	config.Logfile = "Standard Out"

	json.Unmarshal(data, config)

	config.cachedWelcomeMsg = util.ConvertToUtf16(config.WelcomeMessage)

	if config.LogLevel < logger.High || config.LogLevel > logger.Low {
		// The log level must be at least open to critical messages.
		config.LogLevel = logger.High
	}

	charPort, _ := strconv.ParseUint(config.CharacterPort, 10, 16)
	config.redirectPort = uint16(charPort)

	return nil
}

// Establish a connection to the database and ping it to verify.
func (config *configuration) InitDb() error {
	dbName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.DBUsername,
		config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	var err error
	config.database, err = sql.Open("mysql", dbName)
	if err == nil {
		err = config.database.Ping()
	}
	return err
}

func (config *configuration) CloseDb() {
	config.database.Close()
}

// Use this function to obtain a reference to the database so that it can remain
// encapsulated and any consistency checks can be centralized.
func (config *configuration) Database() *sql.DB {
	if config.database == nil {
		// Don't implicitly initialize the database - if there's an error or other action that causes
		// the reference to become nil then we're probably leaking a connection.
		panic("Attempt to reference uninitialized database")
	}
	return config.database
}

// Convert the hostname string into 4 bytes to be used with the redirect packet.
func (config *configuration) HostnameBytes() [4]byte {
	// Hacky, but chances are the IP address isn't going to start with 0 and a
	// fixed-length array can't be null.
	if config.cachedHostBytes[0] == 0x00 {
		parts := strings.Split(config.Hostname, ".")
		for i := 0; i < 4; i++ {
			tmp, _ := strconv.ParseUint(parts[i], 10, 8)
			config.cachedHostBytes[i] = uint8(tmp)
		}
	}
	return config.cachedHostBytes
}

// Convenience method; returns a uint16 representation of the Character port.
func (config *configuration) RedirectPort() uint16 {
	return config.redirectPort
}

func (config *configuration) String() string {
	outfile := config.Logfile
	if outfile == "" {
		outfile = "Standard Out"
	}
	return "Hostname: " + config.Hostname + "\n" +
		"Login Port: " + config.LoginPort + "\n" +
		"Character Port: " + config.CharacterPort + "\n" +
		"Shipgate Port: " + config.ShipgatePort + "\n" +
		"Web Port: " + config.WebPort + "\n" +
		"Max Connections: " + strconv.FormatInt(int64(config.MaxConnections), 10) + "\n" +
		"Parameters Directory: " + config.ParametersDir + "\n" +
		"Keys Directory: " + config.KeysDir + "\n" +
		"Database Host: " + config.DBHost + "\n" +
		"Database Port: " + config.DBPort + "\n" +
		"Database Name: " + config.DBName + "\n" +
		"Database Username: " + config.DBUsername + "\n" +
		"Database Password: " + config.DBPassword + "\n" +
		"Output Logged To: " + outfile + "\n" +
		"Logging Level: " + strconv.FormatInt(int64(config.LogLevel), 10) + "\n" +
		"Debug Mode Enabled: " + strconv.FormatBool(config.DebugMode)
}
