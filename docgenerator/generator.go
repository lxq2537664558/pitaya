package docgenerator

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/topfreegames/pitaya/component"
)

const (
	inputKey  = "input"
	outputKey = "output"
	typeKey   = "type"
)

// Docs returns a map from route to input and output
func Docs(handlers map[string]*component.Handler) map[string]interface{} {
	docs := map[string]interface{}{}

	for name, handler := range handlers {
		docs[name] = docForHandler(handler)
	}

	return docs
}

func docForHandler(handler *component.Handler) map[string]interface{} {
	input, output := map[string]interface{}{}, []interface{}{}

	if handler.Method.Type.NumIn() > 2 {
		isOutput := false
		in := handler.Method.Type.In(2)
		elm := in.Elem()
		for i := 0; i < elm.NumField(); i++ {
			if name, valid := getName(elm.Field(i), isOutput); valid {
				input[name] = parseType(elm.Field(i).Type, isOutput)
			}
		}
	}

	for i := 0; i < handler.Method.Type.NumOut(); i++ {
		isOutput := false
		out := handler.Method.Type.Out(i)
		if out.Kind() == reflect.Ptr {
			elm := out.Elem()
			fields := map[string]interface{}{}
			for j := 0; j < elm.NumField(); j++ {
				if name, valid := getName(elm.Field(j), isOutput); valid {
					fields[name] = parseType(elm.Field(j).Type, isOutput)
				}
			}

			output = append(output, fields)
		} else {
			output = append(output, out.String())
		}
	}

	return map[string]interface{}{
		inputKey:  input,
		outputKey: output,
		typeKey:   handler.Type.String(),
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
