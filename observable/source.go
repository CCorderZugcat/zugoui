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

// writer is a proxy for a MutableSource which does not send its own updates
type writer struct {
	MutableSource
	o *Observe
}

func NewWriter(source MutableSource) MutableSource {
	w := writer{MutableSource: source, o: New()}
	w.MutableSource.AddObserver("", w.o)
	return w
}

// Release should be called when done with the writer
func (w writer) Release() {
	w.MutableSource.RemoveObserver("", w.o)
	w.o.Release()
}

func (w writer) Updating() (done func()) {
	return w.o.Updating()
}

func (w writer) AddObserver(key string, observer Observer) {
	w.o.AddObserver(key, observer)
}

func (w writer) RemoveObserver(key string, observer Observer) {
	w.o.RemoveObserver(key, observer)
}

// Mutable

func (w writer) SetValue(key string, value any) {
	done := w.Updating()
	defer done()
	w.MutableSource.SetValue(key, value)
}

func (w writer) InsertValueAt(index int, value any) {
	done := w.Updating()
	defer done()
	w.MutableSource.InsertValueAt(index, value)
}

func (w writer) RemoveValueAt(index int) {
	done := w.Updating()
	defer done()
	w.MutableSource.RemoveValueAt(index)
}

func (w writer) SetValueAt(index int, value any) {
	done := w.Updating()
	defer done()
	w.MutableSource.SetValueAt(index, value)
}

func (w writer) SetValueFor(key string, value any) {
	done := w.Updating()
	defer done()
	w.MutableSource.SetValueFor(key, value)
}

func (w writer) RemoveValueFor(key string) {
	done := w.Updating()
	defer done()
	w.MutableSource.RemoveValueFor(key)
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
