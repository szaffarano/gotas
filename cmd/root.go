package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/logger"
)

const (
	taskdDataVariableName = "TASKDDATA"

	dataFlag    = "data"
	quietFlag   = "quit"
	verboseFlag = "verbose"
)

var log *logger.Logger

type flags struct {
	quiet    bool
	verbose  bool
	taskData string
}

// Version is the app version
type Version struct {
	Version string `json:",omitempty"`
	Commit  string `json:",omitempty"`
	Date    string `json:",omitempty"`
	BuiltBy string `json:",omitempty"`
}

func init() {
	log = logger.Log()
}

// Execute runs the root command
func Execute(version Version) {
	var flags flags

	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(version); err != nil {
		panic("Error building version")
	}

	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:           "gotas",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buffer.String(),
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
			log.Infof("==== gotas %s - %s - %s ====", version.Version, version.Commit, version.Date)
			return nil
		},
	}

	rootCmd.
		PersistentFlags().
		BoolVarP(&flags.quiet, quietFlag, "q", false, "Turns off verbose output")

	rootCmd.
		PersistentFlags().
		BoolVarP(&flags.verbose, verboseFlag, "v", false, "Generates debugging diagnostics")

	rootCmd.
		PersistentFlags().
		StringVar(&flags.taskData, dataFlag, "", "Data directory (default is $HOME/.gotas")

	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(removeCmd())
	rootCmd.AddCommand(resumeCmd())
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(suspendCmd())

	cobra.CheckErr(rootCmd.Execute())
}
