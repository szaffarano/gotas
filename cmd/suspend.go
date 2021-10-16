package cmd

import (
	"github.com/spf13/cobra"
)

func suspendCmd() *cobra.Command {
	var suspendCmd = cobra.Command{
		Use:   "suspend",
		Short: "Suspends an organization or user.",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}

	return &suspendCmd
}
