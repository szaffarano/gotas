package cmd

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func serverCmd() *cobra.Command {
	daemon := false
	var serverCmd = cobra.Command{
		Use:   "server",
		Short: "Runs the server",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}

	serverCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Runs server as a daemon")

	return &serverCmd
}
