package docgenerator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"reflect"
	"strings"
	"unicode"
)

// HTMLDocForHandlers ...
func HTMLDocForHandlers(handlers map[string]reflect.Method) (string, error) {
	docs, err := docsForHandlers(handlers)
	if err != nil {
		return "", err
	}

	const tpl = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
	</head>
	<body>
		{{range $key, $value := .}}
			<div>
			<b>{{ $key }}</b>
			{{ $value }}
			</div>
		{{end}}
	</body>
</html>`

	t, err := template.New("dochtml").Parse(tpl)
	if err != nil {
		return "", nil
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	err = t.ExecuteTemplate(writer, "dochtml", docs)
	if err != nil {
		return "", nil
	}

	return b.String(), nil
}

// Doc ...
type Doc struct {
	Input  map[string]interface{}
	Output map[string]interface{}
}

func (d *Doc) pretty() (string, error) {
	bts, err := json.MarshalIndent(d, "", "	")
	if err != nil {
		return "", err
	}

	return string(bts), nil
}

func docsForHandlers(handlers map[string]reflect.Method) (map[string]string, error) {
	var err error
	docs := map[string]string{}

	for name, method := range handlers {
		docs[name], err = docForHandler(method).pretty()
		if err != nil {
			return nil, err
		}
	}

	return docs, nil
}

func docForHandler(method reflect.Method) *Doc {
	input, output := map[string]interface{}{}, map[string]interface{}{}

	if method.Type.NumIn() > 2 {
		isOutput := false
		in := method.Type.In(2)
		elm := in.Elem()
		fields := map[string]interface{}{}
		for i := 0; i < elm.NumField(); i++ {
			if name, valid := getName(elm.Field(i), isOutput); valid {
				fields[name] = parseType(elm.Field(i).Type, isOutput)
			}
		}

		input[in.String()] = fields
	}

	for i := 0; i < method.Type.NumOut(); i++ {
		isOutput := false
		out := method.Type.Out(i)
		if out.Kind() == reflect.Ptr {
			elm := out.Elem()
			fields := map[string]interface{}{}
			for j := 0; j < elm.NumField(); j++ {
				if name, valid := getName(elm.Field(j), isOutput); valid {
					fields[name] = parseType(elm.Field(j).Type, isOutput)
				}
			}

			output[out.String()] = fields
		} else {
			output[out.String()] = out.String()
		}
	}

	return &Doc{
		Input:  input,
		Output: output,
	}
}

func parseStruct(typ reflect.Type) reflect.Type {
	switch typ.String() {
	case "time.Time":
		return nil
	default:
		return typ
	}
}

func validName(field reflect.StructField) bool {
	isProtoField := func(name string) bool {
		return strings.HasPrefix(name, "XXX_")
	}

	isPrivateField := func(name string) bool {
		for _, r := range name {
			return unicode.IsLower(r)
		}

		return true
	}

	isIgnored := func(field reflect.StructField) bool {
		return field.Tag.Get("json") == "-"
	}

	return !isProtoField(field.Name) && !isPrivateField(field.Name) && !isIgnored(field)
}

func firstLetterToLower(name string, isOutput bool) string {
	if isOutput {
		return name
	}

	return string(append([]byte{strings.ToLower(name)[0]}, name[1:len(name)]...))
}

func getName(field reflect.StructField, isOutput bool) (name string, valid bool) {
	if !validName(field) {
		return "", false
	}

	name, ok := field.Tag.Lookup("json")
	if !ok {
		return firstLetterToLower(field.Name, isOutput), true
	}

	return strings.Split(name, ",")[0], true
}

func parseType(typ reflect.Type, isOutput bool) interface{} {
	var elm reflect.Type

	switch typ.Kind() {
	case reflect.Ptr:
		elm = typ.Elem()
	case reflect.Struct:
		elm = parseStruct(typ)
		if elm == nil {
			return typ.String()
		}
	case reflect.Slice:
		return []interface{}{parseType(typ.Elem(), isOutput)}
	default:
		return typ.String()
	}

	fields := map[string]interface{}{}
	for i := 0; i < elm.NumField(); i++ {
		if name, valid := getName(elm.Field(i), isOutput); valid {
			fields[name] = parseType(elm.Field(i).Type, isOutput)
		}
	}
	return fields
}
