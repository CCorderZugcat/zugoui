package observable

import (
	"reflect"
)

// MutableValue returns a value if CanSet, or is at least a non-nil map.
// Otherwise, if pointer or interface, dereferences if it can.
// For nil pointers (not interfaces, obviously), creates a new instance.
// If nothing can be done to return a settable value, returns zero value.
func MutableValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	// workable instances regardless of CanSet
	switch v.Kind() {
	case reflect.Map, reflect.Slice:
		if !v.IsNil() {
			return v
		}
	}

	if v.CanSet() && !(v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		return v
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			if v.Kind() == reflect.Interface {
				return reflect.Value{}
			}
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if !v.CanSet() {
		return reflect.Value{}
	}
	return v
}
