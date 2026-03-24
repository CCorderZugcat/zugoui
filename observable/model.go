package observable

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Model allows runtime setting, getting, and observing of an arbitrary model
type Model struct {
	model reflect.Value
	elem  reflect.Type
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

	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		o.elem = v.Type().Elem()
	}

	return o
}

func (m *Model) key(name string, set bool) reflect.Value {
	if !set {
		if v, ok := m.keys.Load(name); ok {
			return v.(reflect.Value)
		}
	}

	var value reflect.Value

	switch m.model.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(name).Convert(m.model.Type().Key())
		if m.model.IsNil() {
			m.model.Set(reflect.MakeMap(m.model.Type()))
		}
		value = m.model.MapIndex(key)
		if !value.IsValid() {
			if !set {
				return value
			}
			value = reflect.New(m.model.Type().Elem()).Elem()
			m.model.SetMapIndex(key, value)
		}
		if !set {
			value = m.modelKey(name, value)
		}

	case reflect.Struct:
		value = m.model.FieldByName(name)
		if !set {
			value = m.modelKey(name, value)
		}

	case reflect.Slice, reflect.Array:
		switch name {
		case "len":
			value = reflect.ValueOf(m.model.Len())

		case "cap":
			value = reflect.ValueOf(m.model.Cap())

		default:
			index, err := strconv.Atoi(name)
			if err == nil && index >= 0 && index < m.model.Len() {
				value = m.model.Index(index)
			}
		}

		if !set && value.IsValid() {
			value = m.modelKey(name, value)
		}

	default:
		switch name {
		case "value":
			value = m.model
		}
	}

	return value
}

func (m *Model) modelKey(name string, value reflect.Value) (ret reflect.Value) {
	defer func() {
		if v, loaded := m.keys.LoadOrStore(name, ret); loaded {
			ret = v.(reflect.Value) // let first one win if raced
		}
	}()

	var t reflect.Type
	if !value.IsValid() {
		t = m.elem
	} else {
		t = value.Type()
	}
	if t == nil {
		return value
	}

	if t.Implements(reflect.TypeFor[MutableSource]()) {
		return value
	}

	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
	default:
		return value
	}

	s := NewModelValue(value)
	if s == nil {
		return value
	}
	return reflect.ValueOf(s)
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
	return m.key(key, false)
}

// Value returns the value for a key path
func (m *Model) Value(keyPath string) any {
	s := Source(m)
	var v reflect.Value

	for key := range strings.SplitSeq(keyPath, ".") {
		if s == nil {
			return nil
		}
		v = s.ModelFor(key)
		if !v.IsValid() || !v.CanInterface() {
			return nil
		}
		if v.Type().Implements(reflect.TypeFor[Source]()) {
			s = v.Interface().(Source)
		} else {
			s = nil
		}
	}

	return v.Interface()
}

// SetValue sets a value for a key path
func (m *Model) SetValue(keyPath string, value any) {
	defer func() {
		// if we've added a new datasource, inform observers
		if s, ok := m.Value(keyPath).(Source); ok {
			for _, key := range s.Keys() {
				v := s.Value(key)
				if v == nil {
					continue
				}
				if vs, ok := v.(Source); ok {
					v = vs.Model().Interface()
				}
				m.SetValue(keyPath+"."+key, v)
			}
		}
	}()

	components := strings.Split(keyPath, ".")

	if len(components) == 1 {
		m.setValue(keyPath, value)
		return
	}

	m.Observe.SetValue(keyPath, value)

	v := m.Value(components[0])
	if v == nil {
		if elem := m.Elem(); elem != nil {
			if elem.Kind() == reflect.Map {
				v = reflect.MakeMap(elem).Interface()
			} else {
				v = reflect.New(elem).Elem().Interface()
			}
			m.setValue(components[0], v)
			v = m.Value(components[0])
		}
		if v == nil {
			return
		}
	}
	if s, ok := v.(MutableSource); ok {
		s.SetValue(strings.Join(components[1:], "."), value)
	}
}

func (m *Model) setValue(key string, value any) {
	valueValue := reflect.ValueOf(value)

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
		keyValue := m.key(key, true)
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

func (m *Model) updateFrom(index int) {
	for i := index; i < m.model.Len(); i++ {
		key := strconv.Itoa(i)

		m.keys.Delete(key)
		m.Observe.SetValue(key, m.ValueAt(i))
	}
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
	if index < l {
		reflect.Copy(
			m.model.Slice(index+1, l+1),
			m.model.Slice(index, l),
		)
	}
	m.model.Index(index).Set(reflect.ValueOf(value))

	m.Observe.InsertValueAt(index, value)
	m.Observe.SetValue("len", m.model.Len())
	m.updateFrom(index)
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

	key := strconv.Itoa(index)
	m.keys.Delete(key)

	m.Observe.SetValue(key, nil)
	m.Observe.RemoveValueAt(index)
	m.Observe.SetValue("len", m.model.Len())
	m.updateFrom(index)
}

// SetValueAt sets a value in a slice or array at index.
func (m *Model) SetValueAt(index int, value any) {
	if m.model.Kind() != reflect.Slice && m.model.Kind() != reflect.Array {
		return
	}

	if index < 0 || index >= m.model.Len() {
		return
	}

	key := strconv.Itoa(index)
	m.keys.Delete(key)

	m.model.Index(index).Set(reflect.ValueOf(value))

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
