package transport

import "github.com/szaffarano/gotas/pkg/config"

// Server implements the transport to communicate taskd clients with the server
type Server interface {
	// NextClient returns a client connection
	NextClient() (Client, error)

	// Close stops taskd server
	Close() error
}

// Client represents a Taskd client connected to taskd server
type Client interface {
	// Read reads an stream of bytes send by the client
	Read(buf []byte) (int, error)

	// Write writes an stream of bytes to the client
	Write(buf []byte) (int, error)

	// Close closes the taskd client connection
	Close() error
}

// NewServer creates a new taskd server working according to the configuration
func NewServer(cfg config.Config) (Server, error) {
	return newTlsServer(cfg)
}
