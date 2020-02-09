package transformer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/mitchellh/mapstructure"
	"github.com/tamasfe/repose/internal/markdown"
	"github.com/tamasfe/repose/pkg/common"
	"github.com/tamasfe/repose/pkg/util"

	"github.com/iancoleman/strcase"

	"github.com/mohae/deepcopy"
	"github.com/tamasfe/repose/pkg/spec"
)

// TagTemplateValues contains values for tag templates.
type TagTemplateValues struct {
	Description string `description:"Description of the schema"`
	FieldName   string `description:"Name of the field"`
	Type        string `description:"Type of the field"`
}

// DefaultOptions alters the behaviour of the code generator.
type DefaultOptions struct {
	Tags map[string][]string `yaml:"tags,omitempty" description:"Add additional tags to struct fields. Supports Go templating with sprig functions"`
}

// MarshalYAML implements YAML Marshaler.
func (d *DefaultOptions) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(d)
}

// Default is the default Transformer.
type Default struct{}

// Name implements Transformer
func (d *Default) Name() string {
	return "default"
}

// Description implements Transformer
func (d *Default) Description() string {
	return "The default Repose specification transformer"
}

// DescriptionMarkdown implements DescriptionMarkdown
func (d *Default) DescriptionMarkdown() string {
	desc := `
# Description

This transformer is part the core of Repose Go code generation.
It doesn't have a lot of options yet, but almost everything relies on it.

# Options

## List of all options

{{ .OptionsTable }}

## Example usage in Repose config

{{ .OptionsExample }}

### Tag template values

{{ .TagsTable }}

`[1:]

	buf := &bytes.Buffer{}

	templ, err := template.New("desc").Parse(desc)
	if err != nil {
		panic(err)
	}

	yamlComments := util.DisableYAMLMarshalComments

	util.DisableYAMLMarshalComments = true

	err = templ.Execute(buf,
		map[string]interface{}{
			"OptionsTable": markdown.OptionsTable(*d.DefaultOptions().(*DefaultOptions)),
			"OptionsExample": "```yaml\n" + string(util.MustMarshalYAML(
				map[string]interface{}{
					"go-general": d.DefaultOptions(),
				},
			)) + "```\n",
			"TagsTable": markdown.TagsTable(TagTemplateValues{}),
		},
	)
	if err != nil {
		panic(err)
	}

	util.DisableYAMLMarshalComments = yamlComments

	return buf.String()
}

// DefaultOptions implements Transformer
func (d *Default) DefaultOptions() interface{} {
	return &DefaultOptions{
		Tags: map[string][]string{
			"json": []string{"{{ .FieldName }}", "omitempty"},
		},
	}
}

// Transform implements Transformer
func (d *Default) Transform(ctx context.Context, rawOpts interface{}, sp *spec.Spec) error {
	opts := d.DefaultOptions().(*DefaultOptions)

	err := mapstructure.Decode(rawOpts, opts)
	if err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	err = d.AddTags(ctx, sp, opts)
	if err != nil {
		return err
	}

	err = d.ExtractSchemas(ctx, sp, opts)
	if err != nil {
		return err
	}

	err = d.ExtractAllOfs(ctx, sp, opts)
	if err != nil {
		return err
	}

	err = d.GeneratePathNames(ctx, sp, opts)
	if err != nil {
		return err
	}

	err = d.GenerateOperationNames(ctx, sp, opts)
	if err != nil {
		return err
	}

	err = d.GenerateResponseNames(ctx, sp, opts)
	if err != nil {
		return err
	}

	err = d.SimplifyInlineSchemas(ctx, sp, opts)
	if err != nil {
		return err
	}

	err = d.OrderResources(ctx, sp, opts)
	if err != nil {
		return err
	}

	err = d.AddPathComments(ctx, sp, opts)
	if err != nil {
		return err
	}

	return nil
}

// ExtractSchemas extracts the nested schemas that need to be created.
// This only extracts schemas with a custom type, and leaves
// allOfs and such alone.
func (d *Default) ExtractSchemas(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	// Root schemas in sp.Schemas
	// are already extracted, extracting
	// them would result in an infinite recursion.
	extractRoot := false

	walkFunc := func(path spec.SchemaPath) error {
		if !extractRoot && len(path) < 2 {
			return nil
		}

		last := path.Last()

		if last == nil {
			return nil
		}
		if last.Create && last.Name != "" {

			exists := false
			for _, sch := range sp.Schemas {
				if sch.Name == last.Name {
					exists = true
					break
				}
			}

			if !exists {
				sp.Schemas = append(sp.Schemas, deepcopy.Copy(last).(*spec.Schema))
			}

			last.Create = false
		}

		return nil
	}

	// Walk all the root schemas
	for _, s := range sp.Schemas {
		err := s.Walk(walkFunc, true)
		if err != nil {
			return err
		}
	}

	// For parameters/responses, we can
	// extract anything.
	extractRoot = true

	for _, p := range sp.Paths {
		for _, o := range p.Operations {
			for _, param := range o.Parameters {
				err := param.Schema.Walk(walkFunc, true)
				if err != nil {
					return err
				}
			}

			for _, res := range o.Responses {
				err := res.Schema.Walk(walkFunc, true)
				if err != nil {
					return err
				}
			}

			for _, cb := range o.Callbacks {
				for _, cbPath := range cb {
					for _, cbOp := range cbPath.Operations {
						for _, param := range cbOp.Parameters {
							err := param.Schema.Walk(walkFunc, true)
							if err != nil {
								return err
							}
						}

						for _, res := range cbOp.Responses {
							err := res.Schema.Walk(walkFunc, true)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// SimplifyInlineSchemas simplifies inline schemas
// so that there will not be any attempts to
// create methods for them, and so on.
func (d *Default) SimplifyInlineSchemas(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	walkFunc := func(path spec.SchemaPath) error {
		last := path.Last()

		if last == nil {
			return nil
		}

		// Anonymous structs cannot have custom JSON marshaling,
		// so we cannot separate additional props in it easily,
		// and use a map instead.
		if last.AdditionalProps != nil && last.Name == "" {
			last.Map(spec.NewSchema().Primitive("string"), last.AdditionalProps)
		}

		if last.Name == "" {
			// These can be anything, don't try to generate helper code with no names
			if last.Variant == spec.VariantAnyOf || last.Variant == spec.VariantOneOf {
				last.SetVariant(spec.VariantAny)
			}
		}

		return nil
	}

	// Walk the root schemas
	for _, s := range sp.Schemas {
		err := s.Walk(walkFunc, true)
		if err != nil {
			return err
		}
	}

	for _, p := range sp.Paths {
		for _, o := range p.Operations {
			for _, param := range o.Parameters {
				err := param.Schema.Walk(walkFunc, true)
				if err != nil {
					return err
				}
			}

			for _, res := range o.Responses {
				err := res.Schema.Walk(walkFunc, true)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ExtractAllOfs creates all the types for AllOfs.
// AllOfs are a special type, all of their content
// are basically embedded structs.
func (d *Default) ExtractAllOfs(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	// We might end up with unknown names,
	// this number avoids conflicts.
	fragmentCount := 0

	walkFunc := func(path spec.SchemaPath) error {
		last := path.Last()

		// We are only interested in allOf types that need to be created.
		if last == nil || last.Variant != spec.VariantAllOf || !last.Create {
			return nil
		}

		for i, child := range last.Children.Array {
			name := child.Name

			// If the child is already extracted/created,
			// we just skip it.
			exists := false
			if name != "" {
				for _, sc := range sp.Schemas {
					if sc.Name == name {
						child.Create = false
						exists = true
						break
					}
				}
			}

			if exists {
				continue
			}

			// We figure some name out for it, if we don't
			// have any already.
			if name == "" {
				if last.Name != "" {
					name = last.Name + "Fragment" + strconv.Itoa(i)
				} else {
					name = "UnnamedFragment" + strconv.Itoa(fragmentCount)
					fragmentCount++
				}
			}

			child.Name = name

			// Check if the schema already exists
			// with the new name
			exists = false
			for _, sc := range sp.Schemas {
				if sc.Name == name {
					exists = true
					child.Create = false
					break
				}
			}

			if exists {
				continue
			}

			// Extract it to be created
			child.Create = true
			sp.Schemas = append(sp.Schemas, deepcopy.Copy(child).(*spec.Schema).
				AddComments(fmt.Sprintf("%v is a part of %v.", child.Name, last.Name)),
			)

			// We don't create it in the struct, only use its name.
			child.Create = false
		}

		return nil
	}

	for _, s := range sp.Schemas {
		err := s.Walk(walkFunc, true)
		if err != nil {
			return err
		}
	}

	for _, p := range sp.Paths {
		for _, o := range p.Operations {
			for _, param := range o.Parameters {
				err := param.Schema.Walk(walkFunc, true)
				if err != nil {
					return err
				}
			}

			for _, res := range o.Responses {
				err := res.Schema.Walk(walkFunc, true)
				if err != nil {
					return err
				}
			}

			for _, cb := range o.Callbacks {
				for _, cbPath := range cb {
					for _, cbOp := range cbPath.Operations {
						for _, param := range cbOp.Parameters {
							err := param.Schema.Walk(walkFunc, true)
							if err != nil {
								return err
							}
						}

						for _, res := range cbOp.Responses {
							err := res.Schema.Walk(walkFunc, true)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// GeneratePathNames generates path names if they don't already have one.
func (d *Default) GeneratePathNames(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	for _, p := range sp.Paths {

		// Parse the callbacks first if there's any.
		for _, o := range p.Operations {
			for cbName, cb := range o.Callbacks {
				for _, cbPath := range cb {
					if cbPath.Name != "" {
						continue
					}

					// Remove all the runtime expression values.
					re := regexp.MustCompile(`\{\$[^}]+\}`)
					cbPath.Name = re.ReplaceAllString(cbPath.PathString, "")

					// Also remove querystring if any.
					qsIndex := strings.Index(cbPath.Name, "?")
					if qsIndex != -1 {
						cbPath.Name = cbPath.Name[:qsIndex]
					}

					pathParts := strings.Split(cbPath.Name, "/")
					for i, x := range pathParts {
						if strings.Contains(x, "{") {
							pathParts[i] = "With" + strcase.ToCamel(strings.Trim(x, "{}"))
						} else {
							pathParts[i] = strcase.ToCamel(x)
						}
					}

					if len(pathParts) == 0 {
						cbPath.Name = cbName + "RootPath"
						continue
					}

					cbPath.Name = util.ToGoName(strings.Title(cbName + strcase.ToCamel(strings.Join(pathParts, "/"))))
				}
			}
		}

		if p.Name != "" {
			continue
		}

		// We replace all the {param} fields with
		// strings like "WithParam", then join the path
		// together with PascalCase to get a unique,
		// but still somewhat readable path name.
		pathParts := strings.Split(p.PathString, "/")
		for i, x := range pathParts {
			if strings.Contains(x, "{") {
				pathParts[i] = "With" + strcase.ToCamel(strings.Trim(x, "{}"))
			} else {
				pathParts[i] = strcase.ToCamel(x)
			}
		}

		if len(pathParts) == 0 {
			p.Name = "RootPath"
			continue
		}

		p.Name = util.ToGoName(strings.Title(strcase.ToCamel(strings.Join(pathParts, "/"))))
	}

	return nil
}

// GenerateOperationNames generates operation names if they don't already have one.
func (d *Default) GenerateOperationNames(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	for _, p := range sp.Paths {
		if p.Name == "" {
			return fmt.Errorf("path %v has no name", p.PathString)
		}

		for _, o := range p.Operations {
			if o.Name != "" {
				continue
			}

			// The operation name is simply the method + the path name
			// This relies on the path name already set.
			o.Name = strcase.ToCamel(strings.ToLower(o.Method) + strings.Title(p.Name))
		}
	}

	return nil
}

// GenerateResponseNames generates response names if they don't already have one.
func (d *Default) GenerateResponseNames(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	for _, p := range sp.Paths {
		for _, o := range p.Operations {
			if o.Name == "" {
				return fmt.Errorf("operation name is empty for path %v", p.PathString)
			}

			for _, r := range o.Responses {
				if r.Name != "" {
					r.Name = util.ToGoName(strcase.ToCamel(r.Name))
					continue
				}

				// The path name should already be set.
				r.Name = o.Name + "Response" + strcase.ToCamel(r.Code)
			}
		}
	}

	return nil
}

// AddTags adds tags that were given in the options
// or automatic tags if they are enabled.
func (d *Default) AddTags(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	// Tags are applied to the schemas, if needed.
	// With refNames we can also apply the same tags
	// to the schemas that are referenced, but are already extracted,
	// in which case setting tags in place is not enough, and has no effect.
	addTagsFunc := func(tags map[string][]string, refNames *[]string) spec.SchemaWalker {
		return func(path spec.SchemaPath) error {
			if len(tags) == 0 {
				return errors.New("should stop")
			}
			sm := path.Last()

			if sm.Tags == nil {
				sm.Tags = make(map[string][]string, len(opts.Tags))
			}

			actualTgs := make(map[string][]string)

			for k, tg := range tags {
				newTags := make([]string, len(tg))
				copy(newTags, tg)

				actualTgs[k] = newTags
			}

			for k, tg := range sm.Tags {
				newTags := make([]string, len(tg))
				copy(newTags, tg)

				actualTgs[k] = newTags
			}

			for _, tag := range actualTgs {
				name := sm.FieldName

				if name == "" {
					name = sm.OriginalName
				}

				for i, tagPart := range tag {
					templ, err := template.New("tag").Funcs(sprig.TxtFuncMap()).Parse(tagPart)
					if err != nil {
						return fmt.Errorf("unexpected error when parsing tags template: %w", err)
					}

					tagBuf := &bytes.Buffer{}
					err = templ.Execute(tagBuf, &TagTemplateValues{
						FieldName:   name,
						Type:        sm.Name,
						Description: sm.Description,
					})
					if err != nil {
						return fmt.Errorf("unexpected error when parsing tags template: %w", err)
					}

					tag[i] = tagBuf.String()
				}
			}

			sm.Tags = actualTgs

			if sm.Name != "" {
				if refNames != nil {
					*refNames = append(*refNames, sm.Name)
				}
			}

			return nil
		}
	}

	for _, s := range sp.Schemas {
		err := s.Walk(addTagsFunc(opts.Tags, nil), true)
		if err != nil {
			return err
		}
	}

	for _, p := range sp.Paths {
		for _, o := range p.Operations {
			err := d.addTagsToOperation(ctx, sp, o, addTagsFunc, opts)
			if err != nil {
				return err
			}

			for _, cb := range o.Callbacks {
				for _, cbPath := range cb {
					for _, cbOp := range cbPath.Operations {
						err := d.addTagsToOperation(ctx, sp, cbOp, addTagsFunc, opts)
						if err != nil {
							return err
						}
					}
				}
			}

		}
	}

	return nil
}

func (d *Default) addTagsToOperation(
	ctx context.Context,
	sp *spec.Spec,
	o *spec.Operation,
	addTagsFunc func(tags map[string][]string, refNames *[]string) spec.SchemaWalker,
	opts *DefaultOptions,
) error {

	for _, param := range o.Parameters {

		refs := make([]string, 0)
		err := param.Schema.Walk(addTagsFunc(opts.Tags, &refs), true)
		if err != nil {
			return err
		}

		// There are referenced schemas, walk them as well.
		for _, rName := range refs {
			for _, s := range sp.Schemas {
				if s.Name == rName {
					err := s.Walk(addTagsFunc(opts.Tags, nil), true)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	for _, res := range o.Responses {
		tags := make(map[string][]string)

		for k, tgs := range opts.Tags {
			tgsCopy := make([]string, len(tgs))
			copy(tgsCopy, tgs)
			tags[k] = tgsCopy
		}

		refs := make([]string, 0)

		err := res.Schema.Walk(addTagsFunc(tags, &refs), true)
		if err != nil {
			return err
		}

		// There are referenced schemas, walk them as well.
		for _, rName := range refs {
			for _, s := range sp.Schemas {
				if s.Name == rName {
					err := s.Walk(addTagsFunc(tags, nil), true)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// AddPathComments adds additional comments such
// as parameters or responses to paths and their operations.
func (d *Default) AddPathComments(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	for _, p := range sp.Paths {
		for _, o := range p.Operations {
			o.Comments = append(o.Comments,
				fmt.Sprintf("%v is the \"%v\" operation for path \"%v\".", o.Name, o.Method, p.PathString),
			)

			if options.DescriptionComments && o.Description != "" {
				o.Comments = append(o.Comments,
					"",
					fmt.Sprintf("Description: %v", strings.TrimSuffix(strings.TrimRight(o.Description, "\n"), ".")+"."),
					"",
				)
			}

			if len(o.Parameters) > 0 {
				o.Comments = append(o.Comments, "Parameters: ")

				for _, param := range o.Parameters {
					contentTypeText := ""

					if param.ContentType != "" {
						contentTypeText = " with content-type " + param.ContentType
					}

					o.Comments = append(o.Comments,
						fmt.Sprintf("    \"%v\" in %v%v.", param.Name, param.Type, contentTypeText),
					)
					if param.Description != "" && options.DescriptionComments {
						o.Comments = append(o.Comments, fmt.Sprintf("    Description: %v\n",
							strings.TrimSuffix(strings.TrimRight(param.Description, "\n"), ".")+"."),
						)
					}
				}

			}

			if len(o.Responses) > 0 {
				o.Comments = append(o.Comments, "Responses: ")
				for _, res := range o.Responses {
					contentTypeText := ""

					if res.ContentType != "" {
						contentTypeText = ": with content-type " + res.ContentType
					}

					resName := res.Name
					if res.Schema != nil && res.Schema.Name != "" {
						resName = res.Schema.Name
					}

					o.Comments = append(o.Comments, fmt.Sprintf("    \"%v\" (%v)%v.", resName, res.Code, contentTypeText))
					if res.Description != "" && options.DescriptionComments {
						o.Comments = append(o.Comments, fmt.Sprintf("    Description: %v\n",
							strings.TrimSuffix(strings.TrimRight(res.Description, "\n"), ".")+"."),
						)
					}
				}
			}

		}
	}

	return nil
}

// OrderResources orders all the spec resources in an alphabetical order.
func (d *Default) OrderResources(ctx context.Context, sp *spec.Spec, opts *DefaultOptions) error {
	sort.Slice(sp.Paths, func(i, j int) bool {
		p1, p2 := sp.Paths[i], sp.Paths[j]

		return p1.Name < p2.Name
	})

	for _, p := range sp.Paths {
		sort.Slice(p.Operations, func(i, j int) bool {
			o1, o2 := p.Operations[i], p.Operations[j]

			return o1.Name < o2.Name
		})

		for _, o := range p.Operations {
			sort.Slice(o.Parameters, func(i, j int) bool {
				p1, p2 := o.Parameters[i], o.Parameters[j]

				return p1.Name < p2.Name
			})

			sort.Slice(o.Responses, func(i, j int) bool {
				r1, r2 := o.Responses[i], o.Responses[j]

				return r1.Name < r2.Name
			})

			for _, cb := range o.Callbacks {
				for _, cbPath := range cb {
					sort.Slice(cb, func(i, j int) bool {
						p1, p2 := cb[i], cb[j]

						return p1.Name < p2.Name
					})

					for _, cbOp := range cbPath.Operations {
						sort.Slice(cbOp.Parameters, func(i, j int) bool {
							p1, p2 := cbOp.Parameters[i], cbOp.Parameters[j]

							return p1.Name < p2.Name
						})

						sort.Slice(cbOp.Responses, func(i, j int) bool {
							r1, r2 := cbOp.Responses[i], cbOp.Responses[j]

							return r1.Name < r2.Name
						})
					}

				}
			}
		}

	}

	return nil
}
