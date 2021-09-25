package server

import (
	"fmt"
	"net"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task/repo"
)

func TestServer(t *testing.T) {
	t.Run("server with valid config", func(t *testing.T) {
		cfg, err := config.Load(filepath.Join("testdata", "repo_one", "config"))
		assert.Nil(t, err)

		before := nextFreePort(t, 1025)

		cfg.Set(repo.BindAddress, fmt.Sprintf("localhost:%d", before))

		srv, err := NewServer(cfg)
		assert.Nil(t, err)
		assert.NotNil(t, srv)

		assert.Less(t, before, nextFreePort(t, 1025))
		assert.NoError(t, srv.Close())
	})

	t.Run("server with invalid key cert", func(t *testing.T) {
		cfg, err := config.Load(filepath.Join("testdata", "repo_invalid_config", "config"))
		assert.Nil(t, err)

		srv, err := NewServer(cfg)
		assert.NotNil(t, err)
		assert.Nil(t, srv)
	})

	t.Run("server with malformed ca cert", func(t *testing.T) {
		cfg, err := config.Load(filepath.Join("testdata", "repo_invalid_config", "config"))
		assert.Nil(t, err)

		cfg.Set(repo.CaCert, filepath.Join("testdata", "repo_invalid_config", "certs", "ca.cert-invalid.pem"))

		_, err = NewServer(cfg)
		assert.NotNil(t, err)
	})

	t.Run("server with invalid ca cert", func(t *testing.T) {
		cfg, err := config.Load(filepath.Join("testdata", "repo_invalid_config", "config"))
		assert.Nil(t, err)

		cfg.Set(repo.CaCert, "invalid")

		_, err = NewServer(cfg)
		assert.NotNil(t, err)
	})

	t.Run("server with invalid bind address", func(t *testing.T) {
		cfg, err := config.Load(filepath.Join("testdata", "repo_one", "config"))
		assert.Nil(t, err)

		cfg.Set(repo.BindAddress, "1:2:3:localhost")

		_, err = NewServer(cfg)
		assert.NotNil(t, err)
	})
}

func nextFreePort(t *testing.T, initial int) int {
	for ; initial < 65535; initial++ {
		if l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", initial)); err == nil {
			defer assert.NoError(t, l.Close())
			return initial
		}
	}
	t.Error("Not available port found")
	return 0
}
