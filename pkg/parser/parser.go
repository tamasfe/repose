package parser

import (
	"context"

	"github.com/tamasfe/repose/pkg/spec"
)

// Parser parses a specification, and returns
// the spec needed for code generation.
type Parser interface {
	// The name of the parser.
	Name() string

	// A short description of the parser.
	Description() string

	// DefaultOptions Returns the default options of the parser, or nil if it has none.
	DefaultOptions() interface{}

	// Parse parses a specification from data.
	Parse(ctx context.Context, options interface{}, data []byte) (*spec.Spec, error)

	// ParseResources parses a specification from one or more resources.
	ParseResources(ctx context.Context, options interface{}, paths ...string) (*spec.Spec, error)
}
