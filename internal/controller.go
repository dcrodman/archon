package internal

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/viper"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/block"
	"github.com/dcrodman/archon/internal/character"
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
	wg      *sync.WaitGroup
	servers []*Frontend
}

func (c *Controller) Start(ctx context.Context) error {
	defer c.Shutdown()

	archon.InitLogger()

	// Connect to the database.
	if err := data.Initialize(dataSource(), debug.Enabled()); err != nil {
		return err
	}
	archon.Log.Infof("connected to database %s:%d",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
	)

	// Start any debug utilities if we're configured to do so.
	if debug.Enabled() {
		debug.StartUtilities()
	}

	// Start the shipgate gRPC server and make sure it launches before the other servers start.
	c.startShipgate(ctx)

	// Configure and run all of our servers.
	c.declareServers()
	return c.run(ctx)
}

const databaseURITemplate = "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s"

// Returns the database URI of the game database.
func dataSource() string {
	return fmt.Sprintf(
		databaseURITemplate,
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.name"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.sslmode"),
	)
}

// Set up all of the servers we want to run.
func (c *Controller) declareServers() {
	shipgateAddr := buildAddress(viper.GetString("shipgate_server.port"))

	// Automatically configure the block servers based on the number of
	// ship blocks requested.
	var blocks []ship.Block
	var blockServers []*Frontend
	for i := 1; i <= viper.GetInt("ship_server.num_blocks"); i++ {
		name := fmt.Sprintf("BLOCK%02d", i)
		address := buildAddress(viper.GetInt("block_server.port") + i)

		blocks = append(blocks, ship.Block{
			Name: name, Address: address, ID: i,
		})
		blockServer := &Frontend{
			Address: address,
			Backend: block.NewServer(
				name,
				shipgateAddr,
				viper.GetInt("block_server.num_lobbies"),
			),
		}
		blockServers = append(blockServers, blockServer)
	}

	c.servers = []*Frontend{
		{
			Address: buildAddress(viper.GetString("patch_server.patch_port")),
			Backend: patch.NewServer("PATCH", viper.GetString("patch_server.data_port")),
		},
		{
			Address: buildAddress(viper.GetString("patch_server.data_port")),
			Backend: patch.NewDataServer("DATA"),
		},
		{
			Address: buildAddress(viper.GetString("login_server.port")),
			Backend: login.NewServer("LOGIN", viper.GetString("character_server.port"), shipgateAddr),
		},
		{
			Address: buildAddress(viper.GetString("character_server.port")),
			Backend: character.NewServer("CHARACTER", shipgateAddr),
		},
		// Note: Eventually the ship and block servers should be able to be run
		// independently of the other four servers
		{
			Address: buildAddress(viper.GetString("ship_server.port")),
			Backend: ship.NewServer("SHIP", blocks, shipgateAddr),
		},
	}

	c.servers = append(c.servers, blockServers...)
}

func (c *Controller) run(ctx context.Context) error {
	// Start all of our servers. Failure to initialize one of the registered servers is considered terminal.
	for _, server := range c.servers {
		if err := server.Start(ctx, c.wg); err != nil {
			return fmt.Errorf("failed to start %s server: %w", server.Backend.Name(), err)
		}
	}

	c.wg.Wait()
	return nil
}

func (c *Controller) startShipgate(ctx context.Context) {
	readyChan := make(chan bool)
	errChan := make(chan error)

	go shipgate.Start(ctx, buildAddress(viper.GetString("shipgate_server.port")), readyChan, errChan)
	go func() {
		if err := <-errChan; err != nil {
			fmt.Printf("exiting due to SHIPGATE error: %v", err)
			os.Exit(1)
		}
	}()

	<-readyChan
}

func buildAddress(port interface{}) string {
	return fmt.Sprintf("%s:%v", viper.GetString("hostname"), port)
}

func (c *Controller) Shutdown() {
	data.Shutdown()
}
