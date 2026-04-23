package observable

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var (
	ErrImmutable = errors.New("object is not a MutableSource")
	ErrNotSource = errors.New("object is not a Source")
)

// PathObserver implements half of a binding by observing a key path.
// PathObserver is not usually used by itself, but is an important component of
// the Binding object.
// Any mutations within the key path are observed, and the last component of the
// key path may be * to indicate a wildcard.
// Wildcards are recusive, so any key paths deeper than the wildcard are observed.
// PathObserver also acts as a mutable source which will set the underlying
// key paths without causing its own notification.
type PathObserver struct {
	*Observe
	prefix, suffix string
	source         Source
	root           *PathObserver
	down           map[string]*PathObserver
}

type pathObserverObserver struct {
	o *PathObserver
	NullObserver
}

func (p *pathObserverObserver) SetValue(key string, value any) {
	p.o.setValue(key, value)
}

// PathSetter allows an object's key path to be set,
// if the leaf object is a MutableSource.
// This object exists as a way to observe a PathObserver.
type PathSetter struct {
	Source
	NullObserver
}

func NewPathSetter(s Source) PathSetter { return PathSetter{Source: s} }

// SetKeyPath is a convenience method to set a key path.
func SetKeyPath(s Source, keyPath string, value any) error {
	key, suffix, _ := strings.Cut(keyPath, ".")
	if suffix == "" {
		m, ok := s.(MutableSource)
		if !ok {
			return fmt.Errorf("%s: %w", keyPath, ErrImmutable)
		}
		m.SetValue(key, value)
		return nil
	}

	s, ok := s.Value(key).(Source)
	if !ok {
		return fmt.Errorf("%s: %w", keyPath, ErrNotSource)
	}

	return SetKeyPath(s, suffix, value)
}

// GetKeyPath is a convenience method to get a key path.
func GetKeyPath(s Source, keyPath string) any {
	key, suffix, _ := strings.Cut(keyPath, ".")
	if suffix == "" {
		return s.Value(key)
	}

	s, ok := s.Value(key).(Source)
	if !ok {
		return nil
	}

	return GetKeyPath(s, suffix)
}

// NewPathObserver creates a new PathObserver.
// the last component may be * to wildcard.
func NewPathObserver(keyPath string, source Source) *PathObserver {
	var o *PathObserver // nil
	return o.newPathObserver("", keyPath, source, false)
}

func (o *PathObserver) newPathObserver(prefix, keyPath string, source Source, notify bool) *PathObserver {
	key, suffix, _ := strings.Cut(keyPath, ".")
	if key == "*" {
		key = ""
		suffix = "*"
	}

	oo := &PathObserver{
		Observe: New(),
		suffix:  suffix,
		source:  source,
		down:    make(map[string]*PathObserver),
	}
	if o == nil {
		oo.root = oo
		oo.prefix = prefix
	} else {
		oo.root = o.root
		oo.prefix = JoinKeyPath(o.prefix, prefix)
		o.down[prefix] = oo
	}
	source.AddObserver(key, &pathObserverObserver{o: oo})

	var keys []string
	if key == "" {
		keys = source.Keys()
	} else {
		keys = []string{key}
	}

	for _, key := range keys {
		value := source.Value(key)
		if s, ok := value.(Source); ok {
			if suffix != "" {
				oo.newPathObserver(key, suffix, s, notify)
			}
		} else {
			if notify {
				oo.root.Observe.SetValue(JoinKeyPath(oo.prefix, key), value)
			}
		}
	}

	return oo
}

func (o *PathObserver) setValue(key string, value any) {
	keyPath := JoinKeyPath(o.prefix, key)
	o.root.Observe.SetValue(keyPath, value)

	if v, ok := o.down[key]; ok {
		v.Release()
		delete(o.down, key)
	}

	if o.suffix != "" {
		if o.suffix != "*" && value == nil {
			o.root.Observe.SetValue(JoinKeyPath(o.prefix, key, o.suffix), nil)
		} else if s, ok := o.source.Value(key).(Source); ok {
			o.newPathObserver(key, o.suffix, s, true)
		}
	}
}

func (o *PathObserver) Release() {
	for _, v := range o.down {
		v.Release()
	}
	clear(o.down)
	o.Observe.Release()
}

func (o *PathObserver) atPrefix(keyPath string) (*PathObserver, string) {
	key, suffix, _ := strings.Cut(keyPath, ".")
	if suffix == "" {
		return o, key
	}

	if v, ok := o.down[key]; ok {
		return v.atPrefix(suffix)
	}

	s, ok := o.source.Value(key).(Source)
	if !ok {
		return nil, ""
	}

	oo := o.newPathObserver(key, suffix, s, false)
	return oo.atPrefix(suffix)
}

func (o *PathObserver) SetValue(keyPath string, value any) {
	done := o.root.Observe.Updating()
	defer done()

	o, key := o.atPrefix(keyPath)
	if o == nil {
		return
	}

	o.source.(MutableSource).SetValue(key, value)
}

func (o *PathObserver) SetValueFor(keyPath string, value any) {
	done := o.root.Observe.Updating()
	defer done()

	o, key := o.atPrefix(keyPath)
	if o == nil {
		return
	}

	o.source.(MutableSource).SetValueFor(key, value)
}

func (o *PathObserver) Value(keyPath string) any {
	o, key := o.atPrefix(keyPath)
	if o == nil {
		return nil
	}

	return o.source.Value(key)
}

func (o *PathObserver) ValueFor(keyPath string) any {
	o, key := o.atPrefix(keyPath)
	if o == nil {
		return nil
	}

	return o.source.ValueFor(key)
}

func (o *PathObserver) ModelFor(keyPath string) reflect.Value {
	o, key := o.atPrefix(keyPath)
	if o == nil {
		return reflect.Value{}
	}

	return o.source.ModelFor(key)
}

func (o *PathObserver) Model() reflect.Value         { return o.source.Model() }
func (o *PathObserver) Keys() []string               { return nil }
func (o *PathObserver) Elem() reflect.Type           { return nil }
func (o *PathObserver) Tag(key, tag string) []string { return nil }

func (o *PathObserver) ValueAt(index int) any {
	return o.source.ValueAt(index)
}

func (s PathSetter) keyPath(keyPath string, f func(m MutableSource, key string)) {
	key, suffix, _ := strings.Cut(keyPath, ".")
	if suffix == "" {
		if m, ok := s.Source.(MutableSource); ok {
			f(m, key)
		}
		return
	}

	if s, ok := s.Value(key).(Source); ok {
		PathSetter{Source: s}.keyPath(suffix, f)
	}
}

func (s PathSetter) SetValue(keyPath string, value any) {
	s.keyPath(keyPath, func(m MutableSource, key string) {
		m.SetValue(key, value)
	})
}

func (s PathSetter) SetValueFor(keyPath string, value any) {
	s.keyPath(keyPath, func(m MutableSource, key string) {
		m.SetValueFor(key, value)
	})
}

func (s PathSetter) RemoveValueFor(keyPath string) {
	s.keyPath(keyPath, func(m MutableSource, key string) {
		m.RemoveValueFor(key)
	})
}

func JoinKeyPath(parts ...string) string {
	keyPath := &strings.Builder{}

	for _, p := range parts {
		if p == "" {
			continue
		}
		if keyPath.Len() > 0 {
			keyPath.WriteRune('.')
		}
		keyPath.WriteString(p)
	}

	return keyPath.String()
}
