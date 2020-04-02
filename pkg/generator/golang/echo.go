package golang

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/labstack/echo/v4"
	"github.com/mitchellh/mapstructure"
	"github.com/tamasfe/repose/internal/markdown"
	"github.com/tamasfe/repose/pkg/util"
	"github.com/tamasfe/repose/pkg/util/gen"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	"github.com/tamasfe/repose/pkg/common"
	"github.com/tamasfe/repose/pkg/spec"
)

const echoPath = "github.com/labstack/echo/v4"

// EchoOptions is the options for the Echo target.
type EchoOptions struct {
	ServerName            string `yaml:"serverName,omitempty" description:"Name of the server interface"`
	ServerImplName        string `yaml:"serverImplName,omitempty" description:"Name of the server interface implementation"`
	AllowNoResponse       bool   `yaml:"allowNoResponse" description:"Add a NoResponse value that indicates that the returned value by a handler should be ignored by the generated wrapper"`
	ServerPackagePath     string `yaml:"serverPackagePath" description:"Path to the generated server package, used for generating the scaffold, if left empty it is assumed that it is in the same package"`
	TypesPackagePath      string `yaml:"typesPackagePath" description:"Path to the generated types package, used for generating the server interface, if left empty it is assumed that it is in the same package"`
	ResponsePostfix       string `yaml:"responsePostfix" description:"Postfix to add for response types, configure it to avoid collisions with actual types"`
	ShortScaffoldComments bool   `yaml:"shortScaffoldComments" description:"Shorter scaffold comments for each method implementation"`
	ServerMiddleware      bool   `yaml:"serverMiddleware" description:"Enable the ability to add middleware to the individual operations from a method on the server interface"`
}

// MarshalYAML implements YAML Marshaler
func (e *EchoOptions) MarshalYAML() (interface{}, error) {
	return util.MarshalYAMLWithDescriptions(e)
}

// Echo provides code generation for the Echo framework
type Echo struct{}

// Generate implements Generator
func (e *Echo) Generate(ctx context.Context, options interface{}, sp *spec.Spec, target string) (interface{}, error) {
	opts := e.DefaultOptions().(*EchoOptions)

	err := mapstructure.Decode(options, opts)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	state, ok := ctx.Value(common.ContextState).(*common.State)
	if ok {
		state.PackageAlias("echo", echoPath)
	}

	switch target {
	case "server", "srv":
		return e.GenerateServer(ctx, sp, opts)
	case "server-scaffold", "scaffold", "srv-scaffold":
		return e.GenerateScaffold(ctx, sp, opts)
	default:
		return nil, fmt.Errorf("target %v is not supported", target)
	}
}

// DefaultOptions implements Generator
func (e *Echo) DefaultOptions() interface{} {
	return &EchoOptions{
		ServerName:            "Server",
		ServerImplName:        "ServerImpl",
		AllowNoResponse:       false,
		ShortScaffoldComments: false,
		ResponsePostfix:       "HandlerResponse",
		ServerMiddleware:      true,
	}
}

// Name implements Generator
func (e *Echo) Name() string {
	return "go-echo"
}

// Description implements Generator
func (e *Echo) Description() string {
	return "Generates code for the Echo framework"
}

// Targets implements Generator
func (e *Echo) Targets() map[string]string {
	return map[string]string{
		"server":          "The server interface, and the register function",
		"server-scaffold": "Scaffold for a server interface",
	}
}

// DescriptionMarkdown implements DescriptionMarkdown
func (e *Echo) DescriptionMarkdown() string {
	desc := `
# Description

This generator provides code generation for the Go [Echo](https://echo.labstack.com/) server framework.

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
			"OptionsTable": markdown.OptionsTable(*e.DefaultOptions().(*EchoOptions)),
			"OptionsExample": "```yaml\n" + string(util.MustMarshalYAML(
				map[string]interface{}{
					"go-stdlib": e.DefaultOptions(),
				},
			)) + "```\n",
			"TargetsTable": markdown.TargetsTable(e.Targets()),
		},
	)
	if err != nil {
		panic(err)
	}

	util.DisableYAMLMarshalComments = yamlComments

	return buf.String()
}

// GenerateServer generates the server interface,
// And the register function.
func (e *Echo) GenerateServer(ctx context.Context, sp *spec.Spec, opts *EchoOptions) (jen.Code, error) {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	handlers := make([]jen.Code, 0)

	for _, p := range sp.Paths {
		for _, o := range p.Operations {
			params := make([]jen.Code, 0, len(o.Parameters)+1)
			returns := make([]jen.Code, 0, 2)
			params = append(params, jen.Id("c").Qual(echoPath, "Context"))

			g := &General{}

			generalOpts, err := g.GetOpts(ctx)
			if err != nil {
				return nil, err
			}

			generalOpts.TypesPackagePath = opts.TypesPackagePath

			// If we need to process the parameters,
			// and pass them as arguments.
			for _, param := range o.Parameters {
				// We skip parameters that aren't supported.
				if !e.isParameterContentTypeSupported(param.ContentType) {
					continue
				}

				if param.Schema == nil {
					continue
				}

				paramCode := jen.Id(util.ToGoName(strcase.ToLowerCamel(param.Name)))

				var c jen.Code
				if param.Schema.Name != "" {
					c = gen.Qual(opts.TypesPackagePath, param.Schema.Name)
				} else {
					cd, err := g.GenerateType(ctx, param.Schema, generalOpts)
					if err != nil {
						return nil, err
					}
					c = cd
				}

				if param.IsPtr() {
					paramCode.Op("*")
				}
				paramCode.Add(c)

				params = append(params, paramCode)
			}

			returns = append(returns, jen.Id(o.Name+opts.ResponsePostfix), jen.Error())

			handler := jen.Line()

			if options.Comments {
				handler.Add(gen.Comments(o.Comments...))
			}

			handler.Id(strcase.ToCamel(o.Name)).Params(params...).Params(returns...)

			handlers = append(handlers, handler)
		}
	}

	if opts.ServerMiddleware {
		mwMethod := jen.Line().Line()
		if options.Comments {
			mwMethod.Commentf("// Middleware allows attaching middleware to each operation.").Line()
		}

		mwMethod.Id("Middleware").Params().Params(jen.Op("*").Id(opts.ServerName + "Middleware"))

		handlers = append(handlers, mwMethod)
	}

	resCode := jen.Null()

	if options.Comments {
		resCode.Commentf("// %v is the server interface with the handlers based on the specification.", opts.ServerName).Line()
		resCode.Commentf("// ").Line()
		resCode.Commentf("// It contains Echo handlers as its methods.").Line()
		resCode.Commentf("// To use it, implement it on a custom type,").Line()
		resCode.Commentf("// and then register it with an echo instance.").Line()
	}

	resCode.Type().Id(opts.ServerName).Interface(
		handlers...,
	)

	mwTypeCode := jen.Null()

	if opts.ServerMiddleware {
		if options.Comments {
			mwTypeCode.Commentf("// %v describes the middleware for operations of %v", opts.ServerName+"Middleware", opts.ServerName).Line()
		}

		fields := make([]jen.Code, 0, len(handlers)-1)

		for _, p := range sp.Paths {
			for _, o := range p.Operations {
				fields = append(fields,
					jen.Id(strcase.ToCamel(o.Name)).Index().Qual(echoPath, "MiddlewareFunc"),
				)
			}
		}

		mwTypeCode.Type().Id(opts.ServerName + "Middleware").Struct(fields...).Line()
	}

	// Create the register function
	wrapperCode, err := e.GenerateWrapper(ctx, sp, opts)
	if err != nil {
		return nil, err
	}

	code := jen.Null()

	returnInterfaces, err := e.generateResponses(ctx, sp, opts)
	if err != nil {
		return nil, err
	}

	return code.
		Add(resCode).Line().
		Add(mwTypeCode).Line().
		Add(wrapperCode).Line().
		Add(returnInterfaces).Line(), nil
}

func (e *Echo) GenerateScaffold(ctx context.Context, sp *spec.Spec, opts *EchoOptions) (jen.Code, error) {
	// Pretty similar to the server
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	scaffoldCode := jen.Null()

	scaffoldCode.Commentf("// repose:keep server_def").Line()
	if options.Comments {
		scaffoldCode.Commentf("// The struct used for implementing %v.", opts.ServerName).Line()
		scaffoldCode.Commentf("// Repose relies on the name, make sure to keep it updated in its config.").Line()
	}
	scaffoldCode.Type().Id(opts.ServerImplName).Struct().Line()
	scaffoldCode.Comment("// repose:endkeep").Line().Line()

	if options.Comments {
		scaffoldCode.Commentf("// Make sure that we implement the correct server.").Line()
	}
	scaffoldCode.Var().Id("_").Add(gen.Qual(opts.ServerPackagePath, opts.ServerName)).Op("=&").Id(opts.ServerImplName).Block().Line().Line()

	for _, p := range sp.Paths {
		for _, o := range p.Operations {
			params := make([]jen.Code, 0, len(o.Parameters)+1)
			returns := make([]jen.Code, 0, 2)
			params = append(params, jen.Id("c").Qual(echoPath, "Context"))

			g := &General{}
			generalOpts, err := g.GetOpts(ctx)
			if err != nil {
				return nil, err
			}

			generalOpts.TypesPackagePath = opts.TypesPackagePath
			for _, param := range o.Parameters {
				// We skip parameters that aren't supported.
				if !e.isParameterContentTypeSupported(param.ContentType) {
					continue
				}

				if param.Schema == nil {
					continue
				}

				paramCode := jen.Id(util.ToGoName(strcase.ToLowerCamel(param.Name)))

				if param.IsPtr() {
					paramCode.Op("*")
				}

				if param.Schema.Name != "" {
					paramCode.Add(gen.Qual(opts.ServerPackagePath, param.Schema.Name))
				} else {
					c, err := g.GenerateType(ctx, param.Schema, generalOpts)
					if err != nil {
						return nil, err
					}

					paramCode.Add(c)
				}

				params = append(params, paramCode)
			}

			returns = append(returns, gen.Qual(opts.ServerPackagePath, o.Name+opts.ResponsePostfix), jen.Error())

			if options.Comments {
				if opts.ShortScaffoldComments {
					scaffoldCode.Add(gen.Comments(o.Comments[0]))
				} else {
					scaffoldCode.Add(gen.Comments(o.Comments...))
				}
			}

			scaffoldCode.Func().Params(jen.Id(strings.ToLower(opts.ServerImplName[:1])).Id("*"+opts.ServerImplName)).
				Id(strcase.ToCamel(o.Name)).
				Params(params...).
				Params(returns...).Block(
				jen.Commentf("// repose:keep "+o.Name+"_body"),
				jen.Panic(jen.Lit("unimplemented")),
				jen.Commentf("// repose:endkeep"),
			).Line().Line()
		}
	}

	if opts.ServerMiddleware {
		if options.Comments {
			scaffoldCode.Commentf("// Middleware allows attaching middleware to each operation.").Line()
		}

		scaffoldCode.Func().Params(jen.Id(strings.ToLower(opts.ServerImplName[:1])).Id("*"+opts.ServerImplName)).
			Id("Middleware").Params().Params(jen.Op("*").Add(gen.Qual(opts.ServerPackagePath, opts.ServerName+"Middleware"))).
			Block(
				jen.Commentf("// repose:keep middleware_body"),
				jen.Panic(jen.Lit("unimplemented")),
				jen.Commentf("// repose:endkeep"),
			).Line().Line()
	}

	return scaffoldCode, nil
}

// Checks whether the parameter content-type is supported, and should be handled.
func (e *Echo) isParameterContentTypeSupported(contentType string) bool {
	ct := strings.TrimSpace(strings.ToLower(contentType))

	if ct == "" {
		return true
	}

	mimeTypes := []string{
		echo.MIMEApplicationJSON,
		echo.MIMEApplicationXML,
		echo.MIMEApplicationForm,
	}

	for _, mt := range mimeTypes {
		if strings.HasPrefix(ct, mt) {
			return true
		}
	}

	return false
}

// GenerateWrapper generates wrapper for an Echo instance
// and the server interface.
func (e *Echo) GenerateWrapper(ctx context.Context, sp *spec.Spec, opts *EchoOptions) (jen.Code, error) {
	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	c := jen.Null()

	if options.Comments {
		c.Comment("// EchoInstance is required to add handlers to").Line()
		c.Comment("// to both Echo instances and Echo groups.").Line()
	}

	// EchoInstance interface
	c.Type().Id("EchoInstance").Interface(
		jen.Id("Add").Params(
			jen.String(),
			jen.String(),
			jen.Qual(echoPath, "HandlerFunc"),
			jen.Op("...").Qual(echoPath, "MiddlewareFunc"),
		).Params(jen.Op("*").Qual(echoPath, "Route")),
	).Line().Line()

	funcHeader := jen.Null()

	if options.Comments {
		funcHeader.Commentf("// RegisterEchoServer registers a %v with an Echo instance,", opts.ServerName).Line()
		funcHeader.Commentf("// and wraps %v's handlers with Echo handlers.", opts.ServerName).Line()
		funcHeader.Comment("// Depending on the options, parameter parsing").Line()
		funcHeader.Comment("// and response encoding are done in the wrapper.").Line()
		funcHeader.Comment("// ").Line()
		funcHeader.Comment("// Note that the parameters and responses are NOT validated by the wrapper.").Line()
		funcHeader.Comment("// The value of an invalid parameter will be the default Go value.").Line()
		funcHeader.Comment("// If you need to do validation, do it in a middleware,").Line()
		funcHeader.Commentf("// or in %v's methods.", opts.ServerName).Line()

	}

	funcHeader.Func().Id("RegisterEchoServer").Params(
		jen.Id("e").Id("EchoInstance"),
		jen.Id("server").Id(opts.ServerName),
	)

	funcBody := make([]jen.Code, 0)

	// If we have middleware declared, we need to
	// handle them.
	if opts.ServerMiddleware {
		funcBody = append(funcBody,
			jen.Id("middleware").Op(":=").Id("server").Dot("Middleware").Call().Line(),
			jen.If(jen.Id("middleware").Op("==").Nil()).Block(
				jen.Id("middleware").Op("=").New(jen.Id(opts.ServerName+"Middleware")),
			).Line().Line(),
		)
	}

	for _, p := range sp.Paths {
		// the parameters are expected like :param
		pathStr := util.ParamStyleToColon(p.PathString)

		// create and register a handler for each operation
		for _, o := range p.Operations {
			handler := jen.Func().Params(jen.Id("c").Qual(echoPath, "Context")).Params(jen.Error())

			paramNames := make([]jen.Code, 0, 1+len(o.Parameters))

			// The first parameter that is passed is always the echo context
			paramNames = append(paramNames, jen.Id("c"))

			// The body of the wrapper handler func before the
			// wrapped handler is called.
			beforeStatements := make([]jen.Code, 0, len(o.Parameters))

			for _, param := range o.Parameters {
				// We skip parameters that aren't supported.
				if !e.isParameterContentTypeSupported(param.ContentType) {
					continue
				}

				paramC := jen.Null()

				c, err := e.generateExtractParam(ctx, param, opts)
				if err != nil {
					return nil, err
				}
				if c != nil {
					paramC.Add(c)
					paramNames = append(paramNames, jen.Id(param.Name))
				}

				beforeStatements = append(beforeStatements, paramC)
			}

			callResultVars := jen.Null()
			callResultVars.List(jen.Id("result"), jen.Err())

			handleResponse := jen.Null()
			handleResponse.Add(gen.MustTemplate(`return result.{{ .InfName }}(c)`,
				gen.Values{
					"InfName": jen.Id(o.Name + opts.ResponsePostfix),
				},
			)).Line()

			handlerCall := gen.MustTemplate(`{{ .CallResultVars }} := server.{{ .Handler }}({{ .Params }})
				if err != nil {
					return err
				}
				{{ .HandleResponse }}`,
				gen.Values{
					"Handler":        jen.Id(strcase.ToCamel(o.Name)),
					"CallResultVars": callResultVars,
					"Params":         jen.List(paramNames...),
					"HandleResponse": handleResponse,
				},
			)

			// Then finally we can Append the body statements
			// to the handler header
			statements := make([]jen.Code, 0)

			statements = append(statements, beforeStatements...)
			statements = append(statements, handlerCall)

			handler.Block(statements...)

			// If we have middleware, add them.
			addMws := jen.Null()

			if opts.ServerMiddleware {
				addMws.Id("middleware").Dot(strcase.ToCamel(o.Name)).Op("...")
			}

			funcBody = append(funcBody,
				jen.Id("e").Op(".").Id("Add").Call(
					jen.Lit(strings.ToUpper(o.Method)),
					jen.Lit(pathStr),
					handler,
					addMws,
				).Line(),
			)
		}
	}
	return c.Add(funcHeader.Block(funcBody...)), nil
}

func (e *Echo) generateExtractParam(ctx context.Context, param *spec.Parameter, opts *EchoOptions) (jen.Code, error) {
	// TODO implement arrays and objects

	g := &General{}

	generalOpts, err := g.GetOpts(ctx)
	if err != nil {
		return nil, err
	}

	generalOpts.TypesPackagePath = opts.TypesPackagePath

	paramC := jen.Null()

	var pType jen.Code
	if param.Schema.Name != "" {
		pType = gen.Qual(opts.TypesPackagePath, param.Schema.Name)
	} else {
		pt, err := g.GenerateType(ctx, param.Schema, generalOpts)
		if err != nil {
			return nil, err
		}
		pType = pt
	}

	if param.IsPtr() {
		pType = jen.Op("=").New(pType)
	}

	paramC.Var().Id(param.Name).Add(pType).Line()

	paramName := param.Name

	// Extract the parameter based on its location/type.
	// Errors are not checked, a separate validation middleware
	// should check for the correctness of the request if needed.
	//
	// If an error happens, the Go default value is passed to the handler.
	switch param.Type {
	case spec.ParameterTypePath:

		switch param.Schema.Variant {
		case spec.VariantPrimitive:
			switch param.Serialization.Style {
			case spec.SerializationSimple:
				c, err := gen.PrimitiveFromString(
					param.Schema,
					param.IsPtr(),
					jen.Id(param.Name),
					jen.Id("c").Dot("Param").Call(jen.Lit(param.Name)),
				)
				if err != nil {
					return nil, err
				}
				paramC.Add(c).Line().Line()
			case spec.SerializationLabel:
				// .paramName
				prefixLen := 1

				c, err := gen.PrimitiveFromString(
					param.Schema,
					param.IsPtr(),
					jen.Id(param.Name),
					jen.Id("c").Dot("Param").Call(jen.Lit(param.Name).Index(jen.Lit(prefixLen).Op(":"))),
				)
				if err != nil {
					return nil, err
				}
				paramC.If(gen.Raw("len(c.Param(\"" + param.Name + "\")) > " + strconv.Itoa(prefixLen))).Block(
					c,
				).Line().Line()
			case spec.SerializationMatrix:
				// ;paramName=
				prefixLen := len(param.Name) + 2

				c, err := gen.PrimitiveFromString(
					param.Schema,
					param.IsPtr(),
					jen.Id(param.Name),
					jen.Id("c").Dot("Param").Call(jen.Lit(param.Name).Index(jen.Lit(prefixLen).Op(":"))),
				)
				if err != nil {
					return nil, err
				}
				paramC.If(gen.Raw("len(c.Param(\"" + param.Name + "\")) > " + strconv.Itoa(prefixLen))).Block(
					c,
				).Line().Line()
			}
		case spec.VariantArray:
			switch param.Serialization.Style {
			case spec.SerializationSimple:
				c, err := gen.PrimitiveFromString(
					param.Schema.Children.GetSchema(),
					param.Schema.Children.GetSchema().ShouldBePtr(),
					jen.Id("_param"),
					jen.Id("_s"),
				)
				if err != nil {
					return nil, err
				}

				arrType, err := g.GenerateType(ctx, param.Schema.Children.GetSchema(), generalOpts)
				if err != nil {
					return nil, err
				}

				arrayC, err := gen.Template(
					`
					for _, _s := range {{ .ParamArr }} {
						var _param {{ .paramType }}
						{{ .deserialize }}
						{{ .paramName }} = append({{ .paramName }}, _param)
					}`[1:],
					gen.Values{
						"paramType":   jen.Add(arrType),
						"deserialize": c,
						"paramName":   jen.Id(param.Name),
						"paramArr": jen.Qual("strings", "Split").Call(
							jen.Id("c").Dot("Param").Call(jen.Lit(param.Name)),
							jen.Lit(","),
						),
					},
				)
				if err != nil {
					return nil, err
				}

				paramC.Add(arrayC).Line().Line()
			}
		}
	case spec.ParameterTypeCookie:
		switch param.Schema.Variant {
		case spec.VariantPrimitive:
			c, err := gen.PrimitiveFromString(
				param.Schema,
				param.IsPtr(),
				jen.Id(param.Name),
				jen.Id("c").Dot("Cookie").Call(jen.Lit(param.Name)),
			)
			if err != nil {
				return nil, err
			}
			paramC.Add(c).Line().Line()
		}

	case spec.ParameterTypeBody:
		addrOp := jen.Null()
		if !param.IsPtr() {
			addrOp.Op("&")
		}

		// TODO this has to be changed, as the body is not always required.

		// We use Echo's binder to bind the value to its type.
		paramC.Id("_").Op("=").Id("c").Op(".").Id("Bind").Call(addrOp.Id(paramName)).
			Line().Line()

	case spec.ParameterTypeHeader:
		c, err := gen.PrimitiveFromString(
			param.Schema,
			param.IsPtr(),
			jen.Id(param.Name),
			jen.Id("c").Dot("Header").Dot("Get").Call(jen.Lit(param.Name)),
		)
		if err != nil {
			return nil, err
		}
		paramC.Add(c).Line().Line()

	case spec.ParameterTypeQuery:
		switch param.Schema.Variant {
		case spec.VariantPrimitive:
			c, err := gen.PrimitiveFromString(
				param.Schema,
				param.IsPtr(),
				jen.Id(param.Name),
				jen.Id("c").Dot("QueryParam").Call(jen.Lit(param.Name)),
			)
			if err != nil {
				return nil, err
			}
			paramC.Add(c).Line().Line()
		case spec.VariantStruct:
			for field, fieldSchema := range param.Schema.Children.GetMap() {
				c, err := gen.PrimitiveFromString(
					fieldSchema,
					fieldSchema.ShouldBePtr() && !fieldSchema.CanBeNil() || fieldSchema.Nullable,
					jen.Id(param.Name).Dot(field),
					jen.Id("c").Dot("QueryParam").Call(jen.Lit(fieldSchema.FieldName)),
				)
				if err != nil {
					return nil, err
				}
				paramC.Add(c).Line().Line()
			}

		case spec.VariantArray:
			c, err := gen.PrimitiveFromString(
				param.Schema.Children.GetSchema(),
				param.Schema.Children.GetSchema().ShouldBePtr(),
				jen.Id("_param"),
				jen.Id("_s"),
			)
			if err != nil {
				return nil, err
			}

			arrType, err := g.GenerateType(ctx, param.Schema.Children.GetSchema(), generalOpts)
			if err != nil {
				return nil, err
			}

			arrayC, err := gen.Template(
				`
				for _, _s := range {{ .ParamArr }} {
						var _param {{ .paramType }}
						{{ .deserialize }}
						{{ .paramName }} = append({{ .paramName }}, _param)
					}`[1:],
				gen.Values{
					"paramType":   jen.Add(arrType),
					"deserialize": c,
					"paramName":   jen.Id(param.Name),
					"paramArr": jen.Qual("strings", "Split").Call(
						jen.Id("c").Dot("QueryParam").Call(jen.Lit(param.Name)),
						jen.Lit(","),
					),
				},
			)
			if err != nil {
				return nil, err
			}

			paramC.Add(arrayC).Line().Line()
		}

	}

	return paramC, nil
}

func (e *Echo) generateResponses(ctx context.Context, sp *spec.Spec, opts *EchoOptions) (jen.Code, error) {
	resC := jen.Null()

	options, ok := ctx.Value(common.ContextCommonOptions).(*common.Options)
	if !ok {
		options = common.DefaultOptions()
	}

	if opts.AllowNoResponse {

		resC.Type().Id("noResponse").String().Line().Line()
		if options.Comments {
			resC.Commentf("// NoResponse indicate that the response should not be handled by the generated code.").Line()
		}
		resC.Const().Id("NoResponse").Id("noResponse").Op("=").Lit("").Line().Line()
	}

	for _, p := range sp.Paths {
		for _, o := range p.Operations {

			if options.Comments {
				resC.Commentf("// %v defines responses for the %v operation.", o.Name+opts.ResponsePostfix, o.Name).Line()
			}
			resC.Type().Id(o.Name + opts.ResponsePostfix).Interface(
				jen.Id(o.Name + opts.ResponsePostfix).Params(jen.Qual(echoPath, "Context")).Params(jen.Error()),
			).Line().Line()

			if opts.AllowNoResponse {
				resC.Func().Params(jen.Id("n").Id("noResponse")).
					Id(o.Name + opts.ResponsePostfix).
					Params(jen.Id("ctx").Qual(echoPath, "Context")).Params(jen.Error()).
					Block(jen.Return(jen.Nil())).Line().Line()
			}

			for _, res := range o.Responses {
				// TODO default and range responses
				if strings.ToLower(strings.TrimSpace(res.Code)) == "default" ||
					strings.Contains(strings.ToLower(res.Code), "x") {
					continue
				}

				// The response is empty
				if res.Schema == nil {

					emptyResName := "res" + o.Name + res.Code
					if res.Name != "" {
						emptyResName = "res" + strings.Title(res.Name)
					}

					resC.Type().Id(emptyResName).String().Line().Line()

					if options.Comments {
						resC.Commentf("// %v defines an empty response for the %v operation.", strings.Title(res.Name), o.Name).Line()
					}
					resC.Const().Id(strings.Title(res.Name)).Id(emptyResName).Op("=").Lit("").Line().Line()

					resC.Func().Params(jen.Id("r").Id(emptyResName)).
						Id(o.Name+opts.ResponsePostfix).
						Params(jen.Id("ctx").Qual(echoPath, "Context")).Params(jen.Error()).
						Block(
							jen.Id("ctx").Op(".").Id("NoContent").Call(jen.Lit(util.MustParseInt(res.Code))),
							jen.Return(jen.Nil()),
						).Line().Line()

					continue
				}

				// We can't handle unnamed schemas
				if res.Schema.Name == "" {
					continue
				}

				var rTypeName string
				if res.IsPtr() {
					rTypeName = "*" + res.Schema.Name
				} else {
					rTypeName = res.Schema.Name
				}

				resCode, err := e.generateResponseInterfaceBody(ctx, res, opts)
				if err != nil {
					return nil, err
				}

				if options.Comments {
					resC.Add(gen.Comments(
						fmt.Sprintf("%v is implemented for %v so that it can be used in a response.",
							o.Name+opts.ResponsePostfix,
							res.Schema.Name,
						),
					))
				}
				resC.Func().Params(jen.Id(strings.ToLower(res.Schema.Name[:1])).Id(rTypeName)).
					Id(o.Name + opts.ResponsePostfix).
					Params(jen.Id("ctx").Qual(echoPath, "Context")).Params(jen.Error()).
					Block(resCode).Line().Line()
			}
		}
	}

	return resC, nil
}

func (e *Echo) generateResponseInterfaceBody(ctx context.Context, res *spec.Response, opts *EchoOptions) (jen.Code, error) {
	// It is assumed that echo context is named "ctx"

	resStatus := util.MustParseInt(res.Code)

	resCode := jen.Null()

	rName := strings.ToLower(res.Schema.Name[:1])

	ptrCheck := jen.Null()

	if res.IsPtr() {
		c := gen.MustTemplate(`if {{ .Value }} == nil {
				ctx.NoContent({{ .Status }})
				return nil
			}`,
			gen.Values{
				"EmptyResponse": ptrCheck,
				"Status":        jen.Lit(resStatus),
				"Value":         jen.Id(rName),
			},
		)

		ptrCheck.Add(c).Line().Line()
	}

	switch {
	case strings.HasPrefix(res.ContentType, "application/json"):

		c := gen.MustTemplate(`{{ .EmptyResponse }}
		err := ctx.JSON({{ .Status }}, {{ .Value }})
			return err`,
			gen.Values{
				"EmptyResponse": ptrCheck,
				"Status":        jen.Lit(resStatus),
				"Value":         jen.Id(rName),
			},
		)

		resCode.Add(c)

	case strings.HasPrefix(res.ContentType, "application/xml"),
		strings.HasPrefix(res.ContentType, "text/xml"):

		c := gen.MustTemplate(`{{ .EmptyResponse }}
		err := ctx.XML({{ .Status }}, {{ .Value }})
		return err`,
			gen.Values{
				"EmptyResponse": ptrCheck,
				"Status":        jen.Lit(resStatus),
				"Value":         jen.Id(rName),
			},
		)

		resCode.Add(c)

	case strings.HasPrefix(res.ContentType, "text/plain"),
		strings.HasPrefix(res.ContentType, "text/plain"):

		c := gen.MustTemplate(`{{ .EmptyResponse }}
		err := ctx.String({{ .Status }}, {{ .Value }})
		return err`,
			gen.Values{
				"EmptyResponse": ptrCheck,
				"Status":        jen.Lit(resStatus),
				"Value":         jen.Qual("fmt", "Sprint").Call(jen.Id(rName)),
			},
		)

		resCode.Add(c)

	default:
		return nil, fmt.Errorf("MIME type %v not supported", res.ContentType)
	}

	return resCode, nil
}
