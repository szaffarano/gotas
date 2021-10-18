package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/task"
	"github.com/szaffarano/gotas/task/repo"
)

func initCmd() *cobra.Command {

	initCmd := cobra.Command{
		Use:   "init",
		Short: "Initializes a server instance at <data> directory.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dataDir := cmd.Flag(dataFlag).Value.String()

			// set default values
			defaultConfig := map[string]string{
				task.Confirmation: "true",
				task.Log:          filepath.Join(os.TempDir(), "taskd.log"),
				task.PidFile:      filepath.Join(os.TempDir(), "taskd.pid"),
				task.QueueSize:    "10",
				task.RequestLimit: "1048576",
				task.Root:         dataDir,
				task.Trust:        "strict",
				task.Verbose:      "true",
			}

			repository, err := repo.NewRepository(dataDir, defaultConfig)
			if err != nil {
				return err
			}

			log.Infof("Empty repository initialized: %q", repository)

			return nil
		},
	}

	return &initCmd
}
