package util

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/mitchellh/go-wordwrap"

	"gopkg.in/yaml.v3"
)

// ParamStyleToColon converts the params in the string from style {param} to :param
func ParamStyleToColon(path string) string {
	re := regexp.MustCompile(`{([^}]+)}`)
	return re.ReplaceAllString(path, `:$1`)
}

// ParamStyleToBraces converts the params in the string from style :param to {param}
func ParamStyleToBraces(path string) string {
	re := regexp.MustCompile(`:([A-Za-z0-9]+)`)
	return re.ReplaceAllString(path, `{$1}`)
}

// DisableYAMLMarshalComments controls MarshalYAMLWithDescriptions
var DisableYAMLMarshalComments = false

// MarshalYAMLWithDescriptions provides marshaling structs with
// Repose descriptions.
//
// Make sure the value (pointer receiver is fine)
// you pass in doesn't implement YAML Marshaler,
// otherwise YAML will get into a Marshal() loop.
func MarshalYAMLWithDescriptions(val interface{}) (interface{}, error) {
	tp := reflect.TypeOf(val)
	if tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}

	if tp.Kind() != reflect.Struct {
		return nil, fmt.Errorf("only structs are supported")
	}

	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	b, err := yaml.Marshal(v.Interface())
	if err != nil {
		return nil, err
	}

	var node yaml.Node
	err = yaml.Unmarshal(b, &node)
	if err != nil {
		return nil, err
	}

	if !DisableYAMLMarshalComments {
		for _, n := range node.Content[0].Content {
			for i := 0; i < v.NumField(); i++ {
				fieldType := tp.Field(i)
				name := strings.Split(fieldType.Tag.Get("yaml"), ",")[0]

				if name == "" {
					name = fieldType.Name
				}

				if n.Value == name {
					desc := fieldType.Tag.Get("description")
					n.HeadComment = wordwrap.WrapString(desc, 80) + "."
					break
				}

			}
		}
	}

	node.Kind = yaml.MappingNode
	node.Content = node.Content[0].Content

	return &node, nil
}

func MustMarshalYAML(i interface{}) []byte {
	b, err := yaml.Marshal(i)
	if err != nil {
		panic(err)
	}
	return b
}
