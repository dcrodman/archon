/*
* Singleton package for handling the global server configuration
* and responsible for establishing a connection to the database
* to be maintained during execution.
 */
package archon

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strconv"
	"strings"
)

// Filesystem locations that will be checked for a config file by default.
var defaultSearchPaths = []string{
	".",
	"/usr/local/etc/archon/",
	"setup/",
}

var (
	// TODO: Remove these and put them closer to where they're actually used.
	cachedIPBytes [4]byte
)

func init() {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")

	for _, path := range defaultSearchPaths {
		viper.AddConfigPath(path)
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("unable to load config file. error: ", err)
		fmt.Printf("please check that one of these files exists and restart the server: %v\n", defaultSearchPaths)
		os.Exit(1)
	}
}

func ConfigFileUsed() string {
	return viper.ConfigFileUsed()
}

// Convert the broadcast IP string into 4 bytes to be used with the redirect packet.
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
