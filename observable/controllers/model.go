package controllers

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/CCorderZugcat/zugoui/observable"
)

var controllers = make(map[string]func(observable.Source, []string) observable.Source)

// RegisterController is called during initialization from a package providing a controller
func RegisterController(name string, ctor func(observable.Source, []string) observable.Source) {
	controllers[name] = ctor
}

// Model allows runtime setting, getting, and observing of an arbitrary model.
// This is the default and root level controller for any object.
type Model struct {
	model   reflect.Value
	elem    reflect.Type
	keys    sync.Map
	sources sync.Map
	*observable.Observe
}

var _ observable.MutableSource = &Model{}

// New creates a new observeable Model instance.
func New(model any) *Model {
	if m, ok := model.(*Model); ok {
		return m
	}
	return NewValue(reflect.ValueOf(model))
}

// NewValue is like NewModel but with a Value
// v should be a  pointer to see results of mutations,
// but if it is not, then an internal copy is used.
func NewValue(v reflect.Value) *Model {
	o := &Model{
		Observe: observable.New(),
	}

	v = observable.MutableValue(v)
	if !v.IsValid() {
		return nil
	}

	o.model = v

	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		o.elem = v.Type().Elem()
	}

	return o
}

func (m *Model) key(name string) (value reflect.Value) {
	if v, ok := m.keys.Load(name); ok {
		return v.(reflect.Value)
	}

	cache := true
	defer func() {
		if cache && value.IsValid() {
			if v, ok := m.keys.LoadOrStore(name, value); ok {
				// let the earlier store win the race
				value = v.(reflect.Value)
			}
		}
	}()

	switch m.model.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(name).Convert(m.model.Type().Key())
		if m.model.IsNil() {
			m.model.Set(reflect.MakeMap(m.model.Type()))
		}
		value = m.model.MapIndex(key)
		// use ValueFor if existence check is needed
		if !value.IsValid() {
			if m.model.Type().Elem().Kind() == reflect.Map {
				value = reflect.MakeMap(m.model.Type().Elem())
			} else {
				value = reflect.New(m.model.Type().Elem()).Elem()
			}
			m.model.SetMapIndex(key, value)
		}

	case reflect.Struct:
		value = m.model.FieldByName(name)

	case reflect.Slice, reflect.Array:
		switch name {
		case "len":
			value = reflect.ValueOf(m.model.Len())
			cache = false

		case "cap":
			value = reflect.ValueOf(m.model.Cap())
			cache = false

		default:
			index, err := strconv.Atoi(name)
			if err == nil && index >= 0 && index < m.model.Len() {
				value = m.model.Index(index)
			}
		}

	default:
		switch name {
		case "value":
			value = m.model
		}
	}

	return value
}

// Interface returns the Model's underlying object.
func (m *Model) Interface() any {
	return m.model.Interface()
}

// Type returns the type for the Model's underlying object.
func (m *Model) Type() reflect.Type {
	return m.model.Type()
}

// Elem returns the type for slice or array elements
func (m *Model) Elem() reflect.Type {
	return m.elem
}

// Keys returns all keys
func (m *Model) Keys() (keys []string) {
	switch m.model.Kind() {
	case reflect.Struct:
		for i := range m.model.NumField() {
			ft := m.model.Type().Field(i)
			if !ft.IsExported() {
				continue
			}
			keys = append(keys, ft.Name)
		}

	case reflect.Map:
		mapKeys := m.model.MapKeys()
		for _, key := range mapKeys {
			keys = append(keys, key.Convert(reflect.TypeFor[string]()).Interface().(string))
		}

	case reflect.Array, reflect.Slice:
		for i := range m.model.Len() {
			keys = append(keys, strconv.Itoa(i))
		}

	default:
		return []string{"value"}
	}
	return keys
}

// ValueAt returns an indexed value of an array or slice
func (m *Model) ValueAt(index int) any {
	if m.model.Kind() != reflect.Slice && m.model.Kind() != reflect.Array {
		return nil
	}
	if index < 0 || index >= m.model.Len() {
		return nil
	}
	return m.model.Index(index).Interface()
}

// ValueFor returns a key's value of a map.
// If the underlying object is a struct, a non-controller value is returned.
func (m *Model) ValueFor(key string) any {
	var value reflect.Value
	if m.model.Kind() != reflect.Map {
		value = m.ModelFor(key)
	} else {
		// do not act like Value, which will create the key if unset.
		// ValueFor returns nil in this case.
		value = m.model.MapIndex(reflect.ValueOf(key).Convert(m.model.Type().Key()))
	}
	if !(value.IsValid() && value.CanInterface()) {
		return nil
	}
	return value.Interface()
}

// Tag returns a struct tag for a key (if the model is a structure, of course).
// If not present or applicable to the type, returns an empty tag.
func (m *Model) Tag(key, tag string) []string {
	if m.model.Kind() != reflect.Struct {
		return nil
	}
	sf, ok := m.model.Type().FieldByName(key)
	if !ok {
		return nil
	}
	tags, ok := sf.Tag.Lookup(tag)
	if !ok || tags == "" {
		return nil
	}
	return strings.Split(tags, ",")
}

// Model returns an introspection value for the whole model
func (m *Model) Model() reflect.Value {
	return m.model
}

// ModelFor returns an introspection value for a key
func (m *Model) ModelFor(key string) reflect.Value {
	return m.key(key)
}

// Value returns the value for a key path
func (m *Model) Value(key string) any {
	if s, ok := m.sources.Load(key); ok {
		return s
	}

	v := m.ModelFor(key)
	if !(v.IsValid() && v.CanInterface()) {
		return nil
	}

	for v.Kind() == reflect.Interface {
		if !v.IsNil() {
			v = v.Elem()
		}
	}

	if tag := m.Tag(key, "controller"); len(tag) > 0 {
		if ctor, ok := controllers[tag[0]]; ok {
			s, ok := v.Interface().(observable.Source)
			if !ok {
				s = NewValue(v)
			}

			mm := ctor(s, tag[1:])
			if s, ok := m.sources.LoadOrStore(key, mm); ok {
				mm.Release()
				return s
			}

			return mm
		}

		panic(fmt.Sprintf("unregistered controller %s", tag[0]))
	}

	t := v.Type()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
		if mm := NewValue(v); mm != nil {
			if s, ok := m.sources.LoadOrStore(key, mm); ok {
				mm.Release()
				return s
			}
			return mm
		}
	}

	return v.Interface()
}

func (m *Model) SetValue(key string, value any) {
	if s, ok := m.sources.LoadAndDelete(key); ok {
		s.(observable.Observable).Release()
	}

	var valueValue reflect.Value
	switch value := value.(type) {
	case reflect.Value:
		valueValue = value

	case observable.Source:
		valueValue = value.Model()

	default:
		valueValue = reflect.ValueOf(value)
	}

	switch m.model.Kind() {
	case reflect.Map:
		if value == nil {
			m.RemoveValueFor(key)
		} else {
			m.SetValueFor(key, value)
		}

	case reflect.Slice, reflect.Array:
		index, err := strconv.Atoi(key)
		if err != nil {
			return
		}

		m.SetValueAt(index, value)

	default:
		keyValue := m.key(key)
		if !(keyValue.IsValid() || keyValue.CanSet()) {
			return
		}

		if valueValue.Type() != keyValue.Type() {
			valueValue = valueValue.Convert(keyValue.Type())
		}

		keyValue.Set(valueValue)
		m.Observe.SetValue(key, value)
	}
}

func (m *Model) updateFrom(index int, removeLast bool) {
	length := m.model.Len()
	if removeLast {
		length++
	}
	for i := index; i < length; i++ {
		key := strconv.Itoa(i)

		m.keys.Delete(key)
		if s, ok := m.sources.LoadAndDelete(key); ok {
			s.(observable.Observable).Release()
		}

		m.Observe.SetValue(key, m.ValueAt(i))
	}
}

func (m *Model) grow(n int) {
	m.model.Grow(n)
	for range n {
		m.model.Set(reflect.Append(m.model, reflect.New(m.Elem()).Elem()))
	}
	m.Observe.SetValue("len", m.model.Len())
}

// InsertValueAt inserts a value in a slice at index, increasing the length.
func (m *Model) InsertValueAt(index int, value any) {
	if m.model.Kind() != reflect.Slice {
		return
	}

	length := m.model.Len()
	if index < 0 {
		return
	}

	grow := 1
	if index >= length {
		grow += (index - length)
	}
	m.grow(grow)

	if index < length {
		reflect.Copy(
			m.model.Slice(index+1, length+1),
			m.model.Slice(index, length),
		)
	}
	m.model.Index(index).Set(reflect.ValueOf(value))

	m.Observe.InsertValueAt(index, value)
	m.Observe.SetValue("len", m.model.Len())
	m.updateFrom(index, false)
}

// RemoveValueAt removes a value from a slice at index, reducing the length.
func (m *Model) RemoveValueAt(index int) {
	if m.model.Kind() != reflect.Slice {
		return
	}

	l := m.model.Len()
	if index < 0 || index > l {
		return
	}

	if index < l-1 {
		reflect.Copy(
			m.model.Slice(index, l-1),
			m.model.Slice(index+1, l),
		)
	}
	m.model.SetLen(l - 1)

	m.Observe.SetValue(strconv.Itoa(index), nil)
	m.Observe.RemoveValueAt(index)
	m.Observe.SetValue("len", m.model.Len())
	m.updateFrom(index, true)
}

// SetValueAt sets a value in a slice or array at index.
func (m *Model) SetValueAt(index int, value any) {
	if m.model.Kind() != reflect.Slice && m.model.Kind() != reflect.Array {
		return
	}

	if index < 0 {
		return
	}
	length := m.model.Len()
	if index >= length {
		if m.model.Kind() == reflect.Array {
			return
		}
		m.grow(1 + (index - length))
	}

	key := strconv.Itoa(index)
	m.keys.Delete(key)

	v := reflect.ValueOf(value)
	if !v.IsValid() {
		v = reflect.New(m.elem).Elem()
	}

	m.model.Index(index).Set(v)

	m.Observe.SetValueAt(index, value)
	m.Observe.SetValue(key, value)
}

// SetValueFor sets a map's key value.
func (m *Model) SetValueFor(key string, value any) {
	if m.model.Kind() != reflect.Map {
		return
	}

	m.keys.Delete(key)

	keyValue := reflect.ValueOf(key)
	keyType := m.model.Type().Key()
	if keyType != reflect.TypeFor[string]() {
		keyValue = keyValue.Convert(keyType)
	}
	if m.model.IsNil() {
		m.model.Set(reflect.MakeMap(m.model.Type()))
	}
	m.model.SetMapIndex(keyValue, reflect.ValueOf(value))

	m.Observe.SetValueFor(key, value)
	m.Observe.SetValue(key, value)
}

// RemoveValueFor removes a map's key value.
func (m *Model) RemoveValueFor(key string) {
	if m.model.Kind() != reflect.Map || m.model.IsNil() {
		return
	}

	m.keys.Delete(key)

	keyType := reflect.ValueOf(key).Convert(m.model.Type().Key())
	m.model.SetMapIndex(keyType, reflect.Value{})

	m.Observe.RemoveValueFor(key)
	m.Observe.SetValue(key, nil)
}

func (m *Model) Release() {
	m.Observe.Release()
	m.sources.Range(func(_, v any) bool {
		v.(observable.Observable).Release()
		return true
	})
	m.sources.Clear()
	m.keys.Clear()
}
