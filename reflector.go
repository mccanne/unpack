package unpack

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/kr/pretty"
)

var (
	ErrNoKey = errors.New("encountered object with no key field")
)

type Reflector map[string]reflect.Type

func New() Reflector {
	return make(map[string]reflect.Type)
}

func (r Reflector) Init(templates ...interface{}) Reflector {
	for _, t := range templates {
		r.Add(t)
	}
	return r
}

func (r Reflector) Add(template interface{}) Reflector {
	val := reflect.ValueOf(template)
	if val.Kind() != reflect.Struct {
		panic("cannot unpack into a non-struct type")
	}
	typ := val.Type()
	name := val.Type().Name()
	if _, ok := r[name]; ok {
		panic(fmt.Sprintf("reflector template for '%s' already exists", name))
	}
	r[name] = typ
	return r
}

func (r *Reflector) UnpackSkeleton(key, src string) (interface{}, error) {
	var jsonVal interface{}
	if err := json.Unmarshal([]byte(src), &jsonVal); err != nil {
		return nil, fmt.Errorf("unpacker error parsing JSON: %w", err)
	}
	object, ok := jsonVal.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot unpack non-object JSON value")
	}
	return r.unpackStruct(key, object)
}

func (r *Reflector) Unpack(key, src string) (interface{}, error) {
	skeleton, err := r.UnpackSkeleton(key, src)
	if err != nil {
		return nil, err
	}
	pretty.Println(skeleton)
	if err := json.Unmarshal([]byte(src), &skeleton); err != nil {
		pretty.Println(skeleton)
		return nil, fmt.Errorf("unpacker error parsing JSON into skeleton: %w", err)
	}
	return skeleton, nil
}

var zero reflect.Value

func (r Reflector) unpackStruct(key string, object map[string]interface{}) (interface{}, error) {
	which, ok := object[key]
	if !ok {
		// XXX we need to recurse here and keep looking for interface
		// values by building a generic map[string]interface{} using
		// reflect at this level and recursively setting a value for each
		// key...
		return r.unpackGenericStruct(key, object)
	}
	templateKey, ok := which.(string)
	if !ok {
		// XXX return type name of struct above if recursive...?
		return nil, fmt.Errorf("field '%s' not found in object", key)
	}
	typ, ok := r[templateKey]
	if !ok {
		return nil, fmt.Errorf("no template for reflector key '%s: \"%s\"'", key, templateKey)
	}
	if typ.Kind() != reflect.Struct {
		panic("unpack internal error: non-structs should not be allowed as a template")
	}
	ptr := reflect.New(typ)
	val := ptr.Elem()
	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		fmt.Println("TYPE", i, val.Field(i).Kind())
		switch val.Field(i).Kind() {
		case reflect.Interface:
			// Get the object field name from the json tag
			// in the template struct.
			name := fieldName(fieldType)
			// Now get the sub-object out of the json map.
			subField, ok := object[name]
			if !ok {
				return zero, fmt.Errorf("expected field '%s' is missing in object with '%s: \"%s\"'", name, key, templateKey)
			}
			subMap, ok := subField.(map[string]interface{})
			if !ok {
				return zero, fmt.Errorf("field '%s' in object with '%s: \"%s\"' is not unpackable object", name, key, templateKey)
			}
			subVal, err := r.unpackStruct(key, subMap)
			if err != nil {
				return zero, err
			}
			val.Field(i).Set(reflect.ValueOf(subVal))
		case reflect.Array:
			return zero, errors.New("unpacker: arrays not supported")
		case reflect.Slice:
			panic("slice")
		}
	}
	return ptr.Interface(), nil
}

func (r Reflector) unpackGenericStruct(keyField string, object map[string]interface{}) (interface{}, error) {
	//generic := reflect.ValueOf(object)
	//ptr := reflect.New(generic.Type())
	//val := ptr.Elem()
	//iter := reflect.ValueOf(m).MapRange()
	//for _, key := range generic.MapKeys() {
	//}

	out := make(map[string]interface{})
	for key, val := range object {
		v := reflect.ValueOf(val)
		if v.Kind() == reflect.Struct {
			subObj, ok := val.(map[string]interface{})
			if !ok {
				return zero, errors.New("problem with JSON-parsed object map")
			}
			pretty.Println("SUBOBJ", subObj)
			field, err := r.unpackStruct(keyField, subObj)
			if err != nil {
				return zero, err
			}
			out[key] = field
		}
	}
	return out, nil
}

const (
	tagName = "json"
	tagSep  = ","
)

func fieldName(f reflect.StructField) string {
	tag := f.Tag.Get(tagName)
	if tag != "" {
		s := strings.SplitN(tag, tagSep, 2)
		if len(s) > 0 && s[0] != "" {
			return s[0]
		}
	}
	return f.Name
}
