package task

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task/repo"
	"github.com/szaffarano/gotas/pkg/task/transport"
)

// Serve starts task server based on an initial configuration.
func Serve(cfg config.Config) (err error) {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	tlsConfig := transport.TLSConfig{
		CaCert:      cfg.Get(CaCert),
		ServerCert:  cfg.Get(ServerCert),
		ServerKey:   cfg.Get(ServerKey),
		BindAddress: cfg.Get(BindAddress),
	}

	auth, err := repo.NewDefaultAuthenticator(cfg.Get(Root))
	if err != nil {
		return err
	}

	ra := repo.NewDefaultReadAppender(cfg.Get(Root))

	handler := func(client io.ReadWriteCloser) {
		Process(client, auth, ra)
	}

	server, err := transport.NewServer(tlsConfig, cfg.GetInt(QueueSize), handler)
	if err != nil {
		return fmt.Errorf("initializing server: %v", err)
	}

	log.Infof("Listening on %s...", tlsConfig.BindAddress)

	<-shutdownChan

	log.Info("Shutting down taskserver...")

	return server.Close()
}
