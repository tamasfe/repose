package gen

import (
	"fmt"
	"regexp"
	"strings"

	jen "github.com/dave/jennifer/jen"
)

// Template mixes jen code with a string template.
func Template(template string, values IntoValues, options ...TemplatingOption) (jen.Code, error) {
	opts := &templatingOptions{
		skipNotFound: false,
	}

	vals := values.Values()

	for _, o := range options {
		o(opts)
	}

	if len(vals) == 0 {
		return Raw(template), nil
	}

	for k, v := range vals {
		normalizedKey := strings.ToLower(strings.TrimSpace(k))

		if k != normalizedKey {
			vals[normalizedKey] = v
			delete(vals, k)
		}
	}

	substRe := regexp.MustCompile(`\{\{\s?\.([a-zA-Z0-9]+)\s?\}\}`)
	indices := substRe.FindAllStringSubmatchIndex(template, -1)

	if len(indices) == 0 {
		return jen.Op(template), nil
	}

	c := jen.Null()

	var lastIdx int
	for _, idx := range indices {
		codeKey := template[idx[2]:idx[3]]

		substCode, ok := vals[strings.ToLower(codeKey)]
		if !ok {
			if opts.skipNotFound {

				c.Op(template[lastIdx:idx[1]])
				lastIdx = idx[1]
				continue
			}

			return nil, fmt.Errorf("no code substitution for \"%v\" found", codeKey)
		}

		c.Op(template[lastIdx:idx[0]]).Add(substCode)
		lastIdx = idx[1]
	}

	// Remaining template after the last substitution
	c.Op(template[lastIdx:])

	return c, nil
}

func MustTemplate(template string, values IntoValues, options ...TemplatingOption) jen.Code {
	c, err := Template(template, values, options...)
	if err != nil {
		panic(err)
	}
	return c
}

// Values are code substitution values for templating.
type Values map[string]jen.Code

type IntoValues interface {
	Values() Values
}

func (v Values) Values() Values {
	return v
}

// TemplatingOptions options for template
type templatingOptions struct {
	skipNotFound bool
}

// TemplatingOption is used to set templating options
type TemplatingOption func(*templatingOptions)

// SkipNotFound is an option to leave not found values
// untouched in the template.
func SkipNotFound() TemplatingOption {
	return func(t *templatingOptions) {
		t.skipNotFound = true
	}
}
