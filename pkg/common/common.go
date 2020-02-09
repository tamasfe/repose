package common

// DescriptionMarkdown simply allows for getting markdown text.
type DescriptionMarkdown interface {
	DescriptionMarkdown() string
}

// Options provides information and settings
// for all parsers, transformers and targers.
type Options struct {
	DescriptionComments bool
	Comments            bool
}

// DefaultOptions returns the default options
func DefaultOptions() *Options {
	return &Options{
		DescriptionComments: true,
		Comments:            true,
	}
}

// State is a shared state for an entire code generation process.
// TODO make it threadsafe if needed
type State struct {
	specData       []byte
	packageAliases map[string]string
}

// SpecData returns the specification data.
func (s *State) SpecData() []byte {
	return s.specData
}

// SetSpecData sets specification data.
func (s *State) SetSpecData(data []byte) {
	s.specData = data
}

// SpecData returns the specification data.
func (s *State) PackageAlias(name, path string) {
	if s.packageAliases == nil {
		s.packageAliases = make(map[string]string)
	}

	s.packageAliases[name] = path
}

// SetSpecData sets specification data.
func (s *State) PackageAliases() map[string]string {
	return s.packageAliases
}

// ContextKey is a custom key type for contexts
type ContextKey string

// Context key values
const (
	ContextState            = "state"
	ContextCommonOptions    = "options"
	ContextGeneratorOptions = "generatorOptions"
)
