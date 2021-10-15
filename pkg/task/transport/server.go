package transport

import (
	"io"
)

// Server implements the transport to communicate taskd clients with the server
type Server interface {
	// NextClient returns a client connection
	NextClient() (io.ReadWriteCloser, error)

	// Close stops taskd server
	Close() error
}

// NewServer creates a new taskd server working according to the configuration
func NewServer(cfg TLSConfig) (Server, error) {
	return newTLSServer(cfg)
}
