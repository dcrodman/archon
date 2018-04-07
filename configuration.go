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
	"errors"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dcrodman/archon/util"
	"gopkg.in/yaml.v2"
)

// DatabaseConfig contains all parameters for db initialization.
type DatabaseConfig struct {
	DBHost     string `yaml:"db_host"`
	DBPort     string `yaml:"db_port"`
	DBName     string `yaml:"db_name"`
	DBUsername string `yaml:"db_username"`
	DBPassword string `yaml:"db_password"`
}

// PatchConfig contains all parameters for the patch server.
type PatchConfig struct {
	PatchPort string `yaml:"patch_port"`
	DataPort  string `yaml:"data_port"`
	PatchDir  string `yaml:"patch_dir"`
	// Message displayed on the welcome screen.
	WelcomeMessage string `yaml:"welcome_message"`
}

// LoginConfig contains all parameters for the login server.
type LoginConfig struct {
	LoginPort     string `yaml:"login_port"`
	CharacterPort string `yaml:"character_port"`
	ParametersDir string `yaml:"parameters_dir"`
	// Scrolling message on ship select.
	ScrollMessage string `yaml:"scroll_message"`
}

// ShipConfig contains all parameters for the ship server.
type ShipConfig struct {
	ShipPort string `yaml:"ship_port"`
	ShipName string `yaml:"ship_name"`
	// Number of blocks to open on the ship server.
	NumBlocks int `yaml:"num_blocks"`
}

// BlockConfig contains all parameters for the block server(s).
type BlockConfig struct {
	BlockPort string `yaml:"block_port"`
	// Number of lobbies available per block.
	NumLobbies int `yaml:"num_lobbies"`
}

// ShipgateConfig contains all parameters for the shipgate.
type ShipgateConfig struct {
	ShipgatePort string `yaml:"shipgate_port"`
}

// WebConfig contains all parameters for the external HTTP server,
// which is used to expose server status and other metadata to external
// callers. This can be disabled.
type WebConfig struct {
	WebPort string `yaml:"http_port"`
}

// Configuration structure that can be shared between sub servers.
// The fields are intentionally exported to cut down on verbosity
// with the intent that they be considered immutable.
type Config struct {
	Hostname       string `yaml:"hostname"`
	ExternalIP     string `yaml:"external_ip"`
	MaxConnections int    `yaml:"max_connections"`
	Logfile        string `yaml:"log_file"`
	LogLevel       string `yaml:"log_level"`
	DebugMode      bool   `yaml:"debug_mode"`

	DatabaseConfig `yaml:"database"`
	PatchConfig    `yaml:"patch_server"`
	LoginConfig    `yaml:"login_server"`
	ShipConfig     `yaml:"ship_server"`
	BlockConfig    `yaml:"block_server"`
	ShipgateConfig `yaml:"shipgate_server"`
	WebConfig      `yaml:"web"`

	cachedIPBytes   [4]byte
	MessageBytes    []byte
	MessageSize     uint16
	cachedScrollMsg []byte
}

// Singleton instance. Provides reasonable default values so
// that some configurations can remain simpler.
var config *Config = &Config{
	Hostname:       "127.0.0.1",
	ExternalIP:     "127.0.0.1",
	Logfile:        "",
	LogLevel:       "warn",
	DebugMode:      false,
	MaxConnections: 30000,
	DatabaseConfig: DatabaseConfig{
		DBHost: "127.0.0.1",
		DBPort: "3306",
		DBName: "archondb",
	},
	PatchConfig: PatchConfig{
		PatchPort:      "11000",
		DataPort:       "11001",
		PatchDir:       "patches/",
		WelcomeMessage: "Unconfigured Welcome Message",
	},
	LoginConfig: LoginConfig{
		LoginPort:     "12000",
		CharacterPort: "12001",
		ParametersDir: "parameters/",
		ScrollMessage: "Add a welcome message here",
	},
	ShipConfig: ShipConfig{
		ShipPort:  "15000",
		ShipName:  "Unconfigured",
		NumBlocks: 2,
	},
	BlockConfig: BlockConfig{
		NumLobbies: 15,
	},
	ShipgateConfig: ShipgateConfig{
		ShipgatePort: "13000",
	},
	WebConfig: WebConfig{
		WebPort: "14000",
	},
}

// GetConfig returns the singleton instance of the config struct containing all of
// the configuration data read from our YAML file.
func GetConfig() *Config {
	return config
}

// Populate config with the contents of a JSON file at path fileName. Config parameters
// in the file must match the above fields exactly in order to be read.
func (config *Config) InitFromFile(fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	if err = yaml.Unmarshal(data, config); err != nil {
		return errors.New("Failed to parse config file: " + err.Error())
	}

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

// Convert the broadcast IP string into 4 bytes to be used with the redirect packet.
func (config *Config) BroadcastIP() [4]byte {
	// Hacky, but chances are the IP address isn't going to start with 0 and a
	// fixed-length array can't be null.
	if config.cachedIPBytes[0] == 0x00 {
		parts := strings.Split(config.ExternalIP, ".")
		for i := 0; i < 4; i++ {
			tmp, _ := strconv.ParseUint(parts[i], 10, 8)
			config.cachedIPBytes[i] = uint8(tmp)
		}
	}
	return config.cachedIPBytes
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
		"Debug Mode Enabled: " + strconv.FormatBool(config.DebugMode) + "\n" +
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
		"Database Host: " + config.DBHost + "\n" +
		"Database Port: " + config.DBPort + "\n" +
		"Database Name: " + config.DBName + "\n" +
		"Database Username: " + config.DBUsername + "\n" +
		"Database Password: " + config.DBPassword + "\n" +
		"Output Logged To: " + outfile + "\n" +
		"Logging Level: " + config.LogLevel
}
