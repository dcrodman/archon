package server

import (
	"net"
)

// Server defines the methods implemented by all sub-servers that can be
// registered and started when the server is brought up.
type Server interface {
	// Uniquely identifying string, mostly used for logging.
	Name() string
	// Port on which the server should listen for connections.
	Port() string
	// Perform any pre-startup initialization.
	Init() error
	// Client factory responsible for performing whatever initialization is
	// needed for Client objects to represent new connections.
	NewClient(conn *net.TCPConn) (*Client, error)
	// Process the packet in the client's buffer. The dispatcher will
	// read the latest packet from the client before calling.
	Handle(c *Client) error
}
