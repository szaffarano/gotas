package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task"
	"github.com/szaffarano/gotas/pkg/task/repo"
	"github.com/szaffarano/gotas/pkg/task/transport"
)

func serverCmd() *cobra.Command {
	daemon := false
	var serverCmd = cobra.Command{
		Use:   "server",
		Short: "Runs the server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dataDir := cmd.Flag(dataFlag).Value.String()

			configFilePath := filepath.Join(dataDir, "config")
			cfg, err := config.Load(configFilePath)
			if err != nil {
				return err
			}

			tlsConfig := transport.TLSConfig{
				CaCert:      cfg.Get(task.CaCert),
				ServerCert:  cfg.Get(task.ServerCert),
				ServerKey:   cfg.Get(task.ServerKey),
				BindAddress: cfg.Get(task.BindAddress),
			}

			transp, err := transport.NewServer(tlsConfig)
			if err != nil {
				return fmt.Errorf("initializing server: %v", err)
			}
			defer func() {
				if err := transp.Close(); err != nil {
					panic(fmt.Sprintf("error closing server: %v", err))
				}
			}()

			auth, err := repo.NewDefaultAuthenticator(dataDir)
			if err != nil {
				return err
			}

			// TODO implement graceful shutdown

			ra := repo.NewDefaultReadAppender(dataDir)

			for {
				client, err := transp.NextClient()
				if err != nil {
					log.Errorf("Error receiving client: %s", err.Error())
				}

				go task.Process(client, auth, ra)
			}
		},
	}

	// TODO implement -d flag
	serverCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Runs server as a daemon")

	return &serverCmd
}
