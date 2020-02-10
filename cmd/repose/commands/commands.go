package commands

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tamasfe/repose/pkg/util/cli"
)

var verbose bool
var silent bool
var noColors bool

var version string = "not versioned"

var rootCmd = &cobra.Command{
	Use:           "repose",
	Short:         "Repose is a code generator for REST servers",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cli.Verbose = verbose
		cli.Silent = silent

		color.NoColor = noColors
	},
}

var versionCmd = &cobra.Command{
	Use:           "version",
	Short:         "Version of Repose",
	Aliases:       []string{"v"},
	SilenceErrors: true,
	SilenceUsage:  true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "print verbose messages")
	rootCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "only print error messages, overwrites verbose")
	rootCmd.PersistentFlags().BoolVarP(&noColors, "no-colors", "", false, "disable colors in the output messages")

	rootCmd.AddCommand(versionCmd)
}

// Execute executes the commands.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		cli.Failureln(err)
		os.Exit(1)
	}
}
