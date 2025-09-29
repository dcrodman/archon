package internal

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/dcrodman/archon/internal/character"
	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/debug"
	"github.com/dcrodman/archon/internal/patch"
	"github.com/dcrodman/archon/internal/ship"
	"github.com/dcrodman/archon/internal/shipgate"
)

// Controller is the main entrypoint for archon. It's responsible for initializing
// any shared resources (such as database and logging), defining the servers, and
// launching everything.
type Controller struct {
	Config *core.Config

	logger *zap.SugaredLogger
	wg     sync.WaitGroup

	servers []*frontend
}

func (c *Controller) Start(ctx context.Context) {
	defer c.Shutdown(ctx)

	var err error
	// Set up the logger, which will be used by all sub-servers.
	c.logger, err = core.NewLogger(c.Config)
	if err != nil {
		fmt.Println("error initializing logger:", err)
		return
	}

	// Start any debug utilities if we're configured to do so.
	if c.Config.Debugging.PacketLoggingEnabled {
		debug.StartPprofServer(c.logger, c.Config.Debugging.PprofPort)
	}

	// Start the shipgate RPC service and make sure it launches before the other servers start.
	shipgate.Start(c.Config, c.logger)

	shipgateAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%d", c.Config.ShipgateServer.Port))
	if err != nil {
		c.logger.Errorf("error resolving shipgate address: %v", err)
		return
	}
	t := time.NewTimer(30 * time.Second)
	for {
		select {
		case <-t.C:
			c.logger.Errorf("timed out waiting for shipgate to initialize")
			return
		default:
		}

		conn, err := net.DialTCP("tcp", nil, shipgateAddr)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(time.Second)
	}

	// Configure, initialize, run all of our servers.
	c.createServers()
	c.run(ctx)
}

// Set up all of the client-facing servers we'll be running.
func (c *Controller) createServers() {
	c.servers = []*frontend{
		{
			Address: c.buildAddress(c.Config.PatchServer.AuthPort),
			Backend: &patch.PatchAuthServer{
				Config: c.Config,
				Logger: c.logger,
			},
		},
		{
			Address: c.buildAddress(c.Config.PatchServer.DataPort),
			Backend: &patch.PatchDataServer{
				Config: c.Config,
				Logger: c.logger,
			},
		},
		{
			Address: c.buildAddress(c.Config.CharacterServer.AuthPort),
			Backend: &character.AuthServer{
				Config: c.Config,
				Logger: c.logger,
			},
		},
		{
			Address: c.buildAddress(c.Config.CharacterServer.DataPort),
			Backend: &character.Server{
				Config: c.Config,
				Logger: c.logger,
			},
		},
		// Note: Eventually the ship and block servers should be able to be run
		// independently of the other four servers
		{
			Address: c.buildAddress(c.Config.ShipServer.AuthPort),
			Backend: &ship.AuthServer{
				Config: c.Config,
				Logger: c.logger,
			},
		},
		{
			Address: c.buildAddress(c.Config.ShipServer.GamePort),
			Backend: &ship.GameServer{
				Config: c.Config,
				Logger: c.logger,
			},
		},
	}
}

func (c *Controller) run(ctx context.Context) {
	for _, server := range c.servers {
		server.Config = c.Config
		server.Logger = c.logger

		if err := server.Backend.Init(ctx); err != nil {
			c.logger.Errorf("error initializing %s server: %v", server.Backend.Identifier(), err)
			return
		}
	}
	for _, server := range c.servers {
		if err := server.Start(ctx, &c.wg); err != nil {
			c.logger.Errorf("error starting %s server: %v", server.Backend.Identifier(), err)
			return
		}
	}

	c.wg.Wait()
}

func (c *Controller) buildAddress(port int) string {
	return fmt.Sprintf("%s:%v", c.Config.Hostname, port)
}

func (c *Controller) Shutdown(ctx context.Context) {
	// Stop the shipgate after all of the other servers have stopped in order to avoid
	// errors from any shipgate calls during the shutdown process.
	c.wg.Wait()
	shipgate.Shutdown(ctx)
}
