package jstypes

import (
	"encoding"
	"encoding/gob"
	"reflect"
)

var (
	stringType = reflect.TypeFor[string]()
	anyType    = reflect.TypeFor[any]()
)

func init() {
	gob.Register(map[string]any{})
	gob.Register([]any{})
}

// ValueOf returns a js friendly instance of in.
// js.ValueOf(out) will work smoothly.
// If this is impossible, returns nil, false.
// seq.Iter types return slices, seq.Iter2 types return maps.
// If it is easly determined the type is compatible, returns the same instance.
// (Keep this in mind if mutating)
func ValueOf(in any) (out any, ok bool) {
	if tm, ok := in.(encoding.TextMarshaler); ok {
		s, err := tm.MarshalText()
		if err != nil {
			return out, false
		}
		return string(s), true
	}

	v, ok := valueOf(reflect.ValueOf(in))
	if !ok || !v.IsValid() {
		return nil, false
	}
	return v.Interface(), true
}

func valueOf(in reflect.Value) (out reflect.Value, ok bool) {
	if !in.IsValid() {
		return out, true
	}

	for in.Kind() == reflect.Pointer || in.Kind() == reflect.Interface {
		if in.IsNil() {
			return out, true
		}
		in = in.Elem()
	}

	switch in.Kind() {
	case reflect.String:
		return stringy(in), true
	case reflect.Bool:
		return scaler(in, reflect.TypeFor[bool]()), true
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return in, true
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		return in, true
	case reflect.Float32, reflect.Float64:
		return in, true
	case reflect.Map:
		return makeMap(in)
	case reflect.Array, reflect.Slice:
		return makeArray(in)
	case reflect.Struct:
		return makeObject(in)
	}

	if in.Type().CanSeq() {
		return makeIterSlice(in)
	}
	if in.Type().CanSeq2() {
		return makeIterMap(in)
	}

	return out, false
}

func makeIterSlice(in reflect.Value) (out reflect.Value, ok bool) {
	out = reflect.MakeSlice(reflect.SliceOf(anyType), 0, 0)
	for value := range in.Seq() {
		out = reflect.Append(out, value)
	}
	return out, true
}

func makeIterMap(in reflect.Value) (out reflect.Value, ok bool) {
	out = reflect.MakeMap(reflect.MapOf(stringType, anyType))
	for key, value := range in.Seq2() {
		if key.Kind() != reflect.String {
			return reflect.Value{}, false
		}
		out.SetMapIndex(scaler(key, stringType), value)
	}
	return out, true
}

// convert a struct into a map of its values
func makeObject(in reflect.Value) (out reflect.Value, ok bool) {
	out = reflect.MakeMap(reflect.MapOf(stringType, anyType))

	for i := range in.NumField() {
		ft := in.Type().Field(i)
		if !ft.IsExported() {
			continue
		}

		v, ok := valueOf(in.Field(i))
		if !ok {
			continue // inconvertible
		}
		if !v.IsValid() {
			v = reflect.Zero(anyType)
		}
		out.SetMapIndex(reflect.ValueOf(ft.Name), v)
	}

	return out, true
}

// convert to map[string]any
func makeMap(in reflect.Value) (out reflect.Value, ok bool) {
	if in.IsNil() {
		return out, true
	}
	out = reflect.MakeMap(reflect.MapOf(stringType, anyType))
	keys := in.MapRange()
	for keys.Next() {
		v, ok := valueOf(keys.Value())
		if !ok {
			continue // skip inconvertible entries
		}
		if !v.IsValid() {
			v = reflect.Zero(anyType)
		}
		out.SetMapIndex(stringy(keys.Key()), v)
	}
	return out, true
}

// convert to []any
func makeArray(in reflect.Value) (out reflect.Value, ok bool) {
	if in.Kind() == reflect.Slice && in.IsNil() {
		return out, true
	}
	out = reflect.MakeSlice(reflect.SliceOf(anyType), 0, in.Len())
	for i := range in.Len() {
		v, ok := valueOf(in.Index(i))
		if !ok {
			continue // skip inconvertible entries
		}
		if !v.IsValid() {
			v = reflect.Zero(anyType)
		}
		out = reflect.Append(out, v)
	}
	return out, true
}

// convert to string, or return same if already
func stringy(in reflect.Value) reflect.Value {
	return scaler(in, stringType)
}

// scaler to type or return same if already, presumes in and typ are same kind
func scaler(in reflect.Value, typ reflect.Type) reflect.Value {
	if in.Type() == typ {
		return in
	}
	return in.Convert(typ)
}
