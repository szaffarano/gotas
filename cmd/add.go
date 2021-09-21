package cmd

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func addCmd() *cobra.Command {
	var addCmd = cobra.Command{
		Use:   "add",
		Short: "Creates a new organization or user.",
		Long: `When creating a new user, shows the resultant UUID that the client software
useѕ to uniquely identify a user, because <user-name> need not be unique.`,
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}

	return &addCmd
}
