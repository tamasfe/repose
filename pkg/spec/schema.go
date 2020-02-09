package spec

import (
	"fmt"
)


// Schema is an abstraction over a specification schema.
type Schema struct {
	// Name is the given name of the schema.
	// It is also the type's name.
	Name string

	// The type in case of simple types.
	PrimitiveType string

	// Whether the type should be an alias
	// for an another one.
	Alias bool

	// OriginalName is the original name of
	// the schema from the specification.
	OriginalName string

	// FieldName is the name of the field
	// the schema is part of, if any.
	FieldName string

	// Description is the description of the original object
	// parsed from the specification.
	Description string

	// Additional comments for the schema, if any.
	Comments []string

	// Nullable defines whether the type can be nil.
	Nullable bool

	// Create indicates that the type must be created
	Create bool

	// Tags of the schema for json and such
	Tags map[string][]string

	// Variant is the variant of the schema.
	Variant SchemaVariant

	// Name of the additional properties field
	// if any.
	AdditionalPropsName string

	// Additional properties of the schema,
	// if it is a struct.
	AdditionalProps *Schema

	// Used for enum types
	Enum []interface{}

	// Children are needed in cases like when the
	// parent object is a struct, or a compound object.
	Children *SchemaObject
}

// SchemaVariant defines the variant of the schema.
// In most cases giving the schema a Go type is not enough,
// for example if a schema is an AllOf or even an object with properties.
type SchemaVariant string

const (
	// VariantPrimitive is a schema with no special attributes,
	// and has a simple Go type.
	VariantPrimitive SchemaVariant = "simple"

	// VariantAny is a schema where the type can be anything.
	// Typically it's an interface{}.
	VariantAny SchemaVariant = "any"

	// VariantArray is a schema where there can be
	// more than one of its children.
	VariantArray SchemaVariant = "array"

	// VariantMap is a schema where the first child
	// is the key type of the map, and the second is the value type
	VariantMap SchemaVariant = "map"

	// VariantStruct is a schema which
	// forms a struct of its children.
	VariantStruct SchemaVariant = "struct"

	// VariantAllOf defines a compound schema
	// of which all its children must be part of.
	VariantAllOf SchemaVariant = "allOf"

	// VariantAnyOf defines a compound schema where
	// one of its children might be present.
	VariantAnyOf SchemaVariant = "anyOf"

	// VariantOneOf defines a compound schema where
	// at least one of its children must be present.
	VariantOneOf SchemaVariant = "oneOf"
)

// SchemaObject is used in cases where multiple forms
// of schemas might be needed.
type SchemaObject struct {
	Schema *Schema
	Array  []*Schema
	Map    map[string]*Schema
}

// NewSchemaObject creates a new schema object of a compatible type
func NewSchemaObject(values interface{}) (*SchemaObject, error) {
	switch ob := values.(type) {
	case *SchemaObject:
		return ob, nil
	case *Schema:
		return &SchemaObject{Schema: ob}, nil
	case []*Schema:
		return &SchemaObject{Array: ob}, nil
	case map[string]*Schema:
		return &SchemaObject{Map: ob}, nil

	default:
		return nil, fmt.Errorf("incompatible type")
	}
}

// IsSchema checks whether the object is a single schema
func (s *SchemaObject) IsSchema() bool {
	return s.Schema != nil
}

// IsArray checks whether the object is an array of schemas
func (s *SchemaObject) IsArray() bool {
	return s.Array != nil
}

// IsMap checks whether the object is a map of schemas
func (s *SchemaObject) IsMap() bool {
	return s.Map != nil
}

// Is checks whether SchemaObject equals the given value
func (s *SchemaObject) Is(obj interface{}) bool {
	if s.Schema == obj {
		return true
	}

	if s.IsArray() {
		if arr, ok := obj.([]*Schema); ok {
			if len(arr) != len(s.Array) {
				return false
			}

			for i := range s.Array {
				if s.Array[i] != arr[i] {
					return false
				}
			}
			return true
		}
	}

	if s.IsMap() {
		if mp, ok := obj.(map[string]*Schema); ok {
			if len(mp) != len(s.Map) {
				return false
			}

			for k := range s.Map {
				if s.Map[k] != mp[k] {
					return false
				}
			}
			return true
		}
	}

	return false
}

// GetSchema is a helper method to avoid nil panics
func (s *SchemaObject) GetSchema() *Schema {
	if s == nil {
		return nil
	}
	return s.Schema
}

// GetMap is a helper method to avoid nil panics
func (s *SchemaObject) GetMap() map[string]*Schema {
	if s == nil {
		return nil
	}
	return s.Map
}

// GetArray is a helper method to avoid nil panics
func (s *SchemaObject) GetArray() []*Schema {
	if s == nil {
		return nil
	}
	return s.Array
}

// NewSchema creates a new empty schema
func NewSchema() *Schema {
	return &Schema{}
}

// WithName sets the name of the schema.
func (s *Schema) WithName(name string) *Schema {
	s.Name = name
	return s
}

// ShouldCreate is a convenience method for Create.
func (s *Schema) ShouldCreate(create bool) *Schema {
	s.Create = create
	return s
}

// WithType sets the type of the schema.
func (s *Schema) WithType(name string) *Schema {
	s.PrimitiveType = name
	return s
}

// SetNullable sets the schema to be nullable.
func (s *Schema) SetNullable() *Schema {
	s.Nullable = true
	return s
}

// SetVariant sets the variant of the type
func (s *Schema) SetVariant(variant SchemaVariant) *Schema {
	s.Variant = variant
	return s
}

// AddComments is a helper method to add comments
func (s *Schema) AddComments(comments ...string) *Schema {
	s.Comments = append(s.Comments, comments...)
	return s
}

// WithChildren sets the children of the schema.
// If the type of the children isn't compatible with a
// SchemaObject, it will panic.
func (s *Schema) WithChildren(children interface{}) *Schema {
	ob, err := NewSchemaObject(children)
	if err != nil {
		panic(err)
	}
	s.Children = ob
	return s
}

// Primitive is a convenience method for Primitive
func (s *Schema) Primitive(name string) *Schema {
	return s.SetVariant(VariantPrimitive).WithType(name)
}

// Any is a convenience method for Any variant
func (s *Schema) Any() *Schema {
	return s.SetVariant(VariantAny)
}

// AllOf is a convenience method for AllOf variant
func (s *Schema) AllOf(children interface{}) *Schema {
	return s.SetVariant(VariantAllOf).WithChildren(children)
}

// AnyOf is a convenience method for AnyOf variant
func (s *Schema) AnyOf(children interface{}) *Schema {
	return s.SetVariant(VariantAnyOf).WithChildren(children)
}

// OneOf is a convenience method for OneOf variant
func (s *Schema) OneOf(children interface{}) *Schema {
	return s.SetVariant(VariantOneOf).WithChildren(children)
}

// Map is a convenience method for setting a Map
func (s *Schema) Map(key *Schema, value *Schema) *Schema {
	s.Variant = VariantMap
	return s.WithChildren([]*Schema{key, value})
}

// Array is a convenience method for setting an Array
func (s *Schema) Array(child *Schema) *Schema {
	s.Variant = VariantArray
	return s.WithChildren(child)
}

// Struct is a convenience method for setting a Struct
func (s *Schema) Struct(child map[string]*Schema) *Schema {
	s.Variant = VariantStruct
	return s.WithChildren(child)
}

// HasChildren check whether the schema has children
func (s *Schema) HasChildren() bool {
	if s.Children == nil {
		return false
	}

	return s.Children.IsArray() || s.Children.IsMap() || s.Children.IsSchema()
}

// SchemaPath is a path of nested schemas.
// It is used while walking a schema.
type SchemaPath []*Schema

// First returns the first schema in the path
func (s SchemaPath) First() *Schema {
	if len(s) == 0 {
		return nil
	}
	return s[0]
}

// Last returns the last schema in the path
func (s SchemaPath) Last() *Schema {
	if len(s) == 0 {
		return nil
	}
	return s[len(s)-1]
}

// SchemaWalker is used when traversing a schema tree.
// A returned non-nil error will stop the traversal.
type SchemaWalker func(path SchemaPath) error

// Walk traverses the schema tree, calling the walker
// function for every schema in it.
func (s *Schema) Walk(walker SchemaWalker, bottomUp bool) error {
	if s == nil {
		return nil
	}

	paths := make([]SchemaPath, 0)

	s.walk(nil, func(path SchemaPath) error {
		paths = append(paths, path)
		return nil
	})

	if bottomUp {
		for i := len(paths) - 1; i >= 0; i-- {
			err := walker(paths[i])
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, p := range paths {
		err := walker(p)
		if err != nil {
			return err
		}
	}

	return nil
}

// ShouldBePtr is a helper method to determine
// if a schema type should be passed by value or by reference.
func (s *Schema) ShouldBePtr() bool {
	return s.Variant == VariantStruct ||
		s.Variant == VariantAllOf ||
		s.Variant == VariantAnyOf
}

// CanBeNil is a helper method to determine
// whether the type can be nil (E.g. maps).
func (s *Schema) CanBeNil() bool {
	return s.Variant == VariantMap ||
		s.Variant == VariantArray ||
		s.Variant == VariantOneOf ||
		s.Variant == VariantAnyOf ||
		s.Variant == VariantAny
}

func (s *Schema) walk(path []*Schema, walker SchemaWalker) {
	newPath := make([]*Schema, len(path), len(path)+1)
	copy(newPath, path)
	newPath = append(newPath, s)

	err := walker(newPath)
	if err != nil {
		return
	}

	if s.AdditionalProps != nil {
		s.AdditionalProps.walk(newPath, walker)
	}

	if s.Children == nil {
		return
	}

	switch {
	case s.Children.IsSchema():
		s.Children.Schema.walk(newPath, walker)

	case s.Children.IsArray():
		for _, c := range s.Children.Array {
			c.walk(newPath, walker)

		}

	case s.Children.IsMap():
		for _, c := range s.Children.Map {
			c.walk(newPath, walker)
		}

	}
}
