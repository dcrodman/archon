package archon

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const defaultConfigName = "config"
const overrideConfigName = "override"

var (
	// TODO: Remove these and put them closer to where they're actually used.
	cachedIPBytes [4]byte
)

// LoadConfig initializes Viper with the contents of the config file under configPath.
func LoadConfig(configPath string) {
	viper.AddConfigPath(configPath)
	viper.SetConfigName(defaultConfigName)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Printf("error reading config file: no config file in path %s", configPath)
		} else {
			fmt.Printf("error reading config file: %v", err)
		}
		os.Exit(1)
	}

	// Attempt to load any configs from the override file if present.
	viper.SetConfigName(overrideConfigName)
	_ = viper.MergeInConfig()
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
