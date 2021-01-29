package server

import "context"

// Backend is an interface for a sub-server that handles a specific set of client
// interactions as part of the game flow.
type Backend interface {
	// Name returns a uniquely identifying string.
	Name() string

	// Init is called before a Backend is started as a hook for the Backend to
	// perform any necessary initialization before it can accept clients.
	Init(ctx context.Context) error

	// CreateExtension returns an implementation of the ClientExtension interface
	// containing a fresh representation of Backend-specific state for a client.
	CreateExtension() ClientExtension

	// StartSession performs any connection initialization necessary to begin
	// communicating with the client. This likely involves sending a "welcome" packet.
	StartSession(c *Client) error

	// Handle is the main entry point for processing client packets. It's responsible
	// for generally handling all packets from a client as well as sending any responses.
	Handle(ctx context.Context, c *Client, data []byte) error
}
