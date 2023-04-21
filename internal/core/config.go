package core

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config contains all of the configuration options available to any of Archon's
// server components. Descriptions are in config.yaml.
type Config struct {
	Hostname       string `mapstructure:"hostname"`
	ExternalIP     string `mapstructure:"external_ip"`
	MaxConnections int    `mapstructure:"max_connections"`
	LogFilePath    string `mapstructure:"log_file_path"`
	LogLevel       string `mapstructure:"log_level"`

	Web struct {
		HTTPPort int `mapstructure:"http_port"`
	} `mapstructure:"web"`

	Database struct {
		Engine   string `mapstructure:"engine"`
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Name     string `mapstructure:"name"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		SSLMode  string `mapstructure:"disable"`
	} `mapstructure:"database"`

	ShipgateServer struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"shipgate_server"`

	PatchServer struct {
		PatchPort      int    `mapstructure:"patch_port"`
		DataPort       int    `mapstructure:"data_port"`
		PatchDir       string `mapstructure:"patch_dir"`
		WelcomeMessage string `mapstructure:"welcome_message"`
	} `mapstructure:"patch_server"`

	LoginServer struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"login_server"`

	CharacterServer struct {
		Port int `mapstructure:"port"`
		// TODO: Restore this config option when adding support for overriding these files.
		// ParametersDir string `mapstructure:"parameters_dir"`
		ScrollMessage string `mapstructure:"scroll_message"`
	} `mapstructure:"character_server"`

	ShipServer struct {
		Port      int    `mapstructure:"port"`
		Name      string `mapstructure:"name"`
		NumBlocks int    `mapstructure:"num_blocks"`
	} `mapstructure:"ship_server"`

	BlockServer struct {
		Port       int `mapstructure:"port"`
		NumLobbies int `mapstructure:"num_lobbies"`
	} `mapstructure:"block_server"`

	Debugging struct {
		PprofEnabled           bool `mapstructure:"pprof_enabled"`
		PprofPort              int  `mapstructure:"pprof_port"`
		PacketLoggingEnabled   bool `mapstructure:"packet_logging_enabled"`
		DatabaseLoggingEnabled bool `mapstructure:"database_logging_enabled"`
	} `mapstructure:"debugging"`

	cachedIPBytes [4]byte
}

const envVarPrefix = "ARCHON"

// LoadConfig initializes Viper with the contents of the config file under configPath.
func LoadConfig(configPath string) *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	if configPath != "" {
		viper.AddConfigPath(configPath)
	}
	viper.AddConfigPath(".")

	viper.SetEnvPrefix(envVarPrefix)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if errors.Is(err, viper.ConfigFileNotFoundError{}) {
			fmt.Printf("error reading config file: no config file in path %s", configPath)
		} else {
			fmt.Printf("error reading config file: %v", err)
		}
		os.Exit(1)
	}

	// This allows us to set nested yaml config options through environment
	// variables. For example, database.host can be set using: <envVarPrefix>_DATABASE_HOST
	for _, k := range viper.AllKeys() {
		envVar := strings.ReplaceAll(strings.ToUpper(k), ".", "_")
		if err := viper.BindEnv(k, envVarPrefix+"_"+envVar); err != nil {
			fmt.Printf("error binding %s to %s", k, envVarPrefix+"_"+envVar)
			os.Exit(1)
		}
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		fmt.Printf("error unmrarshaling config object: %v", err)
		os.Exit(1)
	}
	return config
}

const databaseURITemplate = "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s"

// DatabaseURL returns a database URL generated from the provided config values.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		databaseURITemplate,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.Username,
		c.Database.Password,
		c.Database.SSLMode,
	)
}

// ShipgateAddress returns the fully qualified address of the ship server.
// TODO: Since the current expectation is that the ship server is running alongside
// the other servers, this just uses the hostname. This should be fixed at some point
// to use an actual configurable address and the ship server given its own listen config.
func (c *Config) ShipgateAddress() string {
	return fmt.Sprintf("http://%s:%v", c.Hostname, c.ShipgateServer.Port)
}

// BroadcastIP converts the configured broadcast IP string into 4 bytes to be used
// with the redirect packet common to several servers.
func (c *Config) BroadcastIP() [4]byte {
	// Hacky, but chances are the IP address isn't going to start with 0 and a
	// fixed-length array can't be null.
	if c.cachedIPBytes[0] == 0x00 {
		parts := strings.Split(c.ExternalIP, ".")
		for i := 0; i < 4; i++ {
			tmp, _ := strconv.ParseUint(parts[i], 10, 8)
			c.cachedIPBytes[i] = uint8(tmp)
		}
	}
	return c.cachedIPBytes
}
