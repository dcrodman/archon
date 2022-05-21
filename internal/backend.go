package internal

import (
	"context"

	"github.com/dcrodman/archon/internal/core/client"
)

// Backend is an interface for a sub-server that handles a specific set of client
// interactions as part of the game flow.
type Backend interface {
	// Name returns a uniquely identifying string.
	Identifier() string

	// Init is called before a Backend is started as a hook for the Backend to
	// perform any necessary initialization before it can accept clients.
	Init(ctx context.Context) error

	// SetUpClient performs any initialization on the Client needed to be
	// able to begin the session. Namely, it's the server's responsibility
	// to choose the appropriate encryption implementation.
	SetUpClient(c *client.Client)

	// Handshake performs any connection initialization necessary to begin
	// communicating with the client. This likely involves sending a "welcome" packet.
	Handshake(c *client.Client) error

	// Handle is the main entry point for processing client packets. It's responsible
	// for generally handling all packets from a client as well as sending any responses.
	Handle(ctx context.Context, c *client.Client, data []byte) error
}
