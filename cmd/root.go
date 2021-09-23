package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	taskdDataVariableName = "TASKDDATA"
)

type flags struct {
	quiet    bool
	verbose  bool
	taskData string
}

func Execute(version string) {
	var flags flags

	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:           "gotas",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		Short:         "Taskwarrior server",
		Long: `Gotas aims to implement a taskwarrior server (aka taskd) using Go 
programming language`,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if flags.taskData == "" {
				value, ok := os.LookupEnv(taskdDataVariableName)
				if !ok {
					return fmt.Errorf("you have to define either $%s variable or data flag", taskdDataVariableName)
				}
				flags.taskData = value
			}
			return nil
		},
	}

	rootCmd.
		PersistentFlags().
		BoolVarP(&flags.quiet, "quiet", "q", false, "Turns off verbose output")

	rootCmd.
		PersistentFlags().
		BoolVarP(&flags.verbose, "verbose", "v", false, "Generates debugging diagnostics")

	rootCmd.
		PersistentFlags().
		StringVar(&flags.taskData, "data", "", "Data directory (default is $HOME/.gotas")

	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(removeCmd())
	rootCmd.AddCommand(resumeCmd())
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(suspendCmd())

	cobra.CheckErr(rootCmd.Execute())
}
