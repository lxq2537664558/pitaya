package docgenerator

import (
	"encoding/json"
	"reflect"
	"strings"
	"unicode"

	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/route"
)

const (
	inputKey   = "input"
	outputKey  = "output"
	typeKey    = "type"
	remoteCte  = "remote"
	handlerCte = "handler"
)

type docs struct {
	Handlers docMap `json:"handlers"`
	Remotes  docMap `json:"remotes"`
}

type docMap map[string]*doc

type doc struct {
	Input  interface{}   `json:"input"`
	Output []interface{} `json:"output"`
}

// HandlersDocs returns a map from route to input and output
func HandlersDocs(serverType string, services map[string]*component.Service) map[string]interface{} {
	docs := &docs{
		Handlers: map[string]*doc{},
	}

	for serviceName, service := range services {
		for name, handler := range service.Handlers {
			routeName := route.NewRoute(serverType, serviceName, name)
			docs.Handlers[routeName.String()] = docForMethod(handler.Method)
		}
	}

	return docs.Handlers.toMap()
}

// RemotesDocs returns a map from route to input and output
func RemotesDocs(serverType string, services map[string]*component.Service) map[string]interface{} {
	docs := &docs{
		Remotes: map[string]*doc{},
	}

	for serviceName, service := range services {
		for name, remote := range service.Remotes {
			routeName := route.NewRoute(serverType, serviceName, name)
			docs.Remotes[routeName.String()] = docForMethod(remote.Method)
		}
	}

	return docs.Remotes.toMap()
}

func (d docMap) toMap() map[string]interface{} {
	var m map[string]interface{}
	bts, _ := json.Marshal(d)
	json.Unmarshal(bts, &m)
	return m
}

func docForMethod(method reflect.Method) *doc {
	doc := &doc{
		Output: []interface{}{},
	}

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
			doc.Input = fields
		} else {
			doc.Input = parseType(in, isOutput)
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

			doc.Output = append(doc.Output, fields)
		} else {
			doc.Output = append(doc.Output, out.String())
		}
	}

	return doc
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
