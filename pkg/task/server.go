package task

// Server is the taskd server
type Server interface {
	// NextClient returns a client
	NextClient() (Client, error)

	// Close finishes the client connection
	Close() error
}

// Client represents a Taskd client connected to taskd server
type Client interface {
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	Close() error
}
