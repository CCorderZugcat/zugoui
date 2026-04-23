package observable

import (
	"reflect"
)

// Source is a read only, observable source of data
type Source interface {
	Observable
	Keys() []string                    // returns all known keys
	Tag(key, tag string) []string      // if applicable, returns field tags for key applicatable to tag
	ModelFor(key string) reflect.Value // returns an introspection value for the key
	Model() reflect.Value              // returns an introspection value backing the whole source, if it has one
	Elem() reflect.Type                // for map, slice, and array, the element (or map value) type
	Value(key string) any              // Value returns a value for key. For simple types, the key is "value".
	ValueAt(index int) any             // ValueAt returns slice or array's value at index
	ValueFor(key string) any           // ValueFor returns a map's key value
}

// MutableSource is a mutable, obervable source of data
type MutableSource interface {
	Observer // mutation and observation methods are shared
	Source
}

// NullSource is a convenience for minimal Source objects
type NullSource struct{}

func (n NullSource) Value(string) any              { return nil }
func (n NullSource) ValueFor(string) any           { return nil }
func (n NullSource) ValueAt(int) any               { return nil }
func (n NullSource) Keys() []string                { return []string{"value"} }
func (n NullSource) Tag(string, string) []string   { return nil }
func (n NullSource) ModelFor(string) reflect.Value { return reflect.Value{} }
func (n NullSource) Elem() reflect.Type            { return nil }
func (n NullSource) Model() reflect.Value          { return reflect.Value{} }
