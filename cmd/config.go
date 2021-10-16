package cmd

import (
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	var configCmd = cobra.Command{
		Use:   "config",
		Short: "Displays or modifies a configuration variable value.",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}
	return &configCmd
}
