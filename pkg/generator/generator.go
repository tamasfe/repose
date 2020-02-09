package generator

import (
	"context"

	"github.com/tamasfe/repose/pkg/spec"
)

// Generator generates code (e.g. for frameworks)
type Generator interface {
	// The name of the generator.
	Name() string

	// A short description of the generator.
	Description() string

	// Targets returns the targets the generator supports along with their summaries.
	Targets() map[string]string

	// DefaultOptions Returns the default options of the generator, or nil if it has none.
	DefaultOptions() interface{}

	// Generate generates code based on the options and targets.
	// The generated output must be either jen.Code, []byte, or string.
	Generate(ctx context.Context, options interface{}, specification *spec.Spec, target string) (interface{}, error)
}
