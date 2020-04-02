package server

// Server defines the methods implemented by all sub-servers that can be
// registered and started when the server is brought up.
type Server interface {
	// Name returns a uniquely identifying string.
	Name() string

	// Port returns the local port to which the server should be bound.
	Port() string

	// HeaderSize returns the size of the packet header in bytes.
	HeaderSize() uint16

	// AcceptClient should perform whatever initialization is needed to accept a client
	//connection and return a Client2 that wraps the provided ConnectionSate instance.
	//Note that this initialization may involve sending packets to the client.
	AcceptClient(cs *ConnectionState) (Client2, error)

	// Handle is the main entry point for processing client packets. It's responsible
	// for generally handling all packets from a client as well as sending any responses.
	Handle(c Client2) error
}
