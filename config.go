// Singleton package for handling the global server configuration
// and responsible for establishing a connection to the database
// to be maintained during execution.
package archon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

var (
	// TODO: Remove these and put them closer to where they're actually used.
	cachedIPBytes [4]byte
)

// Load initializes Viper with the contents of file.
func Load(file string) {
	viper.AddConfigPath(filepath.Dir(file))
	viper.SetConfigName(filepath.Base(file))
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("error reading config file: file not found", file)
		} else {
			fmt.Println("error reading config file", err)
		}
		os.Exit(1)
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
