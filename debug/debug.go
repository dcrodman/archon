package debug

import (
	"fmt"
	"github.com/spf13/viper"
	"net/http"
	"runtime/pprof"
)

// Enabled returns whether or not the server was set to debug mode.
func Enabled() bool {
	return viper.GetBool("debug_mode")
}

// If the server was configured in debug mode, this function will launch an HTTP server
// that responds with pprof output containing the stack traces of all running goroutines.
func StartPprofServer() {
	webPort := viper.GetString("web.http_port")

	fmt.Println("opening debug port on " + webPort)
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		pprof.Lookup("goroutine").WriteTo(resp, 1)
	})

	http.ListenAndServe(":"+webPort, nil)
}
