package launcher_test

import (
	"context"
	"flag"
	"github.com/dcrodman/archon/internal/server"
	"github.com/dcrodman/archon/internal/server/launcher"
	"github.com/dcrodman/archon/internal/server/patch"
	"net"
	"sync"
	"testing"
)

// Allow the OS to choose the port for us.
const testPort = "0"

var numConnections = flag.Int("numConnections", 10, "Number of connections to test per backend")

func TestLauncher(t *testing.T) {
	backends := []server.Backend{
		patch.NewServer("Patch", testPort),
		patch.NewServer("Data", testPort),
	}

	l, cancel := startServers(backends)
	defer cancel()

	wg := sync.WaitGroup{}
	for _, f := range l.GetFrontends() {
		t.Run(f.Name(), func(t *testing.T) {
			for i := 0; i < *numConnections; i++ {
				wg.Add(1)
				go testConnection(t, &wg, f.Addr())
			}
		})

	}
	wg.Wait()
}

func startServers(backends []server.Backend) (*launcher.Launcher, func()) {
	l := &launcher.Launcher{}
	l.SetHostname("localhost")

	for _, backend := range backends {
		l.AddServer(testPort, backend)
	}

	ctx, cancel := context.WithCancel(context.Background())
	l.Start(ctx)

	return l, cancel
}

func testConnection(t *testing.T, wg *sync.WaitGroup, addr net.Addr) {
	conn, err := net.Dial(addr.Network(), addr.String())
	if err != nil {
		t.Error("failed to connect to", addr.String())
		return
	}

	data := make([]byte, 256)
	if _, err := conn.Read(data); err != nil {
		t.Error("failed to read from connection:", err)
		return
	}

	if _, err := conn.Write(data); err != nil {
		t.Error("failed to write to connection:", err)
		return
	}

	if err := conn.Close(); err != nil {
		t.Error("failed to close connection:", err)
	}

	wg.Done()
}
