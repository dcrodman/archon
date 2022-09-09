package internal

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dcrodman/archon/internal/block"
	"github.com/dcrodman/archon/internal/character"
	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/debug"
	"github.com/dcrodman/archon/internal/login"
	"github.com/dcrodman/archon/internal/patch"
	"github.com/dcrodman/archon/internal/ship"
	"github.com/dcrodman/archon/internal/shipgate"
)

// Controller is the main entrypoint for archon. It's responsible for initializing
// any shared resources (such as database and logging), defining the servers, and
// launching everything.
type Controller struct {
	Config *core.Config

	logger *logrus.Logger
	wg     sync.WaitGroup

	shipgateServer *shipgate.Server
	servers        []*frontend
}

func (c *Controller) Start(ctx context.Context) {
	defer c.Shutdown(ctx)

	var err error
	// Set up the logger, which will be used by all sub-servers.
	c.logger, err = core.NewLogger(c.Config)
	if err != nil {
		c.logger.Errorf("error initializing logger: %v", err)
		return
	}

	// Start any debug utilities if we're configured to do so.
	if c.Config.Debugging.Enabled {
		debug.StartUtilities(c.logger,
			c.Config.Debugging.PprofPort,
			c.Config.Debugging.PacketAnalyzerAddress,
		)
	}

	// Start the shipgate RPC service and make sure it launches before the other servers start.
	c.shipgateServer = &shipgate.Server{Config: c.Config, Logger: c.logger}
	c.shipgateServer.Start(ctx)

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

	// Configure and run all of our servers.
	c.declareServers()
	c.run(ctx)
}

// Set up all of the servers we want to run.
func (c *Controller) declareServers() {
	// Automatically configure the block servers based on the number of
	// ship blocks requested.
	var blocks []ship.Block
	var blockServers []*frontend
	for i := 1; i <= c.Config.ShipServer.NumBlocks; i++ {
		name := fmt.Sprintf("BLOCK%02d", i)
		address := c.buildAddress(c.Config.BlockServer.Port + i)

		blocks = append(blocks, ship.Block{
			Name: name, Address: address, ID: i,
		})
		blockServer := &frontend{
			Address: address,
			Backend: &block.Server{
				Name:   name,
				Config: c.Config,
				Logger: c.logger,
			},
		}
		blockServers = append(blockServers, blockServer)
	}

	c.servers = []*frontend{
		{
			Address: c.buildAddress(c.Config.PatchServer.PatchPort),
			Backend: &patch.Server{
				Name:   "PATCH",
				Config: c.Config,
				Logger: c.logger,
			},
		},
		{
			Address: c.buildAddress(c.Config.PatchServer.DataPort),
			Backend: &patch.DataServer{
				Name:   "DATA",
				Config: c.Config,
				Logger: c.logger,
			},
		},
		{
			Address: c.buildAddress(c.Config.LoginServer.Port),
			Backend: &login.Server{
				Name:   "LOGIN",
				Config: c.Config,
				Logger: c.logger,
			},
		},
		{
			Address: c.buildAddress(c.Config.CharacterServer.Port),
			Backend: &character.Server{
				Name:   "CHARACTER",
				Config: c.Config,
				Logger: c.logger,
			},
		},
		// Note: Eventually the ship and block servers should be able to be run
		// independently of the other four servers
		{
			Address: c.buildAddress(c.Config.ShipServer.Port),
			Backend: &ship.Server{
				Name:   "SHIP",
				Config: c.Config,
				Blocks: blocks,
				Logger: c.logger,
			},
		},
	}

	c.servers = append(c.servers, blockServers...)
}

func (c *Controller) run(ctx context.Context) {
	// Start all of our servers. Failure to initialize one of the registered servers is considered terminal.
	for _, server := range c.servers {
		server.Config = c.Config
		server.Logger = c.logger

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
	c.shipgateServer.Shutdown(ctx)
}
