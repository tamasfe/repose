package templates

import (
	"github.com/dave/jennifer/jen"
	"github.com/tamasfe/repose/pkg/util/gen"
)

// JSONMarshalAdditionalProps marshals a JSON struct
// with additional properties.
//
// Subst value examples:
//
// receiverName: e
// typeName: Example
// jsonUnmarshal: json.Unmarshal
// jsonMarshal: json.Marshal
// additionalPropsName: UnknownFields
//
var JSONMarshalAdditionalProps = `
func ({{ .receiverName }} *{{ .typeName }}) MarshalJSON() ([]byte, error) {
	if e == nil {
		return nil, nil
	}
	additional := e.{{ .additionalPropsName }}
	e.{{ .additionalPropsName }} = nil
	b, err := {{ .jsonMarshal }}(*e)
	if err != nil {
		return nil, err
	}
	var mapVal map[string]interface{}
	err = {{ .jsonUnmarshal }}(b, &mapVal)
	if err != nil {
		return nil, err
	}
	delete(mapVal, {{.additionalPropsTypeString}})
	for k := range additional {
		mapVal[k] = additional[k]
	}
	return {{ .jsonMarshal }}(mapVal)
}`[1:]

type JSONMarshalAdditionalPropsValues struct {
	ReceiverName              jen.Code
	TypeName                  jen.Code
	AdditionalPropsName       jen.Code
	JsonUnmarshal             jen.Code
	AdditionalPropsTypeString jen.Code
	JsonMarshal               jen.Code
}

func (j *JSONMarshalAdditionalPropsValues) Values() gen.Values {
	return gen.Values{
		"ReceiverName":              j.ReceiverName,
		"TypeName":                  j.TypeName,
		"AdditionalPropsName":       j.AdditionalPropsName,
		"JsonUnmarshal":             j.JsonUnmarshal,
		"AdditionalPropsTypeString": j.AdditionalPropsTypeString,
		"JsonMarshal":               j.JsonMarshal,
	}
}

func JSONMarshalAdditionalPropsDefaults() *JSONMarshalAdditionalPropsValues {
	return &JSONMarshalAdditionalPropsValues{
		ReceiverName:              jen.Id("e"),
		TypeName:                  jen.Id("Example"),
		AdditionalPropsName:       jen.Id("UnknownFields"),
		JsonUnmarshal:             jen.Qual("encoding/json", "Unmarshal"),
		AdditionalPropsTypeString: jen.Lit("UnknownFields"),
		JsonMarshal:               jen.Qual("encoding/json", "Marshal"),
	}
}

// JSONUnmarshalAdditionalProps unmarshals a JSON struct
// with additional properties.
var JSONUnmarshalAdditionalProps = `
func ({{ .receiverName }} *{{ .typeName }}) UnmarshalJSON(b []byte) error {
	knownFields := []string{{{ .knownFields }}}
	type altTp {{ .typeName }}
	var val altTp
	var additional {{ .additionalPropsType }}
	err := {{ .jsonUnmarshal }}(b, &val)
	if err != nil {
		return err
	}
	err = {{ .jsonUnmarshal }}(b, &additional)
	if err != nil {
		return err
	}
	for _, n := range knownFields {
		delete(additional, n)
	}
	val.{{ .additionalPropsName }} = additional
	originalTp := {{ .typeName }}(val)
	e = &originalTp
	return nil
}`[1:]

type JSONUnmarshalAdditionalPropsValues struct {
	ReceiverName        jen.Code
	TypeName            jen.Code
	AdditionalPropsName jen.Code
	KnownFields         jen.Code
	JsonUnmarshal       jen.Code
	AdditionalPropsType jen.Code
}

func (j *JSONUnmarshalAdditionalPropsValues) Values() gen.Values {
	return gen.Values{
		"ReceiverName":        j.ReceiverName,
		"TypeName":            j.TypeName,
		"AdditionalPropsName": j.AdditionalPropsName,
		"KnownFields":         j.KnownFields,
		"JsonUnmarshal":       j.JsonUnmarshal,
		"AdditionalPropsType": j.AdditionalPropsType,
	}
}

func JSONUnmarshalAdditionalPropsDefaults() *JSONUnmarshalAdditionalPropsValues {
	return &JSONUnmarshalAdditionalPropsValues{
		ReceiverName:        jen.Id("e"),
		TypeName:            jen.Id("Example"),
		AdditionalPropsName: jen.Id("UnknownFields"),
		KnownFields:         jen.List(jen.Lit("field1"), jen.Lit("field2")),
		JsonUnmarshal:       jen.Qual("encoding/json", "Unmarshal"),
		AdditionalPropsType: jen.Map(jen.String()).Interface(),
	}
}

// HTTPRequest template
var HTTPRequest = `
func {{ .FuncName }}({{ .Parameters }}) (*{{ .HTTPRequest }}, error) {
	var _bodyData {{ .ioReader }}
	{{ .marshalValues }}

	{{ .createURL }}

	_req, _err := http.NewRequest({{ .Method }}, {{ .URL }}, _bodyData)
	if _err != nil {
		return nil, _err
	}

	_q := _req.URL.Query()

	{{ .AdditionalStatements }}

	_req.URL.RawQuery = _q.Encode()

	return _req, nil
}`[1:]

type HTTPRequestValues struct {
	FuncName             jen.Code
	Parameters           jen.Code
	HTTPRequest          jen.Code
	MarshalValues        jen.Code
	URL                  jen.Code
	Method               jen.Code
	AdditionalStatements jen.Code
	IOReader             jen.Code
	CreateURL            jen.Code
}

func (h *HTTPRequestValues) Values() gen.Values {
	return gen.Values{
		"FuncName":             h.FuncName,
		"Parameters":           h.Parameters,
		"HTTPRequest":          h.HTTPRequest,
		"MarshalValues":        h.MarshalValues,
		"URL":                  h.URL,
		"Method":               h.Method,
		"AdditionalStatements": h.AdditionalStatements,
		"IOReader":             h.IOReader,
		"CreateURL":            h.CreateURL,
	}
}

func HTTPRequestDefaults() *HTTPRequestValues {
	return &HTTPRequestValues{
		FuncName:             jen.Id("Request"),
		Parameters:           jen.List(jen.Id("value").String(), jen.Id("anotherValue").Int()),
		HTTPRequest:          jen.Qual("net/http", "Request"),
		MarshalValues:        jen.Null(),
		URL:                  jen.Id("url"),
		Method:               jen.Lit("POST"),
		AdditionalStatements: jen.Null(),
		IOReader:             jen.Qual("io", "Reader"),
		CreateURL:            jen.Null(),
	}
}

var HTTPRespondEncoder = `
enc := {{ .NewEncoder }}({{ .WriterName }})
{{ .WriterName }}.Header().Add("Content-Type", "application/json; charset=UTF-8")
{{ .WriterName }}.WriteHeader({{ .StatusCode }})
{{ .ErrName }} := enc.Encode({{ .Value }})
{{ .HandleErr }}`[1:]

type HTTPRespondJSONValues struct {
	NewEncoder jen.Code
	WriterName jen.Code
	StatusCode jen.Code
	ErrName    jen.Code
	HandleErr  jen.Code
}

func HTTPRespondJSONDefaults() *HTTPRespondJSONValues {
	return &HTTPRespondJSONValues{
		NewEncoder: jen.Qual("encoding/json", "NewEncoder"),
		WriterName: jen.Id("w"),
		StatusCode: jen.Lit(200),
		ErrName:    jen.Err(),
		HandleErr: jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Err()),
		),
	}
}

var MarshalBytes = `
var {{ .bytesName }} []byte
if _b, err := {{ .Marshal }}({{ .value }}); err != nil {
	{{ .onError }}
} else {
	{{ .bytesName }} = _b
}`[1:]

type MarshalBytesValues struct {
	BytesName jen.Code
	Marshal   jen.Code
	OnError   jen.Code
	Value     jen.Code
}

func (m *MarshalBytesValues) Values() gen.Values {
	return gen.Values{
		"BytesName": m.BytesName,
		"Marshal":   m.Marshal,
		"Value":     m.Value,
		"OnError":   m.OnError,
	}
}

func MarshalBytesDefaults() *MarshalBytesValues {
	return &MarshalBytesValues{
		BytesName: jen.Id("b"),
		Marshal:   jen.Qual("encoding/json", "Marshal"),
		Value:     jen.Id("val"),
		OnError:   jen.Return(jen.List(jen.Nil()), jen.Err()),
	}
}
