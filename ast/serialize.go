package ast

import (
	"bytes"
	"encoding/json"
	"github.com/emilioastarita/gphp/lexer"
	goast "go/ast"
	"reflect"
	"unicode"
)

type serializer struct {
	ptrmap          map[interface{}]bool // *T -> line number
	ignoredFields   []string
	typeOfToken     reflect.Type
	typeOfTokenNode reflect.Type
	tagName         string
}

func Serialize(x interface{}) interface{} {
	s := serializer{
		tagName:         "serialize",
		ptrmap:          make(map[interface{}]bool),
		typeOfToken:     reflect.TypeOf(lexer.Token{}),
		typeOfTokenNode: reflect.TypeOf(TokenNode{}),
	}
	return s.serialize(reflect.ValueOf(x), false)
}

func (s *serializer) formatSubField(x reflect.StructField) string {
	if tag := x.Tag.Get(s.tagName); tag != "" && tag[0] != '-' {
		return tag
	}
	r := []rune(x.Name)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func (s *serializer) isIgnoredField(x reflect.StructField) bool {
	if tag := x.Tag.Get(s.tagName); tag == "-" {
		return true
	}
	return false
}

func (s *serializer) isEmbedded(x reflect.StructField) bool {
	if tag := x.Tag.Get(s.tagName); tag == "-flat" {
		return true
	}
	return false
}
func (s *serializer) isSingleChildren(x reflect.StructField) bool {
	if tag := x.Tag.Get(s.tagName); tag == "-single" {
		return true
	}
	return false
}

func (s *serializer) serialize(x reflect.Value, singleChildren bool) interface{} {

	if !goast.NotNilFilter("", x) {
		return nil
	}

	typeName := x.Type().Name()

	switch x.Kind() {
	case reflect.Interface:
		return s.serialize(x.Elem(), false)
	case reflect.Map:

		me := make(map[string]map[string]interface{})
		me[typeName] = make(map[string]interface{})
		if x.Len() > 0 {
			for _, key := range x.MapKeys() {
				me[typeName][key.String()] = s.serialize(x.MapIndex(key), false)
			}
		}
		return me

	case reflect.Ptr:
		// type-checked ASTs may contain cycles - use ptrmap
		// to keep track of objects that have been printed
		// already and print the respective line number instead
		ptr := x.Interface()
		if _, exists := s.ptrmap[ptr]; exists {
			return nil
		} else {
			s.ptrmap[ptr] = exists
			return s.serialize(x.Elem(), false)
		}

	case reflect.Array:
		me := make([]interface{}, x.Len())

		// some children nodes uses only one object
		// instead an array
		if x.Len() == 1 && singleChildren {
			return s.serialize(x.Index(0), false)
		}

		if x.Len() > 0 {
			for i, n := 0, x.Len(); i < n; i++ {
				me[i] = s.serialize(x.Index(i), false)
			}
		}
		return me

	case reflect.Slice:
		if _, ok := x.Interface().([]byte); ok {
			return nil
		}

		// some children nodes uses only one object
		// instead an array
		if x.Len() == 1 && singleChildren {
			return s.serialize(x.Index(0), false)
		}

		me := make([]interface{}, x.Len())
		if x.Len() > 0 {
			for i, n := 0, x.Len(); i < n; i++ {
				me[i] = s.serialize(x.Index(i), false)
			}
		}
		return me

	case reflect.Struct:
		t := x.Type()

		switch t {
		case s.typeOfToken:
			me := make(map[string]interface{})
			me["kind"] = s.serialize(x.FieldByName("Kind"), false)
			me["fullStart"] = s.serialize(x.FieldByName("FullStart"), false)
			me["start"] = s.serialize(x.FieldByName("Start"), false)
			me["length"] = s.serialize(x.FieldByName("Length"), false)
			return me
		case s.typeOfTokenNode:
			return s.serialize(x.FieldByName("Token"), false)
		default:
			me := make(map[string]map[string]interface{})
			me[typeName] = make(map[string]interface{})
			for i, n := 0, t.NumField(); i < n; i++ {
				// exclude non-exported fields because their
				// values cannot be accessed via reflection
				if field := t.Field(i); goast.IsExported(field.Name) && !s.isIgnoredField(field) {
					value := x.Field(i)
					name := s.formatSubField(field)

					if s.isEmbedded(field) {
						embedded := s.serialize(value, false)
						m, ok := embedded.(map[string]map[string]interface{})
						if ok {
							for k, v := range m[field.Name] {
								me[typeName][k] = v
							}
						}
					} else {
						me[typeName][name] = s.serialize(value, s.isSingleChildren(field))
					}

				}
			}
			return me
		}
	default:
		v := x.Interface()
		switch v := v.(type) {
		case lexer.TokenKind:
			return v.String()
		}
		return v
	}

	return nil
}

func PrettyPrintJSON(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "    ")
	return out.Bytes(), err
}
