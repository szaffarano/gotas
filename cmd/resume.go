package cmd

import (
	"github.com/spf13/cobra"
)

// resumeCmd represents the resume command
func resumeCmd() *cobra.Command {
	var resumeCmd = cobra.Command{
		Use:   "resume",
		Short: "Resumes, or un-suspends an organization or user",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}

	return &resumeCmd
}
