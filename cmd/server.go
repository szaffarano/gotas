package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/config"
	"github.com/szaffarano/gotas/task"
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

			return task.Serve(cfg)
		},
	}

	// TODO implement -d flag
	serverCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Runs server as a daemon")

	return &serverCmd
}
