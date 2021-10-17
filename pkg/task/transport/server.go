package transport

import "io"

// Server implements the transport to communicate taskd clients with the server
type Server interface {
	// NextClient returns a client connection
	// NextClient() (io.ReadWriteCloser, error)

	// Close stops taskd server
	Close() error
}

// Handler contains the logic to process an incoming connection
type Handler func(io.ReadWriteCloser)

// NewServer creates a new taskd server working according to the configuration
func NewServer(cfg TLSConfig, handler Handler) (Server, error) {
	return newTLSServer(cfg, handler)
}
