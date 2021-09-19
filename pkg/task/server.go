// Copyright © 2021 Sebastián Zaffarano <sebas@zaffarano.com.ar>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package task

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"

	"github.com/apex/log"
	"github.com/szaffarano/gotas/pkg/config"
)

type Client struct {
	conn net.Conn
}

func (c *Client) Read(buf []byte) (int, error) {
	return c.conn.Read(buf)
}

func (c *Client) Write(buf []byte) (int, error) {
	return c.conn.Write(buf)
}

func (c *Client) Close() error {
	return c.conn.Close()
}

type Server struct {
	listener net.Listener
}

func (s *Server) NextClient() (*Client, error) {
	if conn, err := s.listener.Accept(); err != nil {
		return nil, err
	} else {
		return &Client{conn}, nil
	}
}

func (s *Server) Close() {
	s.listener.Close()
}

func NewServer() (*Server, error) {
	var ca []byte
	var cert tls.Certificate
	var err error
	conf := config.Get()

	if ca, err = ioutil.ReadFile(conf.Ca.Cert); err != nil {
		log.Fatalf("Failed to read root CA file", err)
	}

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(ca); !ok {
		log.Fatal("failed to parse root certificate")
	}

	if cert, err = tls.LoadX509KeyPair(conf.Server.Cert, conf.Server.Key); err != nil {
		log.Fatalf("Error reading certificate file", err)
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
	return &Server{listener}, nil
}
