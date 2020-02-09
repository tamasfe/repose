package commands

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
	"github.com/tamasfe/repose/cmd/repose/config"
	"github.com/tamasfe/repose/cmd/repose/generate"
	"github.com/tamasfe/repose/pkg/util/cli"
)

func init() {
	genOpts := &config.GenerateOptions{}

	var success bool

	generateCmd := &cobra.Command{
		Use:          "generate [flags] [input]",
		Short:        "Generate code",
		Aliases:      []string{"gen"},
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if success {
				cli.Successln("All done!")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if genOpts.OutPath == "" || genOpts.OutPath == "-" {
				cli.Silent = true
			}

			opts := config.DefaultReposeOptions()

			if genOpts.ConfigPath != "" {
				var bt []byte

				if genOpts.ConfigPath == "-" {
					b, err := ioutil.ReadAll(os.Stdin)
					if err != nil {
						cli.Failuref("Failed to read config: %v\n", err)
						return
					}
					bt = b

					cli.Verboseln("Using config from stdin.")
				} else {
					b, err := ioutil.ReadFile(genOpts.ConfigPath)
					if err != nil {

						cli.Failuref("Failed to read config file: %v\n", err)
						return
					}
					bt = b
					absConfig, err := filepath.Abs(genOpts.ConfigPath)
					if err == nil {
						genOpts.ConfigPath = absConfig
					}

					cli.Verboseln("Using config from \"" + genOpts.ConfigPath + "\".")
				}

				opts = nil
				err := yaml.Unmarshal(bt, &opts)
				if err != nil {
					cli.Failuref("Invalid config file: %v\n", err)
					return
				}
			}

			err := generate.Generate(genOpts, opts, args)
			if err != nil {
				cli.Failuref("Generation failed: %v\n", err)
				return
			}
			success = true
		},
	}
	generateCmd.Flags().StringVarP(&genOpts.ConfigPath, "config", "c", "", "path to the configuration file or - for stdin")
	generateCmd.Flags().StringVarP(&genOpts.OutPath, "out", "o", "", "the output directory or file or - for stdout")
	generateCmd.Flags().BoolVarP(&genOpts.Yes, "yes", "y", false, "answer to all prompts with the default answers")
	generateCmd.Flags().StringVarP(&genOpts.Targets, "targets", "t", "", "targets to generate in the following format: \"go-general:types,spec;go-echo:server\", this overrides the values in the config")

	rootCmd.AddCommand(generateCmd)
}
