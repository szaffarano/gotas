package cmd

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pkg/task/repo"
)

func initCmd() *cobra.Command {
	initCmd := cobra.Command{
		Use:   "init",
		Short: "Initializes a server instance at <data> directory.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dataDir := cmd.Flag(dataFlag).Value.String()

			repository, err := repo.NewRepository(dataDir)
			if err == nil {
				log.Infof("Empty repository initialized: %q", repository)
			}

			return err
		},
	}

	return &initCmd
}
