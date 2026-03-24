//go:build js

package jsglue

import (
	"encoding"
	"reflect"
	"strconv"
	"syscall/js"

	"github.com/CCorderZugcat/zugoui/observable"
)

var object = js.Global().Get("Object")

// Set "unmarshals" a js.Value into obj
func Set(obj any, data js.Value) bool {
	v := reflect.ValueOf(obj)

	return SetValue(v, data)
}

func SetValue(v reflect.Value, data js.Value) bool {
	if !v.IsValid() {
		return false
	}
	if data.Type() == js.TypeString && v.Type().Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) {
		if err := v.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(data.String())); err == nil {
			return true
		}
	}

	if data.IsNull() || data.IsUndefined() {
		v.SetZero()
		return true
	}
	v = observable.MutableValue(v)
	if !v.IsValid() {
		return false
	}

	switch v.Kind() {
	case reflect.String:
		return setScaler(v, data.String())
	case reflect.Bool:
		return setScaler(v, data.Truthy())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if data.Type() != js.TypeNumber {
			i, err := strconv.ParseInt(data.String(), 10, 64)
			if err != nil {
				return false
			}
			return setScaler(v, i)
		}
		return setScaler(v, data.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if data.Type() != js.TypeNumber {
			i, err := strconv.ParseUint(data.String(), 10, 64)
			if err != nil {
				return false
			}
			return setScaler(v, i)
		}
		return setScaler(v, data.Int())
	case reflect.Float32, reflect.Float64:
		if data.Type() != js.TypeNumber {
			f, err := strconv.ParseFloat(data.String(), 64)
			if err != nil {
				return false
			}
			return setScaler(v, f)
		}
		return setScaler(v, data.Float())
	case reflect.Slice:
		if data.Type() != js.TypeObject {
			return false
		}
		return setSlice(v, data)
	case reflect.Array:
		if data.Type() != js.TypeObject {
			return false
		}
		return setArray(v, data)
	case reflect.Map:
		if data.Type() != js.TypeObject {
			return false
		}
		return setMap(v, data)
	case reflect.Struct:
		if data.Type() != js.TypeObject {
			return false
		}
		return setStruct(v, data)
	}

	return false
}

func setStruct(v reflect.Value, data js.Value) bool {
	for i := range v.Type().NumField() {
		ft := v.Type().Field(i)
		if !ft.IsExported() {
			continue
		}
		fv := v.Field(i)
		if !SetValue(fv, data.Get(ft.Name)) {
			fv.SetZero()
		}
	}

	return true
}

func setMap(v reflect.Value, data js.Value) bool {
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}

	for entry := range Iter(object.Call("entries", data)) {
		if entry.Type() != js.TypeObject || entry.Length() < 2 {
			continue
		}
		key, value := entry.Index(0), entry.Index(1)
		mapKey := reflect.New(v.Type().Key()).Elem()
		if !setScaler(mapKey, key.String()) {
			continue
		}
		mapValue := reflect.New(v.Type().Elem()).Elem()
		if !SetValue(mapValue, value) {
			continue
		}
		v.SetMapIndex(mapKey, mapValue)
	}

	return true
}

func setSlice(v reflect.Value, data js.Value) bool {
	v.SetLen(0)

	for elem := range Iter(data) {
		ev := reflect.New(v.Type().Elem()).Elem()
		SetValue(ev, elem) // leave at zero value if returned false
		v.Set(reflect.Append(v, ev))
	}

	return true
}

func setArray(v reflect.Value, data js.Value) bool {
	index := 0
	for elem := range Iter(data) {
		if index >= v.Len() {
			return false
		}
		vi := v.Index(index)
		if !SetValue(vi, elem) {
			vi.SetZero()
		}

		index++
	}

	return true
}

func setScaler(v reflect.Value, x any) bool {
	data := reflect.ValueOf(x)
	if v.Type() == data.Type() {
		v.Set(data)
	} else {
		if !data.CanConvert(v.Type()) {
			return false
		}
		v.Set(data.Convert(v.Type()))
	}

	return true
}
