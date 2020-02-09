package spec

// Spec is an abstraction over a specification.
//
// Due to the ever-changing specificaitions,
// we'd have to rewrite code every few months
// in order to support more versions.
//
// Instead of that, we create an abstraction with
// the information of just what we need.
type Spec struct {
	Paths []*Path `json:"paths"`
	// Schemas used in the specification
	Schemas []*Schema `json:"schemas"`
}

// Path is a HTTP REST-like path.
type Path struct {
	// PathString is the original string of the path
	// like "/pets/{id}/profile"
	PathString string `json:"pathString"`

	// Name of the path, either automatically generated
	// from the path string, or given in an extension.
	Name string `json:"name"`

	// Description of the path if any.
	Description string `json:"description"`

	// Additional comments for the path, if any.
	Comments []string `json:"comments"`

	// Operations of the path
	Operations []*Operation `json:"operations"`
}

// Operation is a HTTP operation.
type Operation struct {
	// Name of the operation if any.
	Name string `json:"name"`

	// The original ID of the operation if any.
	ID string `json:"id"`

	// Description of the operation if any.
	Description string `json:"description"`

	// Additional comments for the operation, if any.
	Comments []string `json:"comments"`

	// HTTP method of the operation
	Method string `json:"method"`

	// Parameters of the operation, if any.
	Parameters []*Parameter `json:"parameters"`

	// Responses of the operation mapped to status codes.
	Responses []*Response `json:"responses"`

	// Callbacks of the operation
	Callbacks map[string][]*Path `json:"callbacks"`
}

// ParameterType describes where the parameter is expected.
type ParameterType string

const (
	// ParameterTypeQuery means the parameter is expected in the query string of the request.
	ParameterTypeQuery ParameterType = "query"

	// ParameterTypeBody means the parameter is expected in the body of the request.
	ParameterTypeBody ParameterType = "body"

	// ParameterTypePath means the parameter is expected in the path of the request.
	ParameterTypePath ParameterType = "path"

	// ParameterTypeHeader means the parameter is expected in a header of the request.
	ParameterTypeHeader ParameterType = "header"

	// ParameterTypeCookie means the parameter is expected in a cookie of the request.
	ParameterTypeCookie ParameterType = "cookie"
)

type ParameterSerializationStyle string

const (
	SerializationSimple         ParameterSerializationStyle = "simple"
	SerializationLabel          ParameterSerializationStyle = "label"
	SerializationMatrix         ParameterSerializationStyle = "matrix"
	SerializationForm           ParameterSerializationStyle = "form"
	SerializationSpaceDelimited ParameterSerializationStyle = "spaceDelimited"
	SerializationPipeDelimited  ParameterSerializationStyle = "pipeDelimited"
	SerializationDeepObject     ParameterSerializationStyle = "deepObject"
)

type ParameterSerialization struct {
	Style   ParameterSerializationStyle `json:"style"`
	Explode bool                        `json:"explode"`
}

// Parameter is a parameter for a HTTP operation.
type Parameter struct {
	// Name of the parameter.
	Name string `json:"name"`

	// Description of the parameter if any.
	Description string `json:"description"`

	// Type of the parameter.
	Type ParameterType `json:"type"`

	// Sometimes parameters are grouped in a struct type.
	// GroupType is the name of it.
	GroupType string `json:"groupType"`

	// The content type, if the parameter.
	// is expected in the body.
	ContentType string `json:"contentType"`

	// The schema of the parameter, if any.
	Schema *Schema `json:"schema"`

	Serialization ParameterSerialization `json:"serialization"`

	// Marks the parameter as required.
	Required bool `json:"required"`
}

func (p *Parameter) IsPtr() bool {
	return p.Schema != nil && (!p.Required || p.Schema.ShouldBePtr()) &&
		!p.Schema.CanBeNil()
}

// Response is one of the expected responses
// for a HTTP operation.
type Response struct {
	// Name of the response if any.
	Name string `json:"name"`

	// Description of the response if any.
	Description string `json:"description"`

	// HTTP status code of the response.
	// It is a sting because Open API schema permits codes like 50x.
	Code string `json:"code"`

	// The content type of the parameter.
	ContentType string `json:"contentType"`

	// The schema of the response, if any.
	Schema *Schema `json:"schema"`
}

func (r *Response) IsPtr() bool {
	return r.Schema != nil && r.Schema.ShouldBePtr() &&
		!r.Schema.CanBeNil()
}
