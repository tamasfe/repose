package golang

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/dave/jennifer/jen"
	"github.com/mitchellh/mapstructure"
	"github.com/tamasfe/repose/internal/markdown"
	"github.com/tamasfe/repose/pkg/common"
	"github.com/tamasfe/repose/pkg/spec"
	"github.com/tamasfe/repose/pkg/util"
	"github.com/tamasfe/repose/pkg/util/gen"
	"github.com/tamasfe/repose/pkg/util/gen/templates"
)

// StdLib generates code for the standard library.
type StdLib struct{}

type StdLibOptions struct {
	TypesPackagePath string `yaml:"typesPackagePath" description:"Path to the generated types package, if left empty it is assumed that it is in the same package"`
}

// Name implements Target
func (s *StdLib) Name() string {
	return "go-stdlib"
}

// Description implements Target
func (s *StdLib) Description() string {
	return "Generates code for the standard Go HTTP library"
}

// DescriptionMarkdown implements DescriptionMarkdown
func (s *StdLib) DescriptionMarkdown() string {
	desc := `
# Description

This generator generates code that only relies on the standard library.

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
			"OptionsTable": markdown.OptionsTable(*s.DefaultOptions().(*StdLibOptions)),
			"OptionsExample": "```yaml\n" + string(util.MustMarshalYAML(
				map[string]interface{}{
					"go-stdlib": s.DefaultOptions(),
				},
			)) + "```\n",
			"TargetsTable": markdown.TargetsTable(s.Targets()),
		},
	)
	if err != nil {
		panic(err)
	}

	util.DisableYAMLMarshalComments = yamlComments

	return buf.String()
}

// Targets implements Target
func (s *StdLib) Targets() map[string]string {
	return map[string]string{
		"client":    "Generate Go HTTP Requests",
		"callbacks": "Generate Go HTTP Requests for callbacks",
	}
}

// DefaultOptions implements Target
func (s *StdLib) DefaultOptions() interface{} {
	return &StdLibOptions{
		TypesPackagePath: "",
	}
}

// Generate implements Target
func (s *StdLib) Generate(ctx context.Context, options interface{}, specification *spec.Spec, target string) (interface{}, error) {
	opts := s.DefaultOptions().(*StdLibOptions)

	err := mapstructure.Decode(options, opts)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	switch target {
	case "c", "client", "clients":
		return s.GenerateClient(ctx, specification, opts)
	case "cb", "callback", "callbacks":
		return s.GenerateCallbacks(ctx, specification, opts)
	default:
		return nil, fmt.Errorf("Target %v is not supported", target)
	}
}

// GenerateClient generates Go HTTP requests.
func (s *StdLib) GenerateClient(ctx context.Context, specification *spec.Spec, opts *StdLibOptions) (jen.Code, error) {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	code := jen.Null()

	for _, p := range specification.Paths {

		clientStructName := "client" + p.Name

		code.Type().Id(clientStructName).Struct(
			jen.Id("server").String(),
		).Line().Line()

		if options.Comments {
			code.Commentf("// %v provides client requests for \"%v\".",
				p.Name+"Client",
				p.PathString,
			).Line()
		}
		code.Func().Id(p.Name + "Client").Params(
			jen.Id("server").String(),
		).Params(jen.Id(clientStructName)).Block(
			jen.Return(jen.Id(clientStructName).Values(
				jen.Dict{
					jen.Id("server"): jen.Id("server"),
				},
			)),
		).Line().Line()

		for _, o := range p.Operations {

			fName := jen.Params(jen.Id("c").Id(clientStructName)).Id(o.Name)

			if options.Comments {
				code.Commentf("// %v provides client request for the operation.",
					o.Name,
				).Line()
			}
			req, err := s.GenerateRequest(ctx, fName, jen.Id("c").Op(".").Id("server"), p.PathString, o, opts)
			if err != nil {
				return nil, err
			}

			code.Add(req).Line().Line()

		}
	}

	return code, nil
}

// GenerateClient generates Go HTTP requests.
func (s *StdLib) GenerateCallbacks(ctx context.Context, specification *spec.Spec, opts *StdLibOptions) (jen.Code, error) {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	code := jen.Null()

	for _, p := range specification.Paths {

		for _, o := range p.Operations {

			callbacksStructName := "callbacks" + o.Name
			if len(o.Callbacks) != 0 {

				code.Type().Id(callbacksStructName).Struct().Line().Line()

				if options.Comments {
					code.Commentf("// %v provides client requests for the callbacks of \"%v\".",
						p.Name+"Client",
						p.PathString,
					).Line()
				}
				code.Func().Id(o.Name + "Callbacks").Params().Params(jen.Id(callbacksStructName)).Block(
					jen.Return(jen.Id(callbacksStructName).Block()),
				).Line().Line()

			}

			for _, cb := range o.Callbacks {
				for _, cbPath := range cb {
					for _, cbOp := range cbPath.Operations {
						fName := jen.Params(jen.Id("c").Id(callbacksStructName)).Id(cbOp.Name)

						if options.Comments {
							code.Commentf("// %v provides client request for the callback operation.",
								o.Name,
							).Line()
						}
						req, err := s.GenerateRequest(ctx, fName, nil, "", cbOp, opts)
						if err != nil {
							return nil, err
						}

						code.Add(req).Line().Line()
					}
				}
			}
		}
	}

	return code, nil
}

func (s *StdLib) GenerateRequest(ctx context.Context, funcName jen.Code, url jen.Code, path string, op *spec.Operation, opts *StdLibOptions) (jen.Code, error) {
	templOpts := templates.HTTPRequestDefaults()

	additionalStatements := jen.Null()
	params := make([]jen.Code, 0, len(op.Parameters))
	marshalValues := jen.Null()

	urlName := "url"

	// Avoid name conflict
	for _, p := range op.Parameters {
		if p.Name == "url" {
			urlName = "_reqURL"
		}
	}

	urlCode := jen.Null()

	if url != nil {
		urlCode.Id(urlName).Op(":=").Add(url).Line()
	} else {
		params = append(params, jen.Id(urlName).String())
	}

	if path != "" {
		urlCode.Add(jen.Id(urlName).Op("+=").Lit(path)).Line()
	}

	g := &General{}
	generalOpts, err := g.GetOpts(ctx)
	if err != nil {
		return nil, err
	}

	generalOpts.TypesPackagePath = opts.TypesPackagePath

	for _, p := range op.Parameters {

		argCode := jen.Id(p.Name)

		if p.Schema.Name != "" {
			argCode.Add(gen.Qual(opts.TypesPackagePath, p.Schema.Name))
		} else {
			tp, err := g.GenerateType(ctx, p.Schema, generalOpts)
			if err != nil {
				return nil, err
			}
			argCode.Add(tp)
		}

		var encoder string
		switch {
		case strings.HasPrefix(p.ContentType, "application/json"):
			encoder = "encoding/json"
		case strings.HasPrefix(p.ContentType, "application/xml"):
			encoder = "encoding/xml"
		}

		mTemplateValues := templates.MarshalBytesDefaults()

		dataName := p.Name + "Data"
		mTemplateValues.BytesName = jen.Id(dataName)
		mTemplateValues.Marshal = jen.Qual(encoder, "Marshal")
		mTemplateValues.Value = jen.Id(p.Name)

		var marshalCode jen.Code
		if encoder == "" {
			switch p.Schema.Variant {
			case spec.VariantPrimitive:
				marshalCode = jen.Id(dataName).Op(":=").Qual("fmt", "Sprint").Call(jen.Id(p.Name))

			case spec.VariantArray:
				c, err := gen.Template(`
				var {{ .paramArrName }} []string
				for _, _p := range {{ .paramName }} {
					{{ .paramArrName }} = append({{ .paramArrName }}, {{ .sprintP }})
				}
				{{ .paramData }} := {{ .stringsJoin }}`[1:],
					gen.Values{
						"paramArrName": jen.Id("_" + p.Name + "Arr"),
						"paramName":    jen.Id(p.Name),
						"sprintP":      jen.Qual("fmt", "Sprint").Call(jen.Id("_p")),
						"paramData":    jen.Id(dataName),
						"stringsJoin":  jen.Qual("strings", "Join").Call(jen.Id("_"+p.Name+"Arr"), jen.Lit(",")),
					},
				)
				if err != nil {
					return nil, err
				}
				marshalCode = c
			}
		} else {
			marshalCode = gen.MustTemplate(
				templates.MarshalBytes,
				mTemplateValues,
			)
		}

		switch p.Type {
		case spec.ParameterTypeBody:
			var newBuf jen.Code
			if encoder == "" {
				newBuf = jen.Qual("bytes", "NewStringBuffer")
			} else {
				newBuf = jen.Qual("bytes", "NewBuffer")
			}

			marshalValues.Add(marshalCode).Line().
				Id("_bodyData").Op("=").Add(newBuf).Call(jen.Id(dataName)).
				Line().Line()

			additionalStatements.Id("_req").Op(".").Id("Header").Op(".").Id("Add").Call(jen.Lit("Content-Type"), jen.Lit(p.ContentType)).Line()

		case spec.ParameterTypeCookie:
			marshalValues.Add(marshalCode).Line()

			additionalStatements.Id("_req").Op(".").Id("AddCookie").Call(
				jen.Op("&").Qual("net/http", "Cookie").Values(
					jen.Dict{
						jen.Id("Name"):  jen.Lit(p.Name),
						jen.Id("Value"): jen.String().Call(jen.Id(dataName)),
					},
				),
			).Line()
		case spec.ParameterTypeHeader:
			marshalValues.Add(marshalCode).Line()
			additionalStatements.Id("_req").Op(".").Id("Header").Op(".").Id("Add").Call(jen.Lit(p.Name), jen.String().Call(jen.Id(dataName))).Line()
		case spec.ParameterTypePath:
			marshalValues.Add(marshalCode).Line()
			urlCode.Id(urlName).Op("=").Qual("strings", "Replace").Call(jen.Id(urlName), jen.Lit("{"+p.Name+"}"), jen.String().Call(jen.Id(dataName)), jen.Lit(1)).Line()
		case spec.ParameterTypeQuery:
			marshalValues.Add(marshalCode).Line()
			additionalStatements.Id("_req").Op(".").Id("URL").Op(".").Id("Query").Call().Op(".").Id("Set").Call(jen.Lit(p.Name), jen.String().Call(jen.Id(dataName))).Line()
		}

		params = append(params, argCode)
	}

	templOpts.MarshalValues = marshalValues
	templOpts.Parameters = jen.List(params...)
	templOpts.AdditionalStatements = additionalStatements
	templOpts.URL = jen.Id(urlName)
	templOpts.Method = jen.Lit(op.Method)
	templOpts.FuncName = funcName
	templOpts.CreateURL = urlCode

	return gen.Template(templates.HTTPRequest, templOpts)
}
