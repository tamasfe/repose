package config

import (
	"github.com/tamasfe/repose/pkg/generator"
	"github.com/tamasfe/repose/pkg/generator/golang"
	"github.com/tamasfe/repose/pkg/parser"
	"github.com/tamasfe/repose/pkg/transformer"
	"github.com/tamasfe/repose/pkg/util"
)

// Generators supported by the CLI.
var Generators = []generator.Generator{
	&golang.General{},
	&golang.StdLib{},
	&golang.Echo{},
}

// Parsers supported by the CLI.
var Parsers = []parser.Parser{
	&parser.OpenAPI3{},
}

// Transformers supported by the CLI.
var Transformers = []transformer.Transformer{
	&transformer.Default{},
}

// Generator groups kinds and options for generator.Generator
type Generator struct {
	Targets []string    `yaml:"targets,omitempty" description:"Targets to generate"`
	Options interface{} `yaml:"options,omitempty" description:"Options for the generator"`
}

// Transformer groups the transformer name and its options
type Transformer struct {
	Name    string      `yaml:"name,omitempty" description:"Name of the transformer"`
	Options interface{} `yaml:"options,omitempty" description:"Options for the transformer"`
}

// MarshalYAML implements YAML Marshaler
func (t *Generator) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(t)
}

// GenerateOptions contains options for the CLI.
type GenerateOptions struct {
	Yes        bool
	Recursive  bool
	ConfigPath string
	OutPath    string
	Targets    string
}

// GetOptions contains options for the CLI.
type GetOptions struct {
	Force      bool
	NoComments bool
	All        bool
	OutPath    string
}

// ReposeOptions options for Repose.
type ReposeOptions struct {
	PackageName         string                 `yaml:"packageName" description:"Name of the package for the generated code"`
	FilePattern         string                 `yaml:"filePattern" description:"Pattern for generated file names if a directory is specified"`
	Timestamp           bool                   `yaml:"timestamp" description:"Add timestamp for the generated code"`
	Comments            bool                   `yaml:"comments" description:"Enable comments in the generated code"`
	DescriptionComments bool                   `yaml:"descriptionComments" description:"Enable descriptions from the specifications as comments in the generated code"`
	Parsers             map[string]interface{} `yaml:"parsers,omitempty" description:"Parsers to use and their options, leave it empty to infer from the input"`
	Transformers        []*Transformer         `yaml:"transformers,omitempty" description:"Transformers to alter the specification with before generating code, and their options"`
	Generators          map[string]*Generator  `yaml:"generators,omitempty" description:"Generators for code generation"`
}

// MarshalYAML implements YAML Marshaler
func (g *ReposeOptions) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(g)
}

// DefaultReposeOptions returns the default config
func DefaultReposeOptions() *ReposeOptions {
	return &ReposeOptions{
		PackageName:         "",
		DescriptionComments: true,
		Timestamp:           false,
		Comments:            true,
		FilePattern:         "{{ .Generator }}-{{ .Target }}.gen.go",
		Parsers:             map[string]interface{}{},
		Transformers:        []*Transformer{},
		Generators:          map[string]*Generator{},
	}
}

// ValidateReposeOptions validates options
func ValidateReposeOptions(opts *ReposeOptions) error {
	// TODO
	return nil
}
