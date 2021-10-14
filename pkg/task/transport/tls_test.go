package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task"
)

func TestServer(t *testing.T) {
	t.Run("server with valid config", func(t *testing.T) {
		server, client, cleanup := newTaskdClientServer(t, "server.conf", "client.conf")
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
			assert.FailNowf(t, "error writing to the server: %v", err.Error())
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
			{"invalid key cert", "server.invalid-key.conf", []string{}},
			{"malformed ca cert", "server.malformed-ca.conf", []string{}},
			{"invalid ca cert", "server.invalid-ca.conf", []string{}},
			{"invalid invalid bind address", "server.conf", []string{task.BindAddress, "1:2:3:localhost"}},
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

func newTaskdClientServer(t *testing.T, srvCfgFile, clCfgFile string) (net.Conn, io.ReadWriteCloser, func()) {
	t.Helper()

	const ack = "ack"
	var client net.Conn
	var server io.ReadWriteCloser

	srvConfig := newTaskdConfig(t, srvCfgFile, task.BindAddress, fmt.Sprintf("localhost:%d", nextFreePort(t, 1025)))
	clientCfg := newTLSConfig(t, clCfgFile)

	ready := make(chan []byte)
	srv, err := newTLSServer(srvConfig)
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

	client, err = tls.Dial("tcp", srvConfig.Get(task.BindAddress), clientCfg)
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

func newTLSConfig(t *testing.T, conf string) *tls.Config {
	t.Helper()

	cfg, err := config.Load(filepath.Join("testdata", conf))
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	ca, err := ioutil.ReadFile(cfg.Get("ca"))
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	cert, err := tls.LoadX509KeyPair(cfg.Get("cert"), cfg.Get("key"))
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

func newTaskdConfig(t *testing.T, conf string, extra ...interface{}) config.Config {
	t.Helper()

	cfg, err := config.Load(filepath.Join("testdata", conf))
	if err != nil {
		assert.FailNowf(t, "Error reading configuration: %s", err.Error())
	}

	for i := 0; i < len(extra); i += 2 {
		cfg.Set(extra[i].(string), extra[i+1].(string))
	}

	return cfg
}
