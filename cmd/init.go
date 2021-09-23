package cmd

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pkg/task"
)

func initCmd() *cobra.Command {
	initCmd := cobra.Command{
		Use:   "init",
		Short: "Initializes a server instance at <data> directory.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dataDir := cmd.Flag("data").Value.String()

			repo, err := task.NewRepository(dataDir)
			if err == nil {
				log.Infof("Empty repository initialized: %q", repo.Config.Get(task.Root))
			}

			return err
		},
	}

	return &initCmd
}
