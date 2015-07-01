/*
* Archon Shipgate Server
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
package shipgate_server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"libarchon/logger"
	"os"
	"strconv"
)

const ServerConfigDir = "/usr/local/share/archon"
const ShipgateConfigFile = "shipgate_config.json"

// Configuration structure that can be shared between the Login and Character servers.
type configuration struct {
	Hostname       string
	ShipgatePort   string
	WebPort        string
	KeyDirectory   string
	MaxConnections int
	Logfile        string
	LogLevel       logger.LogPriority
	DebugMode      bool

	database  *sql.DB
	logWriter io.Writer
}

// Singleton instance.
var shipgateConfig *configuration = nil

// This function should be used to get access to the server config instead of directly
// referencing the loginConfig pointer.
func GetConfig() *configuration {
	if shipgateConfig == nil {
		shipgateConfig = new(configuration)
	}
	return shipgateConfig
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
	config.ShipgatePort = "13000"
	config.WebPort = "13001"
	config.KeyDirectory = "keys"
	config.MaxConnections = 30000
	config.Logfile = "Standard Out"

	json.Unmarshal(data, config)

	if config.LogLevel < logger.CriticalPriority || config.LogLevel > logger.LowPriority {
		// The log level must be at least open to critical messages.
		config.LogLevel = logger.CriticalPriority
	}

	if config.Logfile != "Standard Out" {
		config.logWriter, err = os.OpenFile(config.Logfile,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("\nWARNING: Failed to open log file %s: %s\n",
				config.Logfile, err.Error())
		}
	} else {
		config.logWriter = os.Stdout
	}
	return nil
}

func (config *configuration) String() string {
	return "Hostname: " + config.Hostname + "\n" +
		"Shipgate Port: " + config.ShipgatePort + "\n" +
		"Web Port: " + config.WebPort + "\n" +
		"Key Directory: " + config.KeyDirectory + "\n" +
		"Max Connections: " + strconv.FormatInt(int64(config.MaxConnections), 10) + "\n" +
		"Output Logged To: " + config.Logfile + "\n" +
		"Logging Level: " + strconv.FormatInt(int64(config.LogLevel), 10) + "\n" +
		"Debug Mode Enabled: " + strconv.FormatBool(config.DebugMode)
}
