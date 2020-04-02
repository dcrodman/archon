/*
* Singleton package for handling the global server configuration
* and responsible for establishing a connection to the database
* to be maintained during execution.
 */
package archon

import (
	"fmt"
	"github.com/dcrodman/archon/util"
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
	cachedIPBytes   [4]byte
	MessageBytes    []byte
	MessageSize     uint16
	cachedScrollMsg []byte
)

// Configuration structure that can be shared between sub servers.
// The fields are intentionally exported to cut down on verbosity
// with the intent that they be considered immutable.
//var Config = struct {
//	Hostname       string
//	ExternalIP     string
//	MaxConnections int
//	Logfile        string
//	LogLevel       string
//	DebugMode      bool
//
//	Database struct {
//		Host     string
//		Port     string
//		Name     string
//		Username string
//		Password string
//	}
//
//	PatchServer struct {
//		PatchPort      string
//		DataPort       string
//		PatchDir       string
//		WelcomeMessage string
//	}
//
//	LoginServer struct {
//		LoginPort     string
//		CharacterPort string
//		ParametersDir string
//		ScrollMessage string
//	}
//
//	ShipServer struct {
//		Port      string
//		Name      string
//		NumBlocks int
//	}
//
//	BlockServer struct {
//		BasePort   string
//		NumLobbies int
//	}
//
//	ShipgateServer struct {
//		Port string
//	}
//
//	// WebConfig contains all parameters for the external HTTP server,
//	// which is used to expose server status and other metadata to external
//	// callers. This can be disabled.
//	WebServer struct {
//		Port string
//	}
//}{}

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

	// Convert the welcome message to UTF-16LE and cache it. PSOBB expects this prefix to the message,
	//not completely sure why. Language perhaps?
	// TODO: Move this into the patch server?
	MessageBytes = util.ConvertToUtf16(viper.GetString("patch_server.welcome_message"))
	MessageBytes = append([]byte{0xFF, 0xFE}, MessageBytes...)
	MessageSize = uint16(len(MessageBytes))

	if MessageSize > (1<<16 - 16) {
		fmt.Println("error: message length must be less than 65,000 characters")
		os.Exit(1)
	}

	cachedScrollMsg = util.ConvertToUtf16(viper.GetString("login_server.scroll_message"))
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
