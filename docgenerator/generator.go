package docgenerator

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/topfreegames/pitaya/component"
)

const (
	inputKey   = "input"
	outputKey  = "output"
	serviceKey = "server"
	typeKey    = "type"
	remoteCte  = "remote"
	handlerCte = "handler"
)

// Docs returns a map from route to input and output
func Docs(services map[string]*component.Service) map[string]interface{} {
	docs := map[string]interface{}{}

	for serviceName, service := range services {
		for name, handler := range service.Handlers {
			docs[name] = docForHandler(serviceName, handlerCte, handler.Method)
		}

		for name, remote := range service.Remotes {
			docs[name] = docForHandler(serviceName, remoteCte, remote.Method)
		}
	}

	return docs
}

func docForHandler(serviceName, component string, method reflect.Method) map[string]interface{} {
	var input interface{}
	output := []interface{}{}

	if method.Type.NumIn() > 2 {
		isOutput := false
		in := method.Type.In(2)
		if in.Kind() == reflect.Ptr {
			fields := map[string]interface{}{}
			elm := in.Elem()
			for i := 0; i < elm.NumField(); i++ {
				if name, valid := getName(elm.Field(i), isOutput); valid {
					fields[name] = parseType(elm.Field(i).Type, isOutput)
				}
			}
			input = fields
		} else {
			input = parseType(in, isOutput)
		}
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

			output = append(output, fields)
		} else {
			output = append(output, out.String())
		}
	}

	return map[string]interface{}{
		inputKey:   input,
		outputKey:  output,
		serviceKey: serviceName,
		typeKey:    component,
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
		parsed := parseType(typ.Elem(), isOutput)
		if parsed == "uint8" {
			return "[]byte"
		}
		return []interface{}{parsed}
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
