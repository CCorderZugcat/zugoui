package observable

import "reflect"

type Transformer interface {
	NewTransformer(key string, source Source) Transformer
	Mutable() bool
	Release()
	MutableSource
	Observable
}

var transformers = make(map[string]Transformer)

// RegisterTransformer is called during initialization to register a transformer by name
func RegisterTransformer(name string, x Transformer) {
	transformers[name] = x
}

// NewTransformer returns a new transformber by registered name
func NewTransformer(name string, key string, source Source) Transformer {
	x, ok := transformers[name]
	if !ok {
		return nil
	}
	return x.NewTransformer(key, source)
}

func init() {
	RegisterTransformer("isNil", &isNil{})
	RegisterTransformer("isZero", &isZero{})
	RegisterTransformer("len", &length{})
}

type transformObserver struct {
	*Observe
	xform *BaseTransform
}

func newTransformObserver(xform *BaseTransform) *transformObserver {
	return &transformObserver{
		Observe: New(),
		xform:   xform,
	}
}

func (o *transformObserver) SetValue(key string, value any) {
	if key != o.xform.key {
		return
	}
	o.Observe.SetValue("value", o.xform.valueFrom(value))
}

// BaseTransform gives base level functionality
type BaseTransform struct {
	NullSource
	NullObserver
	o         *transformObserver
	key       string
	source    Source
	valueFrom func(any) any
	valueTo   func(any) any
}

func NewBaseTransform(
	key string, source Source,
	valueFrom, valueTo func(any) any,
) *BaseTransform {
	b := &BaseTransform{
		key:       key,
		source:    source,
		valueFrom: valueFrom,
		valueTo:   valueTo,
	}
	b.o = newTransformObserver(b)
	source.AddObserver(key, b.o)
	return b
}

func (b *BaseTransform) Release() {
	b.source.RemoveObserver(b.key, b.o)
}

func (b *BaseTransform) Updating() func() {
	return b.o.Updating()
}

func (b *BaseTransform) AddObserver(key string, observer Observer) {
	b.o.AddObserver(key, observer)
}

func (b *BaseTransform) RemoveObserver(key string, observer Observer) {
	b.o.RemoveObserver(key, observer)
}

func (b *BaseTransform) Value(key string) any {
	if key != "value" || b.valueFrom == nil {
		return nil
	}
	return b.valueFrom(b.source.Value(b.key))
}

func (b *BaseTransform) Mutable() bool {
	return b.valueTo != nil
}

func (b *BaseTransform) SetValue(key string, value any) {
	if key != b.key || b.valueTo == nil {
		return
	}

	value = b.valueTo(value)

	// let this panic if not Mutable; valueTo shouldn't be set in that case
	b.source.(MutableSource).SetValue(b.key, value)
	b.o.SetValue("value", value)
}

// isNil returns true if the value is nil
type isNil struct {
	*BaseTransform
}

func (_ isNil) NewTransformer(key string, source Source) Transformer {
	return isNil{
		BaseTransform: NewBaseTransform(
			key, source,
			func(value any) any {
				if value == nil {
					return true
				}
				v := reflect.ValueOf(value)
				switch v.Kind() {
				case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
					return v.IsNil()
				default:
					return false
				}
			},
			nil,
		),
	}
}

// isZero returns true if the value is zero
type isZero struct {
	*BaseTransform
}

func (_ isZero) NewTransformer(key string, source Source) Transformer {
	return isZero{
		BaseTransform: NewBaseTransform(
			key, source,
			func(value any) any {
				if value == nil {
					return true
				}

				v := reflect.ValueOf(value)
				for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
					if v.IsNil() {
						return true
					}
					v = v.Elem()
				}

				return v.IsZero()
			},
			nil,
		),
	}
}

// len returns the length of the text input
type length struct {
	*BaseTransform
}

func (_ length) NewTransformer(key string, source Source) Transformer {
	return length{
		BaseTransform: NewBaseTransform(
			key, source,
			func(value any) any {
				if value == nil {
					return 0
				}

				v := reflect.ValueOf(value)
				for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
					if v.IsNil() {
						return 0
					}
					v = v.Elem()
				}
				if v.IsZero() {
					return 0
				}
				switch v.Kind() {
				case reflect.Map, reflect.Slice, reflect.Array, reflect.String:
					return v.Len()
				}
				return 0
			},
			nil,
		),
	}
}
