package cmd

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	var removeCmd = cobra.Command{
		Use:   "remove",
		Short: "Deletes an organization or user.  Permanently.",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}
	return &removeCmd
}
