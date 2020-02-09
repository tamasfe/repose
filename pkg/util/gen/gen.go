// Package gen contains (mostly) generated Jennifer code,
// that can be used at multiple places.
//
// Generating code that generates code from existing code
// is useful, because we just have to write regular code.
//
// The code is generated with "github.com/aloder/tojen",
// and then modified by hand.
package gen

import (
	"fmt"
	"strings"

	jen "github.com/dave/jennifer/jen"
	"github.com/mitchellh/go-wordwrap"
	"github.com/tamasfe/repose/pkg/spec"
)

// Comments creates comments from a list of strings
func Comments(comments ...string) jen.Code {
	comments = strings.Split(wordwrap.WrapString(strings.Join(comments, "\n"), 80), "\n")
	code := jen.Null()
	for _, c := range comments {
		code.Comment("// " + c).Line()
	}
	return code
}

func Qual(path, name string) *jen.Statement {
	if path == "" {
		return jen.Id(name)
	}

	return jen.Qual(path, name)
}

func Raw(str string) *jen.Statement {
	return jen.Op(str)
}

func PrimitiveFromString(s *spec.Schema, ptr bool, varName, strName jen.Code) (jen.Code, error) {

	var assignRight jen.Code
	if ptr {
		assignRight = jen.Op("&").Id("_v")
	} else {
		assignRight = jen.Id("_v")
	}

	switch s.PrimitiveType {
	case "string":
		return jen.Null().Add(varName).Op("=").Add(strName), nil
	case "int":
		return Template(`
			if _parsedVal, err := {{ .parseInt }}({{ .strName }}, 10, 64); err == nil {
				_v := int(_parsedVal)
				{{ .varName }} = {{ .assignRight }}
			}`[1:],
			Values{
				"parseInt":    jen.Qual("strconv", "ParseInt"),
				"strName":     strName,
				"varName":     varName,
				"assignRight": assignRight,
			},
		)
	case "int32":
		return Template(`
			if _parsedVal, err := {{ .parseInt }}({{ .strName }}, 10, 32); err == nil {
				_v := int32(_parsedVal)
				{{ .varName }} = {{ .assignRight }}
			}`[1:],
			Values{
				"parseInt":    jen.Qual("strconv", "ParseInt"),
				"strName":     strName,
				"varName":     varName,
				"assignRight": assignRight,
			},
		)
	case "int64":
		return Template(`
			if _parsedVal, err := {{ .parseInt }}({{ .strName }}, 10, 64); err == nil {
				_v := _parsedVal
				{{ .varName }} = {{ .assignRight }}
			}`[1:],
			Values{
				"parseInt":    jen.Qual("strconv", "ParseInt"),
				"strName":     strName,
				"varName":     varName,
				"assignRight": assignRight,
			},
		)
	case "bool":
		return Template(`
		if _parsedVal, err := {{ .ParseBool }}({{ .strName }}); err == nil {
			_v := _parsedVal
			{{ .varName }} = {{ .assignRight }}
		}`[1:],
			Values{
				"ParseBool":   jen.Qual("strconv", "ParseBool"),
				"strName":     strName,
				"varName":     varName,
				"assignRight": assignRight,
			},
		)
	case "float32":
		return Template(`
		if _parsedVal, err := {{ .ParseFloat }}({{ .strName }}, 32); err == nil {
			_v := float32(_parsedVal)
			{{ .varName }} = {{ .assignRight }}
		}`[1:],
			Values{
				"ParseFloat":  jen.Qual("strconv", "ParseFloat"),
				"strName":     strName,
				"varName":     varName,
				"assignRight": assignRight,
			},
		)
	case "float64":
		return Template(`
		if _parsedVal, err := {{ .ParseFloat }}({{ .strName }}, 64); err == nil {
			_v := _parsedVal
			{{ .varName }} = {{ .assignRight }}
		}`[1:],
			Values{
				"ParseFloat":  jen.Qual("strconv", "ParseFloat"),
				"strName":     strName,
				"varName":     varName,
				"assignRight": assignRight,
			},
		)
	default:
		return nil, fmt.Errorf("not a primitive type")
	}
}
