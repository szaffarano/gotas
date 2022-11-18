package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/szaffarano/gotas/logger"
)

// TLSConfig exposes the configuration needed by the tls transport
type TLSConfig struct {
	CaCert      string
	ServerCert  string
	ServerKey   string
	BindAddress string
}

var log *logger.Logger

func init() {
	log = logger.Log()
}

// NewTlsServer creates a new tls-based server
func newTLSServer(cfg TLSConfig, maxConcurrency int, handlerFunc Handler) (Server, error) {
	var ca []byte
	var cert tls.Certificate
	var err error

	if ca, err = os.ReadFile(cfg.CaCert); err != nil {
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

	server := tlsServer{}

	server.listener = listener
	server.quit = make(chan interface{}, 1)
	server.wg.Add(1)
	server.handler = handlerFunc

	go server.serve(maxConcurrency)

	return &server, nil
}

type tlsServer struct {
	listener net.Listener
	quit     chan interface{}
	wg       sync.WaitGroup
	handler  Handler
}

func (s *tlsServer) Close() error {
	defer close(s.quit)

	s.quit <- true

	err := s.listener.Close()

	// wait indefinitely until all client connections finish
	s.wg.Wait()

	return err
}

func (s *tlsServer) serve(maxConcurrency int) {
	defer s.wg.Done()

	concurrency := make(chan interface{}, maxConcurrency)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Errorf("error receiving connection: %v", err)
			}
		}
		s.wg.Add(1)
		concurrency <- 1
		go func() {
			defer func() {
				<-concurrency
				s.wg.Done()
			}()

			s.handler(conn)
		}()
	}
}
