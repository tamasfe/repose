package errs

import "fmt"

// ErrMissingValue is returned if a value is not given
type ErrMissingValue struct {
	// What is missing
	Kind string

	// Additional info
	Info []string
}

// ErrMissing creates a "missing" error
func ErrMissing(kind string, info ...string) *ErrMissingValue {
	return &ErrMissingValue{
		Kind: kind,
		Info: info,
	}
}

func (e *ErrMissingValue) Error() string {
	return fmt.Sprintf(`%v is missing`, e.Kind)
}
