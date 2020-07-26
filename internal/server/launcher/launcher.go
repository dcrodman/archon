package launcher

import (
	"fmt"
	"github.com/dcrodman/archon/internal/server"
	"log"
	"os"
	"sync"
)

var defaultLauncher Launcher

type handler struct {
	port    string
	backend server.Backend
}

// Launcher manages the association between Backends and their port bindings
// on a specified hostname.
type Launcher struct {
	hostname string
	servers  []handler
}

// SetHostname sets the global hostname to which all registered servers will be bound.
func SerHostname(hostname string) { defaultLauncher.SerHostname(hostname) }
func (l *Launcher) SerHostname(hostname string) {
	l.hostname = hostname
}

// AddServer registers a Backend server instance to a port.
func AddServer(port string, backend server.Backend) { defaultLauncher.AddServer(port, backend) }
func (l *Launcher) AddServer(port string, backend server.Backend) {
	l.servers = append(l.servers, handler{port: port, backend: backend})
}

// Start initializes all of the Backends and starts the set of registered servers
// concurrently, returning a sync.WaitGroup that can be observed to avoid exiting
// until all servers have shut down.
func Start() *sync.WaitGroup { return defaultLauncher.Start() }
func (l *Launcher) Start() *sync.WaitGroup {
	if l.hostname == "" {
		panic("error initializing server: no hostname set")
	}

	for _, s := range l.servers {
		// Failure to initialize one of the registered servers is considered terminal.
		if err := s.backend.Init(); err != nil {
			log.Printf("failed to initialize %s server: %s\n", s.backend.Name(), err)
			os.Exit(1)
		}
	}
	// Minor hack to visually separate any server init messages from the startup output.
	fmt.Println()

	var wg sync.WaitGroup

	for _, s := range l.servers {
		go func(p string, b server.Backend) {
			frontend := newServerFrontend(l.hostname, p, b)

			// Failure to start one of the registered servers is considered terminal.
			if err := frontend.StartListening(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			wg.Done()
		}(s.port, s.backend)
	}

	return &wg
}
