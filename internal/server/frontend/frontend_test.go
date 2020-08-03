package frontend

import (
	"context"
	"flag"
	"github.com/dcrodman/archon/internal/server"
	"github.com/dcrodman/archon/internal/server/patch"
	"net"
	"sync"
	"testing"
)

// Allow the OS to choose the port for us.
const testPort = "0"

var numConnections = flag.Int("numConnections", 10, "Number of connections to test per backend")

func TestFrontend(t *testing.T) {
	backends := []server.Backend{
		patch.NewServer("Patch", testPort),
		patch.NewServer("Data", testPort),
	}

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, backend := range backends {
		t.Run(backend.Name(), func(t *testing.T) {
			addr, err := net.ResolveTCPAddr("tcp", "localhost:"+testPort)
			if err != nil {
				t.Fatal("failed to resolve address:", err)
			}

			f := frontend{
				addr:    addr,
				backend: backend,
			}

			if err := f.StartListening(ctx); err != nil {
				t.Fatal("failed to start frontend:", err)
			}

			for i := 0; i < *numConnections; i++ {
				wg.Add(1)
				go testConnection(t, &wg, f.addr)
			}
		})
	}

	wg.Wait()
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
