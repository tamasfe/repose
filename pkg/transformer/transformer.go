package transformer

import (
	"context"

	"github.com/tamasfe/repose/pkg/common"
	"github.com/tamasfe/repose/pkg/spec"
)

// Transformer transforms a specification
// before code generation.
type Transformer interface {
	common.DescriptionMarkdown

	// The name of the transformer.
	Name() string

	// A short description of the transformer.
	Description() string

	// DefaultOptions Returns the default options of the transformer, or nil if it has none.
	DefaultOptions() interface{}

	// Transform transforms the specification based on options.
	Transform(ctx context.Context, options interface{}, specification *spec.Spec) error
}
