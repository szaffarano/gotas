package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task/repo"
)

func TestServer(t *testing.T) {

	t.Run("server with valid config", func(t *testing.T) {
		bindAddress := fmt.Sprintf("localhost:%d", nextFreePort(t, 1025))

		serverCfg := newTaskdConfig(t, "repo_one", repo.BindAddress, bindAddress)
		clientCfg := newTLSConfig(t, "client.pem", "client.key", "ca.pem")

		server, client, cleanup := newTaskdClientServer(t, serverCfg, clientCfg)
		defer cleanup()

		ch := make(chan []byte)
		go func() {
			buf := make([]byte, 10)
			for {
				size, err := server.Read(buf)
				if !assert.Nil(t, err) {
					assert.FailNowf(t, "Error reading from client: %s", err.Error())
				} else {
					ch <- buf[:size]
					break
				}
			}
		}()

		if _, err := client.Write([]byte("hello")); err != nil {
			assert.Fail(t, "Error receiving next client")
		}

		timeout := time.After(1 * time.Second)
		select {
		case <-timeout:
			assert.FailNow(t, "No payload received from server")
		case fromServer := <-ch:
			assert.Equal(t, "hello", string(fromServer))
		}
	})

	t.Run("invalid configurations", func(t *testing.T) {
		cases := []struct {
			title     string
			repo      string
			extraConf []string
		}{
			{"invalid key cert", "repo_invalid_config", make([]string, 0)},
			{"malformed ca cert", "repo_one", []string{
				repo.CaCert, filepath.Join("testdata",
					"repo_invalid_config",
					"certs",
					"ca.cert-invalid.pem")}},
			{"invalid ca cert", "repo_one", []string{repo.CaCert, "invalid"}},
			{"invalid invalid bind address", "repo_one", []string{repo.BindAddress, "1:2:3:localhost"}},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				opts := make([]interface{}, len(c.extraConf))
				for idx, o := range c.extraConf {
					opts[idx] = o
				}
				cfg := newTaskdConfig(t, c.repo, opts...)

				srv, err := NewServer(cfg)
				assert.NotNil(t, err)
				assert.Nil(t, srv)
			})
		}
	})
}

func newTaskdClientServer(t *testing.T, srvConfig config.Config, clConfig *tls.Config) (client net.Conn, server Client, cleanup func()) {
	t.Helper()

	const ack = "ack"

	ready := make(chan []byte)
	srv, err := NewServer(srvConfig)
	if err != nil {
		assert.FailNowf(t, "Error creating server: %s", err.Error())
	}

	go func() {
		defer srv.Close()
		buf := make([]byte, 10)
		server, err = srv.NextClient()
		if err != nil {
			assert.FailNow(t, err.Error())
		}
		// read something to force TLS handshake
		size, err := server.Read(buf)
		if err != nil {
			ready <- []byte{}
			assert.FailNow(t, err.Error())
		}
		ready <- buf[:size]
	}()

	client, err = tls.Dial("tcp", srvConfig.Get(repo.BindAddress), clConfig)
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	_, err = client.Write([]byte(ack))
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	// wait until server handshake to return to the client
	msg := <-ready
	assert.Equal(t, ack, string(msg))

	return client, server, func() {
		assert.NoError(t, server.Close())
		assert.NoError(t, client.Close())
	}
}
func nextFreePort(t *testing.T, initial int) int {
	t.Helper()

	for ; initial < 65535; initial++ {
		if l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", initial)); err == nil {
			defer assert.NoError(t, l.Close())
			return initial
		}
	}
	assert.FailNow(t, "Not available port found")
	return 0
}

func newTLSConfig(t *testing.T, certFile, keyFile, caFile string) *tls.Config {
	t.Helper()

	base := filepath.Join("testdata", "repo_one", "certs")
	ca, err := ioutil.ReadFile(filepath.Join(base, caFile))
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	cert, err := tls.LoadX509KeyPair(filepath.Join(base, certFile), filepath.Join(base, keyFile))
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
}

func newTaskdConfig(t *testing.T, repo string, extra ...interface{}) config.Config {
	t.Helper()

	path := filepath.Join("testdata", repo, "config")
	cfg, err := config.Load(path)
	if err != nil {
		assert.FailNowf(t, "Error reading configuration: %s", err.Error())
	}

	for i := 0; i < len(extra); i += 2 {
		cfg.Set(extra[i].(string), extra[i+1].(string))
	}

	return cfg
}
