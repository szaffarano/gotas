package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"

	"github.com/apex/log"
)

// TLSConfig exposes the configuration needed by the tls transport
type TLSConfig struct {
	CaCert      string
	ServerCert  string
	ServerKey   string
	BindAddress string
}

// NewTlsServer creates a new tls-based server
func newTLSServer(cfg TLSConfig) (Server, error) {
	var ca []byte
	var cert tls.Certificate
	var err error

	if ca, err = ioutil.ReadFile(cfg.CaCert); err != nil {
		return nil, fmt.Errorf("reading root CA file: %v", err)
	}

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("reading creating root CA pool: %v", err)
	}

	if cert, err = tls.LoadX509KeyPair(cfg.ServerCert, cfg.ServerKey); err != nil {
		return nil, fmt.Errorf("reading certificate file: %v", err)
	}

	// base config from https://ssl-config.mozilla.org/ for "intermediate" systems
	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    roots,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	listener, err := tls.Listen("tcp", cfg.BindAddress, tlsCfg)
	if err != nil {
		return nil, err
	}

	log.Infof("Listening on %s...", cfg.BindAddress)
	return &tlsServer{listener}, nil
}

type tlsClient struct {
	conn net.Conn
}

type tlsServer struct {
	listener net.Listener
}

func (s *tlsServer) Close() error {
	return s.listener.Close()
}

func (c *tlsClient) Read(buf []byte) (int, error) {
	return c.conn.Read(buf)
}

func (c *tlsClient) Write(buf []byte) (int, error) {
	return c.conn.Write(buf)
}

func (c *tlsClient) Close() error {
	return c.conn.Close()
}

func (s *tlsServer) NextClient() (io.ReadWriteCloser, error) {
	conn, err := s.listener.Accept()
	if err != nil {
		return nil, err
	}

	return &tlsClient{conn}, nil
}
