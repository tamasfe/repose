package markdown

import (
	"os/exec"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func OptionsTable(opts interface{}) string {
	var entriesBuilder strings.Builder

	optsTp := reflect.TypeOf(opts)
	optsVal := reflect.ValueOf(opts)

	entriesBuilder.WriteString(`
| Option | Description | Type | Default Value |
|:------:|-------------|:----:|:--------------|	
`[1:])

	fieldNames := make(map[string]int, optsTp.NumField())
	fields := make([]string, 0, optsTp.NumField())
	for i := 0; i < optsTp.NumField(); i++ {
		field := optsTp.Field(i)
		fieldNames[field.Name] = i
		fields = append(fields, field.Name)
	}

	sort.Strings(fields)

	for _, f := range fields {
		field := optsTp.Field(fieldNames[f])
		val := optsVal.Field(fieldNames[f]).Interface()

		valB, err := yaml.Marshal(val)
		if err != nil {
			panic(err)
		}

		_, err = entriesBuilder.WriteString(
			strings.Join(
				[]string{
					strings.Split(field.Tag.Get("yaml"), ",")[0],
					field.Tag.Get("description") + ".",
					field.Type.String(),
					strings.Replace("<pre lang=\"yaml\">"+string(valB[:len(valB)-1])+"</pre>", "\n", "<br>", -1),
				},
				"|",
			) + "|\n",
		)

		if err != nil {
			panic(err)
		}
	}

	return entriesBuilder.String()
}

func ExtensionsTable(ext interface{}) string {
	var entriesBuilder strings.Builder

	optsTp := reflect.TypeOf(ext)

	entriesBuilder.WriteString(`
| Field | Description | Type |
|:-----:|-------------|:----:|
`[1:])

	fieldNames := make(map[string]int, optsTp.NumField())
	fields := make([]string, 0, optsTp.NumField())
	for i := 0; i < optsTp.NumField(); i++ {
		field := optsTp.Field(i)
		fieldNames[field.Name] = i
		fields = append(fields, field.Name)
	}

	sort.Strings(fields)

	for _, f := range fields {
		field := optsTp.Field(fieldNames[f])

		_, err := entriesBuilder.WriteString(
			strings.Join(
				[]string{
					strings.Split(field.Tag.Get("yaml"), ",")[0],
					field.Tag.Get("description") + ".",
					field.Type.String(),
				},
				"|",
			) + "|\n",
		)
		if err != nil {
			panic(err)
		}
	}

	return entriesBuilder.String()
}

func TagsTable(ext interface{}) string {
	var entriesBuilder strings.Builder

	optsTp := reflect.TypeOf(ext)

	entriesBuilder.WriteString(`
| Value | Description |
|:-----:|-------------|
`[1:])

	fieldNames := make(map[string]int, optsTp.NumField())
	fields := make([]string, 0, optsTp.NumField())
	for i := 0; i < optsTp.NumField(); i++ {
		field := optsTp.Field(i)
		fieldNames[field.Name] = i
		fields = append(fields, field.Name)
	}

	sort.Strings(fields)

	for _, f := range fields {
		field := optsTp.Field(fieldNames[f])

		_, err := entriesBuilder.WriteString(
			strings.Join(
				[]string{
					field.Name,
					field.Tag.Get("description"),
				},
				"|",
			) + "|\n",
		)
		if err != nil {
			panic(err)
		}
	}

	return entriesBuilder.String()
}

func TargetsTable(targets map[string]string) string {
	var entriesBuilder strings.Builder

	entriesBuilder.WriteString(`
| Target | Description |
|:------:|-------------|
`[1:])

	keys := make([]string, 0, len(targets))
	for k := range targets {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		entriesBuilder.WriteString(
			strings.Join(
				[]string{
					k,
					targets[k],
				},
				"|") + "|\n",
		)
	}

	return entriesBuilder.String()
}

func GenTOC(header, md string) string {
	cmd := exec.Command("/bin/sh", "./internal/gh-md-toc", "-")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	_, err = stdin.Write([]byte(md))
	if err != nil {
		panic(err)
	}
	stdin.Close()

	toc, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	return header + string(toc) + "\n" + md
}
