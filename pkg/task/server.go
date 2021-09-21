package task

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"github.com/szaffarano/gotas/pkg/config"
)

// Server is the taskd server
type Server interface {
	// NextClient returns a client
	NextClient() (Client, error)
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

func NewServer() (Server, error) {
	var ca []byte
	var cert tls.Certificate
	var err error
	conf := config.Get()

	if ca, err = ioutil.ReadFile(conf.Ca.Cert); err != nil {
		return nil, errors.Wrap(err, "Failed to read root CA file")
	}

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("failed to parse root certificate")
	}

	if cert, err = tls.LoadX509KeyPair(conf.Server.Cert, conf.Server.Key); err != nil {
		return nil, errors.Wrap(err, "Error reading certificate file")
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    roots,
	}

	listener, err := tls.Listen("tcp", conf.Server.BindAddress, cfg)
	if err != nil {
		return nil, err
	}

	log.Infof("Listening on %s...", conf.Server.BindAddress)
	return &server{listener}, nil
}
