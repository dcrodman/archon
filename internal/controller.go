package internal

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/block"
	"github.com/dcrodman/archon/internal/character"
	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/data"
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

	wg      sync.WaitGroup
	servers []*frontend
}

func (c *Controller) Start(ctx context.Context) error {
	defer c.Shutdown()

	archon.InitLogger()

	// Connect to the database.
	if err := data.Initialize(c.Config.DatabaseURL(), c.Config.Debugging.Enabled); err != nil {
		return err
	}
	archon.Log.Infof("connected to database %s:%d", c.Config.Database.Host, c.Config.Database.Port)

	// Start any debug utilities if we're configured to do so.
	if c.Config.Debugging.Enabled {
		debug.StartUtilities(c.Config.Debugging.PprofPort, c.Config.Debugging.PacketAnalyzerAddress)
	}

	// Start the shipgate gRPC server and make sure it launches before the other servers start.
	c.startShipgate(ctx)

	// Configure and run all of our servers.
	c.declareServers()
	return c.run(ctx)
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
			},
		},
		{
			Address: c.buildAddress(c.Config.PatchServer.DataPort),
			Backend: &patch.DataServer{
				Name:   "DATA",
				Config: c.Config,
			},
		},
		{
			Address: c.buildAddress(c.Config.LoginServer.Port),
			Backend: &login.Server{
				Name:   "LOGIN",
				Config: c.Config,
			},
		},
		{
			Address: c.buildAddress(c.Config.CharacterServer.Port),
			Backend: &character.Server{
				Name:   "CHARACTER",
				Config: c.Config,
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
			},
		},
	}

	c.servers = append(c.servers, blockServers...)
}

func (c *Controller) run(ctx context.Context) error {
	// Start all of our servers. Failure to initialize one of the registered servers is considered terminal.
	for _, server := range c.servers {
		server.Config = c.Config
		if err := server.Start(ctx, &c.wg); err != nil {
			return fmt.Errorf("error starting %s server: %w", server.Backend.Identifier(), err)
		}
	}

	c.wg.Wait()
	return nil
}

func (c *Controller) startShipgate(ctx context.Context) {
	readyChan := make(chan bool)
	errChan := make(chan error)

	go shipgate.Start(ctx, c.buildAddress(c.Config.ShipgateServer.Port), readyChan, errChan)
	go func() {
		if err := <-errChan; err != nil {
			fmt.Printf("exiting due to SHIPGATE error: %v", err)
			os.Exit(1)
		}
	}()

	<-readyChan
}

func (c *Controller) buildAddress(port int) string {
	return fmt.Sprintf("%s:%v", c.Config.Hostname, port)
}

func (c *Controller) Shutdown() {
	data.Shutdown()
}
