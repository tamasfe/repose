package golang

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/tamasfe/repose/internal/markdown"
	"github.com/tamasfe/repose/pkg/common"
	"github.com/tamasfe/repose/pkg/util/gen"
	"github.com/tamasfe/repose/pkg/util/gen/templates"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	"github.com/mitchellh/go-wordwrap"
	"github.com/mitchellh/mapstructure"
	"github.com/tamasfe/repose/pkg/errs"
	"github.com/tamasfe/repose/pkg/spec"
	"github.com/tamasfe/repose/pkg/util"
)

// GeneralOptions are options the General target.
type GeneralOptions struct {
	GenerateTypeHelpers       bool   `yaml:"generateTypeHelpers" description:"Generate helper functions and methods for types"`
	GenerateGettersAndSetters bool   `yaml:"generateGettersAndSetters" description:"Generate helper methods for getting and setting properties for maps or structs with unknown names (E.g. additional properties)"`
	GenerateMarshalMethods    bool   `yaml:"generateMarshalMethods" description:"Generate marshal/unmarshal methods for types that need them"`
	TypesPackagePath          string `yaml:"typesPackagePath,omitempty" description:"Package path to already generated types (used internally)"`
	ExpandEnums               bool   `yaml:"expandEnums" description:"Expand enums into const (...) blocks if possible"`
}

// MarshalYAML implements YAML Marshaler
func (g *GeneralOptions) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(g)
}

// General generates framework-independent code.
type General struct{}

// Generate implements Generator
func (g *General) Generate(ctx context.Context, options interface{}, specification *spec.Spec, target string) (interface{}, error) {
	opts := g.DefaultOptions().(*GeneralOptions)

	err := mapstructure.Decode(options, opts)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	switch target {
	case "type", "types":
		return g.GenerateTypes(ctx, specification, opts)
	case "spec", "specification":
		state, ok := ctx.Value(common.ContextState).(*common.State)
		if !ok || state.SpecData() == nil {
			return nil, fmt.Errorf("specification data not supplied")
		}

		return g.GenerateSpec(ctx, state.SpecData(), "APISpecification")
	default:
		return nil, fmt.Errorf("target %v is not supported", target)
	}
}

// DefaultOptions implements Generator
func (g *General) DefaultOptions() interface{} {
	return &GeneralOptions{
		GenerateTypeHelpers:       true,
		GenerateGettersAndSetters: true,
		GenerateMarshalMethods:    true,
		ExpandEnums:               true,
	}
}

// Name implements Generator
func (g *General) Name() string {
	return "go-general"
}

// Description implements Generator
func (g *General) Description() string {
	return "Generates framework-agnostic code, such as types"
}

// Targets implements Generator
func (g *General) Targets() map[string]string {
	return map[string]string{
		"types": "Go types for the schemas in the specification",
		"spec":  "The bytes of the parsed specification file",
	}
}

// DescriptionMarkdown implements DescriptionMarkdown
func (g *General) DescriptionMarkdown() string {
	desc := `
# Description

This generator generates framework-agnostic Go code.

# Options

## List of all options

{{ .OptionsTable }}

## Example usage in Repose config

{{ .OptionsExample }}

# Targets

{{ .TargetsTable }}
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
			"OptionsTable": markdown.OptionsTable(*g.DefaultOptions().(*GeneralOptions)),
			"OptionsExample": "```yaml\n" + string(util.MustMarshalYAML(
				map[string]interface{}{
					"go-general": g.DefaultOptions(),
				},
			)) + "```\n",
			"TargetsTable": markdown.TargetsTable(g.Targets()),
		},
	)
	if err != nil {
		panic(err)
	}

	util.DisableYAMLMarshalComments = yamlComments

	return buf.String()
}

// GenerateTypes generates types from the spec
func (g *General) GenerateTypes(ctx context.Context, specification *spec.Spec, opts *GeneralOptions) (jen.Code, error) {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	if opts == nil {
		o, err := g.GetOpts(ctx)
		if err != nil {
			return nil, err
		}
		opts = o
	}

	// Sort schemas in alphabetical order
	sort.Slice(specification.Schemas, func(i, j int) bool {
		sc1, sc2 := specification.Schemas[i], specification.Schemas[j]

		return sc1.Name < sc2.Name
	})

	code := jen.Null()
	idx := 0
	for _, schema := range specification.Schemas {

		if !schema.Create {
			continue
		}

		if schema.Alias {

			aliasText := ""

			if schema.Children.GetSchema().Name != "" {
				aliasText = " for " + schema.Children.GetSchema().Name
			}

			if options.Comments {
				code.Commentf("// %v is an alias%v.", schema.Name, aliasText).Line()

				if options.DescriptionComments && schema.Description != "" {
					comms := []string{
						"",
						fmt.Sprintf("Description: %v",
							strings.TrimSuffix(strings.TrimRight(schema.Description, "\n"), ".")+"."),
					}

					strings.Split(wordwrap.WrapString(strings.Join(comms, "\n"), 80), "\n")

					code.Add(gen.Comments(comms...))

				}
			}

			targetC, err := g.GenerateType(ctx, schema.Children.GetSchema(), opts)
			if err != nil {
				return nil, err
			}

			code.Type().Id(schema.Name).Op("=").Add(targetC).Line().Line()
			continue
		}

		sCode, err := g.GenerateType(ctx, schema, opts)
		if err != nil {
			return nil, err
		}

		name := schema.Name

		if options.Comments {
			comms := make([]string, 0, len(schema.Comments)+1)

			comms = append(comms, schema.Comments...)

			if options.DescriptionComments && schema.Description != "" {
				comms = append([]string{
					fmt.Sprintf("%v description: %v", name,
						strings.TrimSuffix(strings.TrimRight(schema.Description, "\n"), ".")+"."),
				}, comms...)
			} else {
				var hasHeader bool
				for _, c := range schema.Comments {
					if strings.Contains(c, schema.Name) {
						hasHeader = true
						break
					}

				}
				if !hasHeader {
					comms = append([]string{
						fmt.Sprintf("%v is a generated type based on a schema.", name),
					}, comms...)
				}

			}

			// Word wrap comments, just in case.
			comms = strings.Split(wordwrap.WrapString(strings.Join(comms, "\n"), 80), "\n")

			for _, c := range comms {
				code.Comment("// " + c).Line()
			}
		}

		if name == "" {
			name = "UnnamedType" + strconv.Itoa(idx)
			idx++
		}

		code.Type().Id(name)

		code.Add(sCode).Line().Line()

		helperCode, err := g.GenerateHelpers(ctx, schema, opts)
		if err != nil {
			return nil, err
		}
		code.Add(helperCode)

		if opts.ExpandEnums && len(schema.Enum) > 0 {
			enumCode := jen.Null()

			if options.Comments {
				enumCode.Commentf("// Enum values for %v ", schema.Name).Line()

				defs := make([]jen.Code, 0, len(schema.Enum))

				for _, e := range schema.Enum {
					eName := fmt.Sprint(e)

					if strings.Contains(eName, "_") {
						eName = strings.Title(
							strings.ToLower(
								strings.Replace(fmt.Sprint(e), "_", " ", -1),
							),
						)
					}

					eName = util.ToGoName(strcase.ToCamel(eName))

					if strings.Contains(strings.ToLower(schema.Name), "error") {
						if !strings.HasPrefix(strings.ToLower(eName), "err") {
							eName = "Err" + eName
						}
					} else {
						eName = schema.Name + eName
					}

					defs = append(defs, jen.Id(eName).Id(schema.Name).Op("=").Lit(e))
				}

				enumCode.Const().Defs(
					defs...,
				).Line().Line()

				code.Add(enumCode)
			}
		}

	}

	return code, nil
}

// GenerateType generates a single type from a schema
func (g *General) GenerateType(ctx context.Context, schema *spec.Schema, opts *GeneralOptions) (jen.Code, error) {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	if opts == nil {
		o, err := g.GetOpts(ctx)
		if err != nil {
			return nil, err
		}
		opts = o
	}

	if !schema.Create && schema.Name != "" {
		// The name of the type references an another external package
		if strings.Contains(schema.Name, ".") {
			lastIdx := strings.LastIndex(schema.Name, ".")
			runes := []rune(schema.Name)
			return jen.Qual(string(runes[:lastIdx]), string(runes[lastIdx+1:])), nil
		}

		if opts.TypesPackagePath != "" {
			return jen.Qual(opts.TypesPackagePath, schema.Name), nil
		}

		return jen.Id(schema.Name), nil
	}

	switch schema.Variant {
	case spec.VariantAnyOf,
		spec.VariantOneOf,
		spec.VariantAny:

		return jen.Interface(), nil

	case spec.VariantAllOf:
		fields := make([]jen.Code, 0, len(schema.Children.Array))
		for _, child := range schema.Children.Array {
			if child.Name == "" {
				return nil, fmt.Errorf("Empty schema name in AllOf")
			}
			fields = append(fields, jen.Id(child.Name))
		}

		return jen.Struct(fields...), nil

	case spec.VariantArray:
		item, err := g.GenerateType(ctx, schema.Children.Schema, opts)
		if err != nil {
			return nil, err
		}

		if (schema.Children.Schema.Nullable || schema.Children.Schema.ShouldBePtr()) &&
			!schema.Children.Schema.CanBeNil() {
			item = jen.Op("*").Add(item)
		}

		return jen.Index().Add(item), nil

	case spec.VariantStruct:
		fields := make([]jen.Code, 0, len(schema.Children.Map))

		// Iterate the fields in alphabetical order
		mapKeys := make([]string, 0, len(schema.Children.Map))

		for k := range schema.Children.Map {
			mapKeys = append(mapKeys, k)
		}

		sort.Strings(mapKeys)

		for _, childName := range mapKeys {
			child := schema.Children.Map[childName]

			field := jen.Null()

			field.Id(childName)

			code, err := g.GenerateType(ctx, child, opts)
			if err != nil {
				return nil, err
			}

			if (child.Nullable || child.ShouldBePtr()) && !child.CanBeNil() {
				field.Op("*")
			}

			field.Add(code)

			if len(child.Tags) > 0 {
				tags := make(map[string]string, len(child.Tags))
				for n, t := range child.Tags {
					tags[n] = strings.Join(t, ",")
					if schema.AdditionalProps != nil {
						if child.FieldName == schema.AdditionalPropsName {
							tags["json"] = "-"
							tags["msgpack"] = "-"
							tags["xml"] = "-"
						}
					}
				}
				field.Tag(tags)
			}

			fields = append(fields, field)
		}

		marshalHelpers := jen.Null()

		// Generate custom marshal/unmarshal methods if needed.
		if schema.AdditionalProps != nil && schema.Name != "" {
			c := jen.Line().Line().Id(schema.AdditionalPropsName)

			if opts.GenerateMarshalMethods {
				additionalTp := jen.Null()

				aTp, err := g.GenerateType(ctx, schema.AdditionalProps, opts)
				if err != nil {
					return nil, err
				}

				if (schema.AdditionalProps.Nullable || schema.AdditionalProps.ShouldBePtr()) &&
					!schema.AdditionalProps.CanBeNil() {
					additionalTp.Op("*")
				}

				additionalTp.Add(aTp)

				hasJSON := false
				for _, f := range schema.Children.Map {
					for t := range f.Tags {
						if t == "json" {
							hasJSON = true
							break
						}
					}
				}

				if hasJSON {
					marshalHelpers.Line().Line()

					if options.Comments {
						marshalHelpers.Comment("// MarshalJSON is a custom marshaler because").Line()
						marshalHelpers.Comment("// the type has additional unknown properties.").Line()
					}

					objectType := schema.Name
					objectName := strings.ToLower(string([]rune(schema.Name)[0]))
					additionalPropsName := schema.AdditionalPropsName

					knownFields := make([]jen.Code, 0, len(schema.Children.Map))

					for _, c := range schema.Children.Map {
						knownFields = append(knownFields, jen.Lit(c.FieldName))
					}

					marshalCode := gen.MustTemplate(
						templates.JSONMarshalAdditionalProps,
						&templates.JSONMarshalAdditionalPropsValues{
							ReceiverName:              jen.Id(objectName),
							TypeName:                  jen.Id(objectType),
							AdditionalPropsName:       jen.Id(additionalPropsName),
							JsonMarshal:               jen.Qual("encoding/json", "Marshal"),
							JsonUnmarshal:             jen.Qual("encoding/json", "Unmarshal"),
							AdditionalPropsTypeString: jen.Lit(schema.AdditionalPropsName),
						},
					)

					marshalHelpers.Add(marshalCode).Line().Line()

					if options.Comments {
						marshalHelpers.Comment("// UnmarshalJSON is a custom unmarshaler because").Line()
						marshalHelpers.Comment("// the type has additional unknown properties.").Line()
					}

					unmarshalCode := gen.MustTemplate(
						templates.JSONUnmarshalAdditionalProps,
						&templates.JSONUnmarshalAdditionalPropsValues{
							ReceiverName:        jen.Id(objectName),
							TypeName:            jen.Id(objectType),
							AdditionalPropsName: jen.Id(additionalPropsName),
							KnownFields:         jen.List(knownFields...),
							JsonUnmarshal:       jen.Qual("encoding/json", "Unmarshal"),
							AdditionalPropsType: jen.Map(jen.String()).Add(additionalTp),
						},
					)

					marshalHelpers.Add(unmarshalCode).Line().Line()
				}

				c.Add(jen.Map(jen.String()).Add(additionalTp))

				fields = append(fields, c)
			}
		}

		return jen.Struct(fields...).Add(marshalHelpers), nil

	case spec.VariantMap:
		keyC, err := g.GenerateType(ctx, schema.Children.Array[0], opts)
		if err != nil {
			return nil, err
		}

		valSchema := schema.Children.GetArray()[1]

		valC := jen.Null()

		if (valSchema.Nullable || valSchema.ShouldBePtr()) &&
			!valSchema.CanBeNil() {
			valC.Op("*")
		}

		vC, err := g.GenerateType(ctx, schema.Children.Array[1], opts)
		if err != nil {
			return nil, err
		}

		valC.Add(vC)

		return jen.Map(keyC).Add(valC), nil

	case spec.VariantPrimitive:
		if strings.Contains(schema.PrimitiveType, ".") {
			lastIdx := strings.LastIndex(schema.PrimitiveType, ".")
			runes := []rune(schema.PrimitiveType)
			return jen.Qual(string(runes[:lastIdx]), string(runes[lastIdx+1:])), nil
		}

		return jen.Id(schema.PrimitiveType), nil

	default:
		return nil, errs.ErrMissing("variant")
	}
}

// GenerateHelpers generates various helper functions
// for a type.
func (g *General) GenerateHelpers(ctx context.Context, schema *spec.Schema, opts *GeneralOptions) (jen.Code, error) {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	if opts == nil {
		o, err := g.GetOpts(ctx)
		if err != nil {
			return nil, err
		}
		opts = o
	}

	code := jen.Null()

	shortName := strings.ToLower(string(schema.Name[0]))

	// Generate AnyOf/OneOf helper methods to cast the type.
	if opts.GenerateTypeHelpers {
		if schema.Name != "" &&
			(schema.Variant == spec.VariantAnyOf ||
				schema.Variant == spec.VariantOneOf) {

			for _, c := range schema.Children.GetArray() {
				childName := c.Name

				// If we don't know what it is supposed to be,
				// we just skip it.
				if childName == "" {
					continue
				}

				// If the name is the same, it's likely a
				// recursive schema, either way, the cast is pointless.
				if childName == schema.Name {
					continue
				}

				childName = util.ToGoName(strcase.ToCamel(childName))

				cCode, err := g.GenerateType(ctx, c, opts)
				if err != nil {
					return nil, err
				}

				cCodeNullable := jen.Code(jen.Op("*").Add(cCode))

				if c.CanBeNil() {
					cCodeNullable = cCode
				}

				headerCode := jen.Func().Id(util.ToGoName(strcase.ToCamel(schema.Name + "As" + childName))).
					Params(jen.Id(shortName).Id(schema.Name)).Params(cCodeNullable)

				bodyCode := jen.Null()

				// First we check for nil
				bodyCode.If(jen.Id(shortName).Op("==").Nil()).Block(
					jen.Return(jen.Nil()),
				).Line().Line()

				// Then we try to simply assert it.
				bodyCode.If(
					jen.List(jen.Id("val"), jen.Id("ok")).Op(":=").Id(shortName).Assert(cCodeNullable),
					jen.Id("ok"),
				).Block(
					jen.Return(jen.Id("val")),
				).Line().Line()

				// If the type can be a pointer,
				// we need to check for value type as well,
				// and return a pointer for it.
				if !c.CanBeNil() {
					bodyCode.If(
						jen.List(jen.Id("val"), jen.Id("ok")).Op(":=").Id(shortName).Assert(cCode),
						jen.Id("ok"),
					).Block(
						jen.Return(jen.Op("&").Id("val")),
					).Line().Line()
				}

				// Then we try to JSON marshal the value
				bodyCode.List(jen.Id("b"), jen.Err()).Op(":=").
					Add(g.jsonCall(false, "Marshal")).Call(jen.Id(shortName)).Line().Line()

				// Check for marshal error
				bodyCode.If(jen.Err().Op("!=").Nil()).Block(
					jen.Return(jen.Nil()),
				).Line().Line()

				// Then we will attempt to unmarshal it
				// into the correct type.
				bodyCode.Var().Id("val").Add(cCodeNullable).Line().Line()

				// Create a strict decoder that
				// won't allow unknown fields.
				bodyCode.Id("d").Op(":=").Add(g.jsonCall(false, "NewDecoder")).
					Call(jen.Qual("bytes", "NewReader").Call(jen.Id("b"))).Line()
				bodyCode.Id("d").Dot("DisallowUnknownFields").Call().Line().Line()

				// Finally decode into the correct type
				bodyCode.Id("d").Dot("Decode").Call(jen.Op("&").Id("val")).Line().Line()

				// Return the result, which is possibly nil
				bodyCode.Return(jen.Id("val")).Line()

				if options.Comments {

					code.Commentf("// %v casts %v to %v if possible.",
						strcase.ToCamel(schema.Name+"As"+childName), schema.Name, childName).Line()
				}

				code.Add(headerCode.Block(bodyCode)).Line().Line()
			}
		}
	}

	// Generate Getters and Setters for
	// schemas with additional properties.
	if opts.GenerateGettersAndSetters {
		if schema.AdditionalProps != nil {
			schemaType := jen.Id(schema.Name)

			additionalType := jen.Null()

			if (schema.AdditionalProps.Nullable || schema.AdditionalProps.ShouldBePtr()) &&
				!schema.AdditionalProps.CanBeNil() {
				additionalType.Op("*")
			}

			addTp, err := g.GenerateType(ctx, schema.AdditionalProps, opts)
			if err != nil {
				return nil, err
			}

			additionalType.Add(addTp)

			returnEmptyVal := jen.Null()

			if (schema.AdditionalProps.Nullable || schema.AdditionalProps.ShouldBePtr()) &&
				!schema.AdditionalProps.CanBeNil() {
				returnEmptyVal.Nil()
			} else {
				returnEmptyVal.Op("*").New(additionalType)
			}

			switch schema.Variant {
			case spec.VariantMap:
				setter := jen.Null()

				if options.Comments {
					setter.Comment("// Set sets a value for the map with the given key.").Line()
					setter.Comment("// If the map is nil, it is created.").Line()
				}

				// The setter method
				setter.Func().
					Params(jen.Id(shortName).Id(schema.Name)).
					Id("Set").
					Params(jen.Id("key").String(), jen.Id("value").Add(additionalType)).
					Block(
						jen.If(jen.Id(shortName).Op("==").Nil()).Block(
							jen.Id(shortName).Op("=").Make(jen.Add(schemaType)),
						),
						jen.Id(shortName).Index(jen.Id("key")).Op("=").Id("value"),
					).Line().Line()

				getter := jen.Null()

				if options.Comments {
					getter.Comment("// Get sets a value from the map with the given key.").Line()
					getter.Comment("// If the map is nil, or the value doesn't exist,").Line()
					getter.Comment("// the default value (or nil) is returned.").Line()
				}

				// The getter method
				getter.Func().
					Params(jen.Id(shortName).Id(schema.Name)).
					Id("Get").
					Params(jen.Id("key").String()).
					Params(additionalType).
					Block(
						// If the map is nil
						jen.If(jen.Id(shortName).Op("==").Nil()).Block(
							jen.Return(returnEmptyVal),
						).Line().Line(),

						// Try to get the value by key
						jen.If(
							jen.List(jen.Id("val"), jen.Id("ok")).Op(":=").
								Id(shortName).Index(jen.Id("key")),
							jen.Id("ok"),
						).Block(
							jen.Return(jen.Id("val")),
						).Line().Line(),
						jen.Return(returnEmptyVal),
					).Line().Line()

				code.Add(getter, setter)

			case spec.VariantStruct:
				setter := jen.Null()

				if options.Comments {
					setter.Comment("// Set sets a value for the additional properties with the given key.").Line()
					setter.Comment("// If the type is nil, or the additional types are nil, they are created.").Line()
				}

				// The setter method
				setter.Func().
					Params(jen.Id(shortName).Op("*").Id(schema.Name)).
					Id("Set").
					Params(jen.Id("key").String(), jen.Id("value").Add(additionalType)).
					Block(

						// If the struct is nil
						jen.If(jen.Id(shortName).Op("==").Nil()).
							Block(
								jen.Id(shortName).Op("=&").Id(schema.Name).Op("{}"),
							).Line().Line(),

						// If the additional properties map is nil
						jen.If(jen.Id(shortName).Dot(schema.AdditionalPropsName).Op("==").Nil()).
							Block(
								jen.Id(shortName).Dot(schema.AdditionalPropsName).Op("=").
									Make(jen.Map(jen.String()).Add(additionalType)),
							).Line().Line(),

						// Set the value by key
						jen.Id(shortName).Dot(schema.AdditionalPropsName).Index(jen.Id("key")).Op("=").Id("value"),
					).Line().Line()

				getter := jen.Null()

				if options.Comments {
					getter.Comment("// Get gets a value from the additional properties.").Line()
					getter.Comment("// If the value doesn't exist, the default value (or nil) is returned.").Line()
				}

				// The getter method
				getter.Func().
					Params(jen.Id(shortName).Op("*").Id(schema.Name)).
					Id("Get").
					Params(jen.Id("key").String()).
					Params(additionalType).
					Block(

						// If either the struct is nil or the additional
						// types is nil, return the empty value.
						jen.If(
							jen.Id(shortName).Op("==").Nil().Op("||").
								Id(shortName).Dot(schema.AdditionalPropsName).Op("==").Nil(),
						).
							Block(
								jen.Return(returnEmptyVal),
							).Line().Line(),

						// Try to get the value
						jen.If(
							jen.List(jen.Id("val"), jen.Id("ok")).Op(":=").
								Id(shortName).Dot(schema.AdditionalPropsName).Index(jen.Id("key")),
							jen.Id("ok"),
						).Block(
							jen.Return(jen.Id("val")),
						).Line().Line(),
						jen.Return(returnEmptyVal),
					).Line().Line()

				code.Add(getter, setter)
			}
		}
	}

	return code, nil
}

// GenerateSpec generates code that stores the
// specifications in base64, and a function to decode them to a map of bytes.
func (g *General) GenerateSpec(ctx context.Context, spBytes []byte, funcName string) (jen.Code, error) {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	if spBytes == nil {
		return nil, fmt.Errorf("no specification given")
	}

	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(spBytes)
	if err != nil {
		return nil, err
	}
	err = zw.Close()
	if err != nil {
		return nil, err
	}

	specB64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	c := jen.Null()

	if options.Comments && funcName != "" {
		c.Commentf("// %v returns the specification file", funcName).Line()
	}

	c.Func().Id(funcName).Params().Params(jen.Index().Byte()).Block(
		jen.Var().Id("specB64").Op("=").Lit(specB64),

		jen.List(jen.Id("b"), jen.Err()).Op(":=").Qual("encoding/base64", "StdEncoding").
			Dot("DecodeString").Call(jen.Id("specB64")),

		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Panic(jen.Err()),
		),

		jen.Var().Id("buf").Op("=").Qual("bytes", "NewBuffer").Call(jen.Id("b")),
		jen.Var().Id("outBuf").Qual("bytes", "Buffer"),

		jen.List(jen.Id("zr"), jen.Id("err")).Op(":=").Qual("compress/gzip", "NewReader").Call(jen.Id("buf")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Panic(jen.Err()),
		),
		jen.Defer().Id("zr").Op(".").Id("Close").Call(),

		jen.List(jen.Id("_"), jen.Id("err")).Op("=").Qual("io", "Copy").Call(jen.Op("&").Id("outBuf"), jen.Id("zr")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Panic(jen.Err()),
		),

		jen.Return(jen.Id("outBuf").Op(".").Id("Bytes").Call()),
	).Line().Line()

	return c, nil
}

// Calls either encoding/json or the "json" value created by jsoniter
func (g *General) jsonCall(jsoniter bool, target string) *jen.Statement {
	if jsoniter {
		return jen.Id("echo_jsonIter").Op(".").Id(target)
	}
	return jen.Qual("encoding/json", target)
}

func (g *General) GetOpts(ctx context.Context) (*GeneralOptions, error) {
	generatorOptions, ok := ctx.Value(common.ContextGeneratorOptions).(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no options provided")
	}

	rawOpts, ok := generatorOptions[g.Name()]
	if !ok {
		return nil, fmt.Errorf("no options provided")
	}

	ctxOpts := g.DefaultOptions().(*GeneralOptions)

	err := mapstructure.Decode(rawOpts, ctxOpts)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	return ctxOpts, nil
}
