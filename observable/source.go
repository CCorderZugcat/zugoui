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

// RemoveAllObservers should be called when done with the writer
func (w writer) RemoveAllObservers() {
	w.MutableSource.RemoveObserver("", w.o)
	w.o.RemoveAllObservers()
}

func (w writer) Updating(key string) (done func()) {
	return w.o.Updating(key)
}

func (w writer) AddObserver(key string, observer Observer) {
	w.o.AddObserver(key, observer)
}

func (w writer) RemoveObserver(key string, observer Observer) {
	w.o.RemoveObserver(key, observer)
}

// Mutable

func (w writer) SetValue(key string, value any) {
	done := w.Updating(key)
	defer done()
	w.MutableSource.SetValue(key, value)
}

func (w writer) InsertValueAt(index int, value any) {
	done := w.Updating("value")
	defer done()
	w.MutableSource.InsertValueAt(index, value)
}

func (w writer) RemoveValueAt(index int) {
	done := w.Updating("value")
	defer done()
	w.MutableSource.RemoveValueAt(index)
}

func (w writer) SetValueAt(index int, value any) {
	done := w.Updating("value")
	defer done()
	w.MutableSource.SetValueAt(index, value)
}

func (w writer) SetValueFor(key string, value any) {
	done := w.Updating(key)
	defer done()
	w.MutableSource.SetValueFor(key, value)
}

func (w writer) RemoveValueFor(key string) {
	done := w.Updating(key)
	defer done()
	w.MutableSource.RemoveValueFor(key)
}
