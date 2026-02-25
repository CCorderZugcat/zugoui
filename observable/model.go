package observable

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Model allows runtime setting, getting, and observing of an arbitrary model
type Model struct {
	lck   sync.RWMutex
	model reflect.Value
	keys  sync.Map
	*Observe
}

var _ MutableSource = ((*Model)(nil))

// NewModel creates a new observeable Model instance.
func NewModel(model any) *Model {
	return NewModelValue(reflect.ValueOf(model))
}

// NewModel creates a new observeable for a model.
// v should be a value pointer to see results of mutations,
// but if it is not, then an internal copy is used.
func NewModelValue(v reflect.Value) *Model {
	o := &Model{
		Observe: New(),
	}

	v = MutableValue(v)
	if !v.IsValid() {
		return nil
	}

	o.model = v
	return o
}

func (m *Model) key(name string) reflect.Value {
	// check cache for fixed keys returning a settable reference
	if v, ok := m.keys.Load(name); ok {
		return v.(reflect.Value)
	}

	var value reflect.Value

	switch m.model.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(name).Convert(m.model.Type().Key())
		value = m.model.MapIndex(key)
		if !value.IsValid() {
			value = reflect.New(m.model.Type().Elem()).Elem()
			m.model.SetMapIndex(key, value)
		}

	case reflect.Struct:
		value = m.structKey(name)

	case reflect.Slice, reflect.Array:
		if name == "len" {
			value = reflect.ValueOf(m.model.Len())
		} else {
			index, err := strconv.Atoi(name)
			if err == nil && index >= 0 && index < m.model.Len() {
				value = m.model.Index(index)
			}
		}

	default:
		if name == "value" {
			value = m.model
		}
	}

	return value
}

func (m *Model) structKey(name string) reflect.Value {
	value := m.model.FieldByName(name)
	if v, loaded := m.keys.LoadOrStore(name, value); loaded {
		return v.(reflect.Value)
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

// ValueFor returns a key's value of a map
func (m *Model) ValueFor(key string) any {
	if m.model.Kind() != reflect.Map {
		return nil
	}
	value := m.model.MapIndex(reflect.ValueOf(key).Convert(m.model.Type().Key()))
	if !value.IsValid() {
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

// Value returns the value for a key
func (m *Model) Value(key string) any {
	v := m.ModelFor(key)
	if !v.IsValid() || !v.CanInterface() {
		return nil
	}
	return v.Interface()
}

// SetValue sets a value for a key
func (m *Model) SetValue(key string, value any) {
	valueValue := reflect.ValueOf(value)
	m.key(key).Set(valueValue)
	if m.model.Kind() == reflect.Map {
		keyValue := reflect.ValueOf(key).Convert(m.model.Type().Key())
		m.model.SetMapIndex(keyValue, valueValue)
	}
	m.Observe.SetValue(key, value)
}

// InsertValueAt inserts a value in a slice at index, increasing the length.
func (m *Model) InsertValueAt(index int, value any) {
	if m.model.Kind() != reflect.Slice {
		return
	}
	l := m.model.Len()
	if index < 0 || index > l {
		return
	}
	m.model.Set(reflect.Append(m.model, reflect.New(m.model.Type().Elem()).Elem()))
	if index < (l - 1) {
		reflect.Copy(
			m.model.Slice(index+1, l+1),
			m.model.Slice(index, l),
		)
	}
	m.model.Index(index).Set(reflect.ValueOf(value))

	m.Observe.InsertValueAt(index, value)
	if m.model.Kind() == reflect.Slice {
		m.Observe.SetValue("len", m.model.Len())
	}
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

	m.Observe.RemoveValueAt(index)
	if m.model.Kind() == reflect.Slice {
		m.Observe.SetValue("len", m.model.Len())
	}
}

// SetValueAt sets a value in a slice or array at index.
func (m *Model) SetValueAt(index int, value any) {
	if m.model.Kind() != reflect.Slice && m.model.Kind() != reflect.Array {
		return
	}
	if index < 0 || index >= m.model.Len() {
		return
	}
	m.model.Index(index).Set(reflect.ValueOf(value))

	m.Observe.SetValueAt(index, value)
}

// SetValueFor sets a map's key value.
func (m *Model) SetValueFor(key string, value any) {
	if m.model.Kind() != reflect.Map {
		return
	}
	if m.model.IsNil() {
		m.model.Set(reflect.MakeMap(m.model.Type()))
	}
	keyType := reflect.ValueOf(key).Convert(m.model.Type().Key())
	m.model.SetMapIndex(keyType, reflect.ValueOf(value))

	m.Observe.SetValueFor(key, value)
}

// RemoveValueFor re3moves a map's key value.
func (m *Model) RemoveValueFor(key string) {
	if m.model.Kind() != reflect.Map {
		return
	}
	keyType := reflect.ValueOf(key).Convert(m.model.Type().Key())
	m.model.SetMapIndex(keyType, reflect.Value{})

	m.Observe.RemoveValueFor(key)
}
