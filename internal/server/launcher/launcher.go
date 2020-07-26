package launcher

import (
	"context"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/server"
	"net"
	"os"
	"sync"
)

var defaultLauncher Launcher

type handler struct {
	backend server.Backend
	port    string
	addr    *net.TCPAddr
}

// Launcher manages the association between Backends and their port bindings
// on a specified hostname.
type Launcher struct {
	hostname string
	servers  []handler

	addressOverrides map[server.Backend]*net.TCPAddr
}

// SetHostname sets the global hostname to which all registered servers will be bound.
func SerHostname(hostname string) { defaultLauncher.SetHostname(hostname) }
func (l *Launcher) SetHostname(hostname string) {
	l.hostname = hostname
}

// AddServer registers a Backend server instance to a port.
func AddServer(port string, backend server.Backend) { defaultLauncher.AddServer(port, backend) }
func (l *Launcher) AddServer(port string, backend server.Backend) {
	l.servers = append(l.servers, handler{port: port, backend: backend})
}

func (l *Launcher) AddServerWithAddress(addr *net.TCPAddr, backend server.Backend) {
	l.servers = append(l.servers, handler{addr: addr, backend: backend})
}

// Start initializes all of the Backends and starts the set of registered servers
// concurrently, returning a sync.WaitGroup that can be observed to avoid exiting
// until all servers have shut down.
func Start(ctx context.Context) *sync.WaitGroup { return defaultLauncher.Start(ctx) }
func (l *Launcher) Start(ctx context.Context) *sync.WaitGroup {
	if l.hostname == "" {
		panic("error initializing server: no hostname set")
	}

	l.initServers()

	var wg sync.WaitGroup
	for _, s := range l.servers {
		addr := s.addr
		if addr == nil {
			var err error
			addr, err = l.buildAddr(s.port)
			if err != nil {
				archon.Log.Error(err)
				os.Exit(1)
			}
		}

		l.startServer(ctx, &wg, addr, s.backend)
	}
	return &wg
}

func (l *Launcher) initServers() {
	for _, s := range l.servers {
		// Failure to initialize one of the registered servers is considered terminal.
		if err := s.backend.Init(); err != nil {
			archon.Log.Error("failed to initialize %s server: %s\n", s.backend.Name(), err)
			os.Exit(1)
		}
	}

	// Minor hack to visually separate any server init messages from the startup output.
	fmt.Println()
}

func (l *Launcher) buildAddr(port string) (*net.TCPAddr, error) {
	hostAddr, err := net.ResolveTCPAddr("tcp", l.hostname+":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address %s", err.Error())
	}

	return hostAddr, nil
}

func (l *Launcher) startServer(ctx context.Context, wg *sync.WaitGroup, addr *net.TCPAddr, backend server.Backend) {
	go func() {
		frontend := newFrontend(addr, backend)

		// Failure to start one of the registered servers is considered terminal.
		if err := frontend.StartListening(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		wg.Done()
	}()
}
