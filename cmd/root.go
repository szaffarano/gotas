// Copyright © 2021 Sebastián Zaffarano <sebas@zaffarano.com.ar>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
			config.InitConfig(flags)
		},
	}

	rootCmd.
		PersistentFlags().
		BoolVarP(&flags.Quiet, "quiet", "q", false, "Turns off verbose output")

	rootCmd.
		PersistentFlags().
		BoolVarP(&flags.Debug, "debug", "d", false, "Generates debugging diagnostics")

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
