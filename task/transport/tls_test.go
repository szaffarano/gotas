package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/config"
)

func TestServer(t *testing.T) {
	t.Run("server with valid config", func(t *testing.T) {
		server, client, cleanup := newTaskdClientServer(t, "client.conf")
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
		base := filepath.Join("testdata", "certs")
		dummyHandler := func(_ io.ReadWriteCloser) {
			assert.Fail(t, "unexpected handler call")
		}

		cases := []struct {
			title       string
			caCert      string
			serverCert  string
			serverKey   string
			bindAddress string
		}{
			{"invalid key cert", "ca.pem", "server.pem", "invalid-server-key", ""},
			{"malformed ca cert", "ca-invalid.pem", "server.pem", "server.key", ""},
			{"invalid ca cert", "non-existent", "server.pem", "server.key", ""},
			{"invalid bind address", "ca.pem", "server.pem", "server.key", "1:2:3:localhost"},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				cfg := TLSConfig{
					CaCert:      filepath.Join(base, c.caCert),
					ServerCert:  filepath.Join(base, c.serverCert),
					ServerKey:   filepath.Join(base, c.serverKey),
					BindAddress: filepath.Join(base, c.bindAddress),
				}

				srv, err := NewServer(cfg, 1, dummyHandler)
				assert.NotNil(t, err)
				assert.Nil(t, srv)
			})
		}
	})
}

func TestMaxConcurrency(t *testing.T) {
	maxConcurrency := 3

	base := filepath.Join("testdata", "certs")
	srvConfig := TLSConfig{
		CaCert:      filepath.Join(base, "ca.pem"),
		ServerCert:  filepath.Join(base, "server.pem"),
		ServerKey:   filepath.Join(base, "server.key"),
		BindAddress: fmt.Sprintf("localhost:%d", nextFreePort(t, 1025)),
	}
	clientCfg := newTLSConfig(t, "client.conf")
	var wg sync.WaitGroup
	wg.Add(1)
	ack := make(chan interface{})

	handler := func(client io.ReadWriteCloser) {
		defer client.Close()

		buf := make([]byte, 10)
		count, err := client.Read(buf)
		assert.Nil(t, err)
		assert.Greater(t, count, 0)
		ack <- 1
		wg.Wait()
	}

	srv, err := newTLSServer(srvConfig, maxConcurrency, handler)
	assert.Nil(t, err)
	defer srv.Close()

	for i := 0; i < maxConcurrency+1; i++ {
		go func() {
			client, err := tls.Dial("tcp", srvConfig.BindAddress, clientCfg)
			if err != nil {
				assert.FailNow(t, err.Error())
			}

			// force handshake
			_, err = client.Write([]byte("ping"))
			if err != nil {
				assert.FailNow(t, err.Error())
			}
		}()
	}

	received := 0
	timeouted := false
	for received < maxConcurrency+1 {
		select {
		case <-ack:
			received++
		case <-time.After(1000 * time.Millisecond):
			assert.False(t, timeouted)
			assert.Equal(t, maxConcurrency, received)
			timeouted = true
			wg.Done()
		}
	}
	if !assert.True(t, timeouted, "No concurrency bounded applied") {
		// finish all the ongoing connections
		wg.Done()
	}

}

func newTaskdClientServer(t *testing.T, clCfgFile string) (net.Conn, io.ReadWriteCloser, func()) {
	t.Helper()

	const ack = "ack"
	var client net.Conn
	var server io.ReadWriteCloser

	base := filepath.Join("testdata", "certs")
	srvConfig := TLSConfig{
		CaCert:      filepath.Join(base, "ca.pem"),
		ServerCert:  filepath.Join(base, "server.pem"),
		ServerKey:   filepath.Join(base, "server.key"),
		BindAddress: fmt.Sprintf("localhost:%d", nextFreePort(t, 1025)),
	}
	clientCfg := newTLSConfig(t, clCfgFile)

	ready := make(chan []byte)
	handler := func(client io.ReadWriteCloser) {
		buf := make([]byte, 10)
		// read something to force TLS handshake
		size, err := client.Read(buf)
		if err != nil {
			ready <- []byte{}
			assert.FailNow(t, err.Error())
		}
		server = client
		ready <- buf[:size]
	}

	srv, err := newTLSServer(srvConfig, 1, handler)
	if err != nil {
		assert.FailNowf(t, "Error creating server: %s", err.Error())
	}

	client, err = tls.Dial("tcp", srvConfig.BindAddress, clientCfg)
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
		srv.Close()
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
