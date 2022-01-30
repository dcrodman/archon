package archon

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

var (
	// TODO: Remove these and put them closer to where they're actually used.
	cachedIPBytes [4]byte
)

// LoadConfig initializes Viper with the contents of the config file under configPath.
func LoadConfig(configPath string) {
	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	envPrefix := "ARCHON"
	// Config values
	viper.SetEnvPrefix(envPrefix)
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
	// variables. For example:
	// database.host can be set using: <envPrefix>_DATABASE_HOST
	for _, k := range viper.AllKeys() {
		envVar := strings.ReplaceAll(strings.ToUpper(k), ".", "_")
		if err := viper.BindEnv(k, envPrefix+"_"+envVar); err != nil {
			fmt.Printf("error binding %s to %s", k, envPrefix+"_"+envVar)
			os.Exit(1)
		}
	}
}

// BroadcastIP converts the configured broadcast IP string into 4 bytes to be used
// with the redirect packet common to several servers.
func BroadcastIP() [4]byte {
	// Hacky, but chances are the IP address isn't going to start with 0 and a
	// fixed-length array can't be null.
	if cachedIPBytes[0] == 0x00 {
		parts := strings.Split(viper.GetString("external_ip"), ".")
		for i := 0; i < 4; i++ {
			tmp, _ := strconv.ParseUint(parts[i], 10, 8)
			cachedIPBytes[i] = uint8(tmp)
		}
	}
	return cachedIPBytes
}
