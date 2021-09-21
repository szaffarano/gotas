package cmd

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	initCmd := cobra.Command{
		Use:   "init",
		Short: "Initializes a server instance at <data> directory.",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}

	return &initCmd
}
