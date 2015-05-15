/*
* Archon Patch Server
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
package patch_server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"libarchon/logger"
	"libarchon/util"
	"os"
	"strconv"
	"strings"
)

const ServerConfigDir = "/usr/local/share/archon"
const patchConfigFile = "patch_config.json"

// Configuration structure that can be shared between the Patch and Data servers.
type configuration struct {
	Hostname       string
	PatchPort      string
	DataPort       string
	PatchDir       string
	WelcomeMessage string
	Logfile        string
	LogLevel       logger.LogPriority
	DebugMode      bool

	database        *sql.DB
	logWriter       io.Writer
	cachedHostBytes [4]byte
	redirectPort    uint16
	MessageBytes    []byte
	MessageSize     uint16
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
	config.PatchPort = "11000"
	config.DataPort = "11001"
	config.Logfile = "Standard Out"
	config.WelcomeMessage = "Unconfigured Welcome Message"

	json.Unmarshal(data, config)

	// Convert the welcome message to UTF-16LE and cache it.
	config.MessageBytes = util.ConvertToUtf16(config.WelcomeMessage)
	// PSOBB expects this prefix to the message, not completely sure why...
	config.MessageBytes = append([]byte{0xFF, 0xFE}, config.MessageBytes...)
	msgLen := len(config.MessageBytes)
	// Sanity check, the message can't exceed the max size of a packet.
	if msgLen > (1 << 16) {
		return errors.New("Message length must be less than 65,000 characters")
	}
	config.MessageSize = uint16(msgLen)

	if config.LogLevel < logger.LogPriorityCritical || config.LogLevel > logger.LogPriorityLow {
		// The log level must be at least open to critical messages.
		config.LogLevel = logger.LogPriorityCritical
	}
	// Convert the data port to a BE uint for the redirect packet.
	dataPort, _ := strconv.ParseUint(config.DataPort, 10, 16)
	config.redirectPort = uint16((dataPort >> 8) | (dataPort << 8))

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
	return "Hostname: " + config.Hostname + "\n" +
		"Patch Port: " + config.PatchPort + "\n" +
		"Data Port: " + config.DataPort + "\n" +
		"Patch Directory: " + config.PatchDir + "\n" +
		"Output Logged To: " + config.Logfile + "\n" +
		"Logging Level: " + strconv.FormatInt(int64(config.LogLevel), 10) + "\n" +
		"Debug Mode Enabled: " + strconv.FormatBool(config.DebugMode) + "\n" +
		"Welcome Message: " + config.WelcomeMessage
}
