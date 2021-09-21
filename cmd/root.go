package cmd

import (
	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pkg/config"
)

func Execute(version string) {
	var flags config.Flags

	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     "gotas",
		Version: version,
		Short:   "Taskwarrior server",
		Long: `Gotas aims to implement a taskwarrior server (aka taskd) using Go 
programming language`,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if err := config.InitConfig(flags); err != nil {
				panic(err.Error())
			}
		},
	}

	rootCmd.
		PersistentFlags().
		BoolVarP(&flags.Quiet, "quiet", "q", false, "Turns off verbose output")

	rootCmd.
		PersistentFlags().
		BoolVarP(&flags.Verbose, "verbose", "v", false, "Generates debugging diagnostics")

	rootCmd.
		PersistentFlags().
		StringVar(&flags.DataDir, "data", "", "Data directory (default is $HOME/.gotas")

	rootCmd.
		PersistentFlags().
		StringVar(&flags.ConfigFile, "config", "", "config file (default is $HOME/.gotas.yaml)")

	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(removeCmd())
	rootCmd.AddCommand(resumeCmd())
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(suspendCmd())

	cobra.CheckErr(rootCmd.Execute())
}
