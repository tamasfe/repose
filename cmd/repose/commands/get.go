package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tamasfe/repose/cmd/repose/config"
	"github.com/tamasfe/repose/pkg/util"
	"github.com/tamasfe/repose/pkg/util/cli"
	"gopkg.in/yaml.v3"
)

func init() {
	getCmd := &cobra.Command{
		Use:          "get [target]",
		Short:        "Get available values",
		SilenceUsage: false,
	}

	getOpts := &config.GetOptions{}

	getConfigCmd := &cobra.Command{
		Use:          "configuration",
		Short:        "Provides an example configuration",
		Aliases:      []string{"c", "conf", "config"},
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			if getOpts.OutPath == "" || getOpts.OutPath == "-" {
				cli.Silent = true
			}

			cfg := ""

			if getOpts.NoComments {
				util.DisableYAMLMarshalComments = true
			}

			configComment := "# Generated config file for Repose, a Go RESTful API code generation tool.\n\n"

			if !getOpts.All {
				b, err := marshalYAML(config.DefaultReposeOptions())
				if err != nil {
					cli.Failureln(err)
					return
				}
				cfg = string(b)
			} else {
				conf := config.DefaultReposeOptions()
				conf.Generators = make(map[string]*config.Generator)
				for _, t := range config.Generators {
					kinds := make([]string, 0, len(t.Targets()))
					for k := range t.Targets() {
						kinds = append(kinds, k)
					}

					conf.Generators[t.Name()] = &config.Generator{
						Targets: kinds,
						Options: t.DefaultOptions(),
					}
				}

				conf.Transformers = make([]*config.Transformer, 0)
				for _, t := range config.Transformers {

					conf.Transformers = append(conf.Transformers, &config.Transformer{
						Name:    t.Name(),
						Options: t.DefaultOptions(),
					})
				}

				conf.Parsers = make(map[string]interface{})
				for _, p := range config.Parsers {
					conf.Parsers[p.Name()] = p.DefaultOptions()
				}
				b, err := marshalYAML(conf)
				if err != nil {
					cli.Failureln(err)
					return
				}
				cfg = string(b)
			}

			if getOpts.OutPath != "" && getOpts.OutPath != "-" {
				if !getOpts.Force {
					_, err := os.Stat(getOpts.OutPath)
					if err == nil {
						cli.Failureln("file already exists, use \"-f\" to force overwrite.")
						return
					}
				}

				err := os.MkdirAll(filepath.Dir(getOpts.OutPath), os.ModePerm)
				if err != nil {
					cli.Failureln(err)
					return
				}
				info, err := os.Stat(getOpts.OutPath)
				if err != nil {
					if !os.IsNotExist(err) {
						cli.Failureln(err)
						return
					}
				}

				if info != nil && info.IsDir() {
					cli.Failureln("Output path should be a file, not a directory")
					return
				}

				f, err := os.Create(getOpts.OutPath)
				if err != nil {
					cli.Failureln(err)
					return
				}
				defer f.Close()

				_, err = f.Write([]byte(configComment + cfg))
				if err != nil {
					cli.Failureln(err)
					return
				}
				return
			}

			fmt.Println(configComment + cfg)
		},
	}

	getConfigCmd.Flags().BoolVarP(&getOpts.NoComments, "no-comments", "", false, "Disables all comments")
	getConfigCmd.Flags().StringVarP(&getOpts.OutPath, "out", "o", "", "the output directory or file")
	getConfigCmd.Flags().BoolVarP(&getOpts.All, "all", "a", false, "include all possible values")
	getConfigCmd.Flags().BoolVarP(&getOpts.Force, "force", "f", true, "force overwriting files")

	getParsersCmd := &cobra.Command{
		Use:          "parsers",
		Short:        "List all parsers",
		Aliases:      []string{"p", "parser", "parse"},
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			printParsers()
		},
	}

	getTransformersCmd := &cobra.Command{
		Use:          "transformers",
		Short:        "List all transformers",
		Aliases:      []string{"t", "trans", "transform"},
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			printTransformers()
		},
	}

	getGeneratorsCmd := &cobra.Command{
		Use:          "generators",
		Short:        "List all generators",
		Aliases:      []string{"g", "gen", "generator"},
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			printGenerators()
		},
	}

	getAllCmd := &cobra.Command{
		Use:          "all",
		Short:        "List all components",
		Aliases:      []string{"a"},
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			printParsers()
			fmt.Println()
			printTransformers()
			fmt.Println()
			printGenerators()
		},
	}

	getCmd.AddCommand(getAllCmd)
	getCmd.AddCommand(getGeneratorsCmd)
	getCmd.AddCommand(getTransformersCmd)
	getCmd.AddCommand(getParsersCmd)
	getCmd.AddCommand(getConfigCmd)

	rootCmd.AddCommand(getCmd)
}

func printParsers() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	cli.Infof("Available parsers:\n")
	for _, p := range config.Parsers {
		fmt.Fprintf(w, "\t%v\t%v\n", p.Name(), p.Description())
	}
	w.Flush()
}

func printTransformers() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	cli.Infof("Available transformers:\n")
	for _, p := range config.Transformers {
		fmt.Fprintf(w, "\t%v\t%v\n", p.Name(), p.Description())
	}
	w.Flush()
}

func printGenerators() {
	cli.Infof("Available generators:\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	for _, p := range config.Generators {
		targets := make([]string, 0, len(p.Targets()))

		for t := range p.Targets() {
			targets = append(targets, t)
		}

		fmt.Fprintf(w, "\t%v (%v)\t%v\n", p.Name(), strings.Join(targets, ", "), p.Description())
	}
	w.Flush()
}

// marshalYAML formats the getOpts.OutPathput YAML properly.
func marshalYAML(v interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}

	e := yaml.NewEncoder(buf)

	e.SetIndent(2)

	err := e.Encode(v)
	if err != nil {
		return nil, err
	}

	return []byte(strings.ReplaceAll(buf.String(), "\n\n\n", "\n\n")), nil
}
