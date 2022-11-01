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
// server components.
type Config struct {
	// Hostname or IP address on which the servers will listen for connections.
	Hostname string `mapstructure:"hostname"`
	// IP broadcast to clients in the redirect packets.
	ExternalIP string `mapstructure:"external_ip"`
	// Maximum number of concurrent connections the server will allow.
	MaxConnections int `mapstructure:"max_connections"`
	// Full path to file to which logs will be written. Blank will write to stdout.
	LogFilePath string `mapstructure:"log_file_path"`
	// Minimum level of a log required to be written. Options: debug, info, warn, error
	LogLevel string `mapstructure:"log_level"`
	// X.509 certificate for the shipgate server.
	ShipgateCertFile string `mapstructure:"shipgate_certificate_file"`

	Web struct {
		// HTTP endpoint port for publically accessible API endpoints.
		HTTPPort int `mapstructure:"http_port"`
	} `mapstructure:"web"`

	Database struct {
		// Hostname of the Postgres database instance.
		Host string `mapstructure:"host"`
		// Port on db_host on which the Postgres instance is accepting connections.
		Port int `mapstructure:"port"`
		// Name of the database in Postgres for archon.
		Name string `mapstructure:"name"`
		// Username and password of a user with full RW privileges to ${db_name}.
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		// Set to verify-full if the Postgres instance supports SSL.
		SSLMode string `mapstructure:"disable"`
	} `mapstructure:"database"`

	ShipgateServer struct {
		// Port on which the Shipgate's gRPC server will listen.
		Port int `mapstructure:"port"`
		// Private key file corresponding to shipgate_certificate_file (above).
		SSLKeyFile string `mapstructure:"ssl_key_file"`
	} `mapstructure:"shipgate_server"`

	PatchServer struct {
		// Port on whith the PATCH server will listen.
		PatchPort int `mapstructure:"patch_port"`
		// Port on which the patch DATA Server will listen.
		DataPort int `mapstructure:"data_port"`
		// Full (or relative to the current directory) path to the directory containing the patch files.
		PatchDir string `mapstructure:"patch_dir"`
		// Welcome message displayed on the patch screen.
		WelcomeMessage string `mapstructure:"welcome_message"`
	} `mapstructure:"patch_server"`

	LoginServer struct {
		// Port on which the LOGIN server will listen.
		Port int `mapstructure:"port"`
	} `mapstructure:"login_server"`

	CharacterServer struct {
		// Port on which the LOGIN server will listen.
		Port int `mapstructure:"port"`
		// Full (or relative to the current directory) path to the directory containing your
		// parameter files (defaults to /usr/local/etc/archon/parameters).
		ParametersDir string `mapstructure:"parameters_dir"`
		// Scrolling welcome message to display to the user on the ship selection screen.
		ScrollMessage string `mapstructure:"scroll_message"`
	} `mapstructure:"character_server"`

	ShipServer struct {
		// Port on which the SHIP server will listen.
		Port int `mapstructure:"port"`
		// Name of the ship that will appear in the selection screen.
		Name string `mapstructure:"name"`
		// Number of block servers to run for this ship.
		NumBlocks int `mapstructure:"num_blocks"`
	} `mapstructure:"ship_server"`

	BlockServer struct {
		// Base block port.
		Port int `mapstructure:"port"`
		// Number of lobbies to create per block.
		NumLobbies int `mapstructure:"num_lobbies"`
	} `mapstructure:"block_server"`

	Debugging struct {
		// Enable extra info-providing mechanisms for the server.
		PprofEnabled bool `mapstructure:"pprof_enabled"`
		// Port on which a pprof server will be started if debug mode is enabled.
		PprofPort int `mapstructure:"pprof_port"`
		// # Log packets to stdout.
		PacketLoggingEnabled bool `mapstructure:"packet_logging_enabled"`
		//  Enable database-level query logging.
		DatabaseLoggingEnabled bool `mapstructure:"database_logging_enabled"`
	} `mapstructure:"debugging"`

	cachedIPBytes [4]byte
}

const envVarPrefix = "ARCHON"

// LoadConfig initializes Viper with the contents of the config file under configPath.
func LoadConfig(configPath string) *Config {
	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

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
	return fmt.Sprintf("https://%s:%v", c.Hostname, c.ShipgateServer.Port)
}

// BroadcastIP converts the configured broadcast IP string into 4 bytes to be used
// with the redirect packet common to several servers.
func (c *Config) BroadcastIP() [4]byte {
	// Hacky, but chances are the IP address isn't going to start with 0 and a
	// fixed-length array can't be null.
	if c.cachedIPBytes[0] == 0x00 {
		parts := strings.Split(viper.GetString("external_ip"), ".")
		for i := 0; i < 4; i++ {
			tmp, _ := strconv.ParseUint(parts[i], 10, 8)
			c.cachedIPBytes[i] = uint8(tmp)
		}
	}
	return c.cachedIPBytes
}
