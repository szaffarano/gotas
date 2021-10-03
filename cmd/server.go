package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task/server"
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

			transp, err := transport.NewServer(cfg)
			if err != nil {
				return fmt.Errorf("initializing server: %v", err)
			}
			defer func() {
				if err := transp.Close(); err != nil {
					panic(fmt.Sprintf("error closing server: %v", err))
				}
			}()

			// TODO implement graceful shutdown

			for {
				client, err := transp.NextClient()
				if err != nil {
					log.Errorf("Error receiving client: %s", err.Error())
				}

				go server.Process(client, cfg)
			}
		},
	}

	// TODO implement -d flag
	serverCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Runs server as a daemon")

	return &serverCmd
}
