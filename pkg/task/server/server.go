package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/apex/log"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task/repo"
)

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

func (s *server) Close() error {
	return s.listener.Close()
}

type client struct {
	conn net.Conn
}

func (c *client) Read(buf []byte) (int, error) {
	return c.conn.Read(buf)
}

func (c *client) Write(buf []byte) (int, error) {
	return c.conn.Write(buf)
}

func (c *client) Close() error {
	return c.conn.Close()
}

type server struct {
	listener net.Listener
}

func (s *server) NextClient() (Client, error) {
	conn, err := s.listener.Accept()
	if err != nil {
		return nil, err
	}

	return &client{conn}, nil
}

func NewServer(cfg config.Config) (Server, error) {
	var ca []byte
	var cert tls.Certificate
	var err error

	if ca, err = ioutil.ReadFile(cfg.Get(repo.CaCert)); err != nil {
		return nil, fmt.Errorf("reading root CA file: %v", err)
	}

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("reading creating root CA pool: %v", err)
	}

	if cert, err = tls.LoadX509KeyPair(cfg.Get(repo.ServerCert), cfg.Get(repo.ServerKey)); err != nil {
		return nil, fmt.Errorf("reading certificate file: %v", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    roots,
	}

	listener, err := tls.Listen("tcp", cfg.Get(repo.BindAddress), tlsCfg)
	if err != nil {
		return nil, err
	}

	log.Infof("Listening on %s...", cfg.Get(repo.BindAddress))
	return &server{listener}, nil
}
