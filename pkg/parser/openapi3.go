package parser

import (
	"bytes"
	"context"
	jsonstd "encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/tamasfe/repose/internal/markdown"
	"github.com/tamasfe/repose/pkg/common"
	"github.com/tamasfe/repose/pkg/util"
	"github.com/tamasfe/repose/pkg/util/types"

	"github.com/iancoleman/strcase"
	"github.com/mohae/deepcopy"

	"github.com/getkin/kin-openapi/openapi3"
	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
	"github.com/tamasfe/repose/pkg/errs"
	"github.com/tamasfe/repose/pkg/spec"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// OpenAPI3Options are options for the OpenAPI 3 parser.
type OpenAPI3Options struct {
	ExtensionName            string `yaml:"extensionName,omitempty" description:"The name of the extension field"`
	ResolveReferencesAt      string `yaml:"resolveReferencesAt,omitempty" description:"Resolve references at the given URL"`
	ResolveReferencesIn      string `yaml:"resolveReferencesIn,omitempty" description:"Resolve references in a local folder"`
	AdditionalPropertiesName string `yaml:"additionalPropertiesName" description:"Name of the additionalProperties field in structs that have them"`
	StripExtension           bool   `yaml:"stripExtension" description:"Strip the repose extension from the specification, the spec extension is used for code generation, and in most cases it's useless after that. Removing it for public APIs is also generally a good idea, where the specification will be visible"`
}

// MarshalYAML implements YAML Marshaler
func (o *OpenAPI3Options) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(o)
}

// OpenAPI3PathExtension is for specifications that support extensions.
// With it, a specification can alter the properties of code generation of the path.
type OpenAPI3PathExtension struct {
	Name *string `yaml:"name,omitempty" json:"name,omitempty" description:"The name of the path"`
}

// MarshalYAML implements YAML Marshaler
func (o *OpenAPI3PathExtension) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(o)
}

// OpenAPI3ResponseExtension is for specifications that support extensions.
// With it, a specification can alter the properties of code generation of the path.
type OpenAPI3ResponseExtension struct {
	Name *string `yaml:"name,omitempty" json:"name,omitempty" description:"The name of the response"`
}

// MarshalYAML implements YAML Marshaler
func (o *OpenAPI3ResponseExtension) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(o)
}

// OpenAPI3SchemaExtension is for specifications that support extensions.
// A specification can alter the properties of code generation of the schema with it.
type OpenAPI3SchemaExtension struct {
	Type     *string             `yaml:"type,omitempty" json:"type,omitempty" description:"The Go type of the schema"`
	Create   *bool               `yaml:"create,omitempty" json:"create,omitempty" description:"Whether the type should be created"`
	CanBeNil *bool               `yaml:"canBeNil,omitempty" json:"canBeNil,omitempty" description:"Whether the type can be nil, and should not have a pointer to it (e.g. slices, maps, or interfaces), it is only needed when a custom Go type is set, but create is set to false, so only the type name is known to Repose"`
	Tags     map[string][]string `yaml:"tags,omitempty" json:"tags,omitempty" description:"Additional tags for the field"`
}

// MarshalYAML implements YAML Marshaler
func (o *OpenAPI3SchemaExtension) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(o)
}

// OpenAPI3 parses Open API 3.x.x specifications
type OpenAPI3 struct{}

// Name implements Parser
func (o *OpenAPI3) Name() string {
	return "openapi3"
}

// Description implements Parser
func (o *OpenAPI3) Description() string {
	return "Supports parsing Open API 3 specifications"
}

// DescriptionMarkdown implements DescriptionMarkdown
func (o *OpenAPI3) DescriptionMarkdown() string {
	desc := `
# Description

This parser supports parsing Open API 3 specifications using [kin-openapi](https://github.com/getkin/kin-openapi).

Currently only one input file is supported, so definitions in local files are not resolved,
however resolving external resources are supported by kin-openapi.

# Options

## List of all options

{{ .OptionsTable }}

## Example usage in Repose config

{{ .OptionsExample }}

# Extensions

The parser supports several [extensions](https://swagger.io/docs/specification/openapi-extensions/)
that can be used in the specification to enhance code generation.

## Path

Extension for Open API 3 [paths](https://swagger.io/docs/specification/paths-and-operations/).

### Fields

{{ .PathExtensionTable }}

### Example

{{ .PathExtensionExample }}

## Response

Extension for Open API 3 [responses](https://swagger.io/docs/specification/describing-responses/).

### Fields

{{ .ResponseExtensionTable }}

### Example

{{ .ResponseExtensionExample }}

## Schema

Extension for Open API 3 [schemas](https://swagger.io/docs/specification/data-models/).

### Fields

{{ .SchemaExtensionTable }}

### Example

{{ .SchemaExtensionExample }}

{{ .SchemaExtensionCreateExample }}

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
			"OptionsTable": markdown.OptionsTable(*o.DefaultOptions().(*OpenAPI3Options)),
			"OptionsExample": "```yaml\n" + string(util.MustMarshalYAML(
				map[string]interface{}{
					"openapi3": o.DefaultOptions(),
				},
			)) + "```\n",
			"PathExtensionTable": markdown.ExtensionsTable(OpenAPI3PathExtension{}),
			"PathExtensionExample": "```yaml\n" + string(
				util.MustMarshalYAML(map[string]interface{}{
					"/gooddogs": map[string]interface{}{
						"x-repose": &OpenAPI3PathExtension{
							Name: types.StringPtr("GetGoodDogs"),
						},
					},
				})) + "```\n",
			"ResponseExtensionTable": markdown.ExtensionsTable(OpenAPI3ResponseExtension{}),
			"ResponseExtensionExample": "```yaml\n" + string(
				util.MustMarshalYAML(map[string]interface{}{
					"200": map[string]interface{}{
						"x-repose": &OpenAPI3ResponseExtension{
							Name: types.StringPtr("AllGoodDogs"),
						},
					},
				})) + "```\n",
			"SchemaExtensionTable": markdown.ExtensionsTable(OpenAPI3SchemaExtension{}),
			"SchemaExtensionExample": "```yaml\n" + string(
				util.MustMarshalYAML(map[string]interface{}{
					"GoodDog": map[string]interface{}{
						"x-repose": &OpenAPI3SchemaExtension{
							Type:     types.StringPtr("petslibrary.GoodDog"),
							Create:   types.BoolPtr(false),
							CanBeNil: types.BoolPtr(true),
						},
					},
				})) + "```\n",
			"SchemaExtensionCreateExample": "```yaml\n" + string(
				util.MustMarshalYAML(map[string]interface{}{
					"GoodDog": map[string]interface{}{
						"x-repose": &OpenAPI3SchemaExtension{
							Type:   types.StringPtr("LocalGoodDog"),
							Create: types.BoolPtr(true),
							Tags: map[string][]string{
								"json": []string{"localGoodDog", "omitempty"},
							},
						},
					},
				})) + "```\n",
		},
	)
	if err != nil {
		panic(err)
	}

	util.DisableYAMLMarshalComments = yamlComments

	return buf.String()
}

// DefaultOptions implements Parser
func (o *OpenAPI3) DefaultOptions() interface{} {
	return &OpenAPI3Options{
		ExtensionName:            "x-repose",
		ResolveReferencesAt:      "",
		ResolveReferencesIn:      "",
		AdditionalPropertiesName: "AdditionalProperties",
		StripExtension:           true,
	}
}

// ExtensionExamples implements Parser
func (o *OpenAPI3) ExtensionExamples() map[string]interface{} {
	return map[string]interface{}{
		"schema": &OpenAPI3SchemaExtension{
			Type:   &[]string{"CustomGoType"}[0],
			Create: &[]bool{false}[0],
			Tags: map[string][]string{
				"customTag": []string{"tagValue", "omitempty"},
			},
		},
		"path": &OpenAPI3PathExtension{
			Name: &[]string{"Users"}[0],
		},
		"response": &OpenAPI3ResponseExtension{
			Name: &[]string{"SomeResponse"}[0],
		},
	}
}

// Parse implements Parser
func (o *OpenAPI3) Parse(ctx context.Context, rawOpts interface{}, data []byte) (*spec.Spec, error) {
	opts := o.DefaultOptions().(*OpenAPI3Options)

	err := mapstructure.Decode(rawOpts, opts)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	state, ok := ctx.Value("state").(*common.State)
	if ok {
		state.SetSpecData(data)
	}

	sp := &spec.Spec{}

	// Load the swagger file
	loader := openapi3.NewSwaggerLoader()

	swagger, err := loader.LoadSwaggerFromData(data)
	if err != nil {
		return nil, err
	}

	// Resolve schema references at URL
	if opts.ResolveReferencesAt != "" {
		refURL, err := url.Parse(opts.ResolveReferencesAt)
		if err != nil {
			return nil, err
		}

		err = loader.ResolveRefsIn(swagger, refURL)
		if err != nil {
			// It's not a fatal error, we can continue
			fmt.Printf("failed to resolve references at %v", opts.ResolveReferencesAt)
		}
	}

	// The loader doesn't support resolving references locally
	// so we create a http server and serve the folder.
	if opts.ResolveReferencesIn != "" {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			panic(err)
		}
		defer listener.Close()

		addr := listener.Addr().String()

		srv := &http.Server{}

		http.Handle("/", http.FileServer(http.Dir(opts.ResolveReferencesIn)))

		go func() {
			// returns ErrServerClosed on graceful close
			if err := srv.Serve(listener); err != http.ErrServerClosed {
				fmt.Println(err)
			}
		}()
		defer srv.Close()

		localURL, err := url.Parse("http://" + addr)
		if err != nil {
			return nil, err
		}

		err = loader.ResolveRefsIn(swagger, localURL)
		if err != nil {
			// It's not a fatal error, we can continue
			fmt.Printf("failed to resolve references in folder %v", opts.ResolveReferencesIn)
		}
	}

	// Parse all the schemas
	err = o.ParseSchemas(ctx, sp, swagger, opts)
	if err != nil {
		return nil, err
	}

	// Then parse all thep aths
	err = o.ParsePaths(ctx, sp, swagger, opts)
	if err != nil {
		return nil, err
	}

	if opts.StripExtension {
		err := o.StripExtension(ctx, swagger, opts)
		if err != nil {
			return nil, err
		}
	}

	return sp, nil
}

// ParseResources implements Parser
func (o *OpenAPI3) ParseResources(ctx context.Context, options interface{}, paths ...string) (*spec.Spec, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths supplied")
	}

	// TODO resolve refs in multiple files
	b, err := ioutil.ReadFile(paths[0])
	if err != nil {
		return nil, err
	}

	return o.Parse(ctx, options, b)
}

// ParseSchemas parses the schema definitions
func (o *OpenAPI3) ParseSchemas(ctx context.Context, sp *spec.Spec, swagger *openapi3.Swagger, opts *OpenAPI3Options) error {
	if sp == nil {
		return fmt.Errorf("spec is nil")
	}

	if swagger == nil {
		return errs.ErrMissing("spec")
	}

	// Go over all the schemas
	for name, oapi3schema := range swagger.Components.Schemas {
		if oapi3schema.Value == nil {
			continue
		}

		schema, err := o.ParseSchema(ctx, oapi3schema, opts)
		if err != nil {
			return err
		}

		// Top level schemas need some extra checks,
		// they always have to be created if no
		// name is given. Except if it is explicitly
		// told not to create them in the extension.
		if schema.Name == "" {
			schema.Name = name

			var ext OpenAPI3SchemaExtension
			err := o.GetExtension(opts.ExtensionName, oapi3schema.Value.Extensions, &ext)
			if err != nil && err != ErrExtNotFound {
				return err
			}

			if ext.Create != nil && !*ext.Create {

				schema.Create = false
			} else {
				schema.Create = true
			}

		}

		sp.Schemas = append(sp.Schemas, schema)
	}

	return nil
}

// ParseSchema parses the Open API 3 schema and returns a Schema
func (o *OpenAPI3) ParseSchema(
	ctx context.Context,
	oapi3Schema *openapi3.SchemaRef,
	opts *OpenAPI3Options,
	visited ...*spec.Schema,
) (*spec.Schema, error) {
	if oapi3Schema == nil {
		return nil, errs.ErrMissing("schema")
	}

	schema := spec.NewSchema()

	// If the schema only has a Ref, it
	// will only have the name of it, nothing else.
	if oapi3Schema.Value == nil {
		if oapi3Schema.Ref != "" {
			rf := strings.Split(oapi3Schema.Ref, "/")
			schema.Name = rf[len(rf)-1]
			schema.OriginalName = rf[len(rf)-1]
			return schema, nil
		}
		return nil, errs.ErrMissing("schema")
	}

	// If it had a ref, we already know its name
	if oapi3Schema.Ref != "" {
		rf := strings.Split(oapi3Schema.Ref, "/")
		schema.Name = rf[len(rf)-1]
		schema.OriginalName = rf[len(rf)-1]
	}

	schema.Description = oapi3Schema.Value.Description

	var ext OpenAPI3SchemaExtension
	err := o.GetExtension(opts.ExtensionName, oapi3Schema.Value.Extensions, &ext)
	if err != nil && err != ErrExtNotFound {
		return nil, err
	}

	if ext.Type != nil && *ext.Type != "" {
		schema.Name = *ext.Type
	}

	if ext.Create != nil {

		if *ext.Create {
			schema.Create = true
		} else if schema.Name != "" {
			// We know the name, so
			// it should not be created
			if ext.CanBeNil != nil && *ext.CanBeNil {
				schema.Any()
			} else {
				schema.SetVariant(spec.VariantPrimitive)
			}

			return schema, nil
		}
	}

	// If we already have a schema with the same
	// name, we stop, because it is likely a
	// recursive schema.
	if schema.Name != "" {
		for _, s := range visited {
			if s.Name == schema.Name {
				return deepcopy.Copy(s).(*spec.Schema), nil
			}
		}
	}

	if ext.Tags != nil {
		schema.Tags = ext.Tags
	}

	if oapi3Schema.Value.Nullable {
		schema.SetNullable()
	}

	if oapi3Schema.Value.AllOf != nil {
		children := make([]*spec.Schema, 0, len(oapi3Schema.Value.AllOf))
		for _, v := range oapi3Schema.Value.AllOf {
			s, err := o.ParseSchema(ctx, v, opts, append(visited, schema)...)
			if err != nil {
				return nil, err
			}
			children = append(children, s)
		}
		return schema.AllOf(children), nil
	}

	if oapi3Schema.Value.AnyOf != nil {
		children := make([]*spec.Schema, 0, len(oapi3Schema.Value.AnyOf))
		for _, v := range oapi3Schema.Value.AnyOf {
			s, err := o.ParseSchema(ctx, v, opts, append(visited, schema)...)
			if err != nil {
				return nil, err
			}
			children = append(children, s)
		}
		return schema.AnyOf(children), nil
	}

	if oapi3Schema.Value.OneOf != nil {
		children := make([]*spec.Schema, 0, len(oapi3Schema.Value.OneOf))
		for _, v := range oapi3Schema.Value.OneOf {
			s, err := o.ParseSchema(ctx, v, opts, append(visited, schema)...)
			if err != nil {
				return nil, err
			}
			children = append(children, s)
		}
		return schema.OneOf(children), nil
	}

	if oapi3Schema.Value.Enum != nil {
		schema.Enum = deepcopy.Copy(oapi3Schema.Value.Enum).([]interface{})
	}

	switch strings.TrimSpace(oapi3Schema.Value.Type) {
	case "":
		schema.Any()
	case "object":
		props := make(map[string]*spec.Schema, len(oapi3Schema.Value.Properties))

		for propname, o3s := range oapi3Schema.Value.Properties {
			s, err := o.ParseSchema(ctx, o3s, opts, append(visited, schema)...)
			if err != nil {
				return nil, err
			}

			// Nullable and not required are the same
			// so if the field is not required,
			// we just set its type to nullable.
			nullable := true
			for _, k := range oapi3Schema.Value.Required {
				if propname == k {
					nullable = false
					break
				}
			}

			if !s.Nullable {
				s.Nullable = nullable
			}

			// propname is the field's name in the Go type,
			// but we also need to keep its original field name
			s.FieldName = propname
			propname = util.ToGoName(strcase.ToCamel(propname))

			props[propname] = s
		}
		schema.Struct(props)

		// Check if it has additional props
		if oapi3Schema.Value.AdditionalPropertiesAllowed != nil &&
			*oapi3Schema.Value.AdditionalPropertiesAllowed {
			// The additional properties can be anything
			schema.AdditionalProps = spec.NewSchema().SetVariant(spec.VariantAny)
		}

		if oapi3Schema.Value.AdditionalProperties != nil {
			// We know what the additional properties should look like
			additionalSchema, err := o.ParseSchema(ctx,
				oapi3Schema.Value.AdditionalProperties,
				opts,
				append(visited, schema)...,
			)
			if err != nil {
				return nil, err
			}
			schema.AdditionalProps = additionalSchema
		}

		// The name of the field can be set via an option
		// to avoid conflicts
		schema.AdditionalPropsName = opts.AdditionalPropertiesName

		// If we don't know any of the fields, we don't need a struct,
		// rather just a map.
		if schema.AdditionalProps != nil && len(schema.Children.Map) == 0 {
			schema.Map(spec.NewSchema().Primitive("string"), schema.AdditionalProps)
		}

	case "array":
		item, err := o.ParseSchema(ctx, oapi3Schema.Value.Items, opts, append(visited, schema)...)
		if err != nil {
			return nil, err
		}
		schema.Array(item)
	case "string":
		switch oapi3Schema.Value.Format {
		case "date", "date-time":
			schema.Primitive("time.Time")
		case "byte", "binary":
			schema.Array(spec.NewSchema().Primitive("byte"))
		default:
			schema.Primitive("string")
		}
	case "number":
		switch oapi3Schema.Value.Format {
		case "float":
			schema.Primitive("float32")
		case "double":
			schema.Primitive("float64")
		default:
			schema.Primitive("float64")
		}
	case "integer":
		switch oapi3Schema.Value.Format {
		case "int32":
			schema.Primitive("int32")
		case "int64":
			schema.Primitive("int64")
		default:
			schema.Primitive("int")
		}
	case "boolean":
		schema.Primitive("bool")
	default:
		return nil, fmt.Errorf("unknown type %v", oapi3Schema.Value.Type)
	}

	return schema, nil
}

// ParsePaths parses the paths of the specification
func (o *OpenAPI3) ParsePaths(ctx context.Context, sp *spec.Spec, swagger *openapi3.Swagger, opts *OpenAPI3Options) error {
	if sp == nil {
		return fmt.Errorf("spec cannot be nil")
	}

	// Go over all the paths
	for url, swaggerPath := range swagger.Paths {
		path, err := o.ParsePath(ctx, swaggerPath, opts)
		if err != nil {
			return err
		}

		path.PathString = url

		sp.Paths = append(sp.Paths, path)
	}

	return nil
}

// ParsePath parses a single path item.
func (o *OpenAPI3) ParsePath(ctx context.Context, swPath *openapi3.PathItem, opts *OpenAPI3Options) (*spec.Path, error) {

	path := &spec.Path{
		Description: swPath.Description,
	}

	var ext OpenAPI3PathExtension
	err := o.GetExtension(opts.ExtensionName, swPath.Extensions, &ext)
	if err != nil && err != ErrExtNotFound {
		return nil, err
	}

	if ext.Name != nil && *ext.Name != "" {
		path.Name = *ext.Name
	}

	// Parse each operation individually
	for method, op := range swPath.Operations() {
		specOp, err := o.ParseOperation(ctx, op, opts)
		if err != nil {
			return nil, err
		}
		specOp.Method = method
		path.Operations = append(path.Operations, specOp)
	}

	// We also need to add the parameters defined
	// on the path to all the operations
	for _, p := range swPath.Parameters {
		if p.Value == nil {
			continue
		}
		params, err := o.ParseParameter(ctx, p, opts)
		if err != nil {
			return nil, err
		}

		for _, op := range path.Operations {
			op.Parameters = append(op.Parameters, params...)
		}
	}

	return path, nil
}

// ParseOperation parses an Open API 3 operation
func (o *OpenAPI3) ParseOperation(ctx context.Context, op *openapi3.Operation, opts *OpenAPI3Options) (*spec.Operation, error) {

	specOp := &spec.Operation{
		Name:        strcase.ToCamel(op.OperationID),
		ID:          op.OperationID,
		Description: op.Description,
	}

	for _, p := range op.Parameters {
		if p.Value == nil {
			continue
		}

		params, err := o.ParseParameter(ctx, p, opts)
		if err != nil {
			return nil, err
		}

		specOp.Parameters = append(specOp.Parameters, params...)
	}

	// Request body is also a parameter, but we need to
	// parse it differently.
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		reqBody := op.RequestBody.Value

		for contentType, content := range reqBody.Content {
			param := &spec.Parameter{
				Name:        "body",
				Description: reqBody.Description,
				Required:    reqBody.Required,
				Type:        spec.ParameterTypeBody,
				ContentType: contentType,
			}

			if content.Schema != nil {
				s, err := o.ParseSchema(ctx, content.Schema, opts)
				if err != nil {
					return nil, err
				}
				param.Schema = s
			}

			specOp.Parameters = append(specOp.Parameters, param)
		}
	}

	for code, res := range op.Responses {
		if res.Value == nil {
			continue
		}

		var ext OpenAPI3ResponseExtension
		err := o.GetExtension(opts.ExtensionName, res.Value.Extensions, &ext)
		if err != nil && err != ErrExtNotFound {
			return nil, err
		}

		responseName := ""

		if ext.Name != nil && strings.TrimSpace(*ext.Name) != "" {
			responseName = strings.TrimSpace(*ext.Name)
		}

		if len(res.Value.Content) == 0 {
			specOp.Responses = append(specOp.Responses, &spec.Response{
				Name:        responseName,
				Description: res.Value.Description,
				Code:        code,
			})

			continue
		}

		for contentType, content := range res.Value.Content {
			specRes := &spec.Response{
				Name:        responseName,
				Description: res.Value.Description,
				ContentType: contentType,
				Code:        code,
			}

			if content.Schema != nil {
				s, err := o.ParseSchema(ctx, content.Schema, opts)
				if err != nil {
					return nil, err
				}
				specRes.Schema = s
			}

			specOp.Responses = append(specOp.Responses, specRes)
		}
	}

	cbs, err := o.ParseCallbacks(ctx, op.Callbacks, opts)
	if err != nil {
		return nil, err
	}
	specOp.Callbacks = cbs

	return specOp, nil
}

// ParseCallbacks parses the callbacks of an operation.
func (o *OpenAPI3) ParseCallbacks(ctx context.Context, cbs map[string]*openapi3.CallbackRef, opts *OpenAPI3Options) (map[string][]*spec.Path, error) {
	specCbs := make(map[string][]*spec.Path)

	for cbEvent, cbRef := range cbs {
		if cbRef == nil {
			continue
		}

		cbPaths := make([]*spec.Path, 0, len(*cbRef.Value))

		for cbURL, cb := range *cbRef.Value {
			specCb, err := o.ParsePath(ctx, cb, opts)
			if err != nil {
				return nil, err
			}

			specCb.PathString = cbURL

			var ext OpenAPI3PathExtension
			err = o.GetExtension(opts.ExtensionName, cb.Extensions, &ext)
			if err != nil && err != ErrExtNotFound {
				return nil, err
			}

			if ext.Name != nil && len(*ext.Name) != 0 {
				specCb.Name = *ext.Name
			}

			cbPaths = append(cbPaths, specCb)
		}

		specCbs[util.ToGoName(strings.Title(strcase.ToCamel(cbEvent)))] = cbPaths
	}

	return specCbs, nil
}

// ParseParameter parses a parameter.
// It can return multiple parameters, if the parameter has
// multiple content types.
func (o *OpenAPI3) ParseParameter(ctx context.Context, p *openapi3.ParameterRef, opts *OpenAPI3Options) ([]*spec.Parameter, error) {
	params := make([]*spec.Parameter, 0)

	if p.Value == nil {
		return nil, errs.ErrMissing("parameter")
	}

	sMethod, err := p.Value.SerializationMethod()
	if err != nil {
		return nil, err
	}

	simpleParam := &spec.Parameter{
		Name:        p.Value.Name,
		Description: p.Value.Description,
		Required:    p.Value.Required,
		Serialization: spec.ParameterSerialization{
			Style:   spec.ParameterSerializationStyle(sMethod.Style),
			Explode: sMethod.Explode,
		},
	}

	switch p.Value.In {
	case "query":
		simpleParam.Type = spec.ParameterTypeQuery
	case "header":
		simpleParam.Type = spec.ParameterTypeHeader
	case "path":
		simpleParam.Type = spec.ParameterTypePath
	case "cookie":
		simpleParam.Type = spec.ParameterTypeCookie
	default:
		return nil, fmt.Errorf("invalid parameter type: %v", p.Value.In)
	}

	// It's a simple parameter
	if p.Value.Schema != nil {
		s, err := o.ParseSchema(ctx, p.Value.Schema, opts)
		if err != nil {
			return nil, err
		}
		simpleParam.Schema = s
		params = append(params, simpleParam)
	}

	// The parameter might have multiple content types
	for contentType, content := range p.Value.Content {

		param := &spec.Parameter{
			Name:        p.Value.Name,
			Description: p.Value.Description,
			Required:    p.Value.Required,
			Type:        simpleParam.Type,
			Serialization: spec.ParameterSerialization{
				Style:   spec.ParameterSerializationStyle(sMethod.Style),
				Explode: sMethod.Explode,
			},
			ContentType: contentType,
		}

		if content.Schema != nil {
			s, err := o.ParseSchema(ctx, content.Schema, opts)
			if err != nil {
				return nil, err
			}
			param.Schema = s
			params = append(params, param)
		}
	}

	return params, nil
}

// GetExtension gets an extension from a schema
func (o *OpenAPI3) GetExtension(name string, extensions map[string]interface{}, dst interface{}) error {
	if extensions == nil {
		return ErrExtNotFound
	}

	if ext, ok := extensions[name]; ok {
		raw, isRawMessage := ext.(jsonstd.RawMessage)
		if !isRawMessage {
			return fmt.Errorf("invalid extension")
		}

		err := json.Unmarshal(raw, dst)
		if err != nil {
			return fmt.Errorf("invalid extension type: %v", err)
		}

		return nil
	}

	return ErrExtNotFound
}

// StripExtension strips the extension from the swagger specification
// and serialize it to the options.
func (o *OpenAPI3) StripExtension(ctx context.Context, swagger *openapi3.Swagger, opts *OpenAPI3Options) error {
	b, err := swagger.MarshalJSON()
	if err != nil {
		return err
	}

	var specMap map[string]interface{}
	err = json.Unmarshal(b, &specMap)
	if err != nil {
		return err
	}

	o.stripExtMap(specMap, opts.ExtensionName)

	finalB, err := json.Marshal(specMap)
	if err != nil {
		return err
	}

	state, ok := ctx.Value("state").(*common.State)
	if !ok {
		return fmt.Errorf("state is missing from context")
	}

	state.SetSpecData(finalB)

	return nil
}

func (o *OpenAPI3) stripExtMap(m map[string]interface{}, extName string) {
	delete(m, extName)
	for _, val := range m {
		switch v := val.(type) {
		case map[string]interface{}:
			delete(v, extName)
			o.stripExtMap(v, extName)
		case []interface{}:
			o.stripExtArray(v, extName)
		}
	}
}

func (o *OpenAPI3) stripExtArray(a []interface{}, extName string) {
	for _, val := range a {
		switch v := val.(type) {
		case map[string]interface{}:
			o.stripExtMap(v, extName)
		case []interface{}:
			o.stripExtArray(v, extName)
		}
	}
}

// ErrExtNotFound is returned if an extension doesn't exist
var ErrExtNotFound = errors.New("extension not found")
