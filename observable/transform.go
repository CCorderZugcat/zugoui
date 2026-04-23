package observable

import (
	"reflect"
)

type Transformer interface {
	NewTransformer() Transformer
	Mutable() bool
	Get(any) any
	Set(any) any
}

var transformers = make(map[string]Transformer)

// RegisterTransformer is called during initialization to register a transformer by name
func RegisterTransformer(name string, x Transformer) {
	transformers[name] = x
}

// NewTransformer returns a new transformber by registered name
func NewTransformer(name string) Transformer {
	x, ok := transformers[name]
	if !ok {
		return nil
	}
	return x.NewTransformer()
}

func init() {
	RegisterTransformer("isNil", &isNil{})
	RegisterTransformer("isZero", &isZero{})
	RegisterTransformer("len", &length{})
}

// BaseTransform gives base level functionality
type BaseTransform struct {
	valueFrom func(any) any
	valueTo   func(any) any
}

func NewBaseTransform(
	valueFrom, valueTo func(any) any,
) *BaseTransform {
	b := &BaseTransform{
		valueFrom: valueFrom,
		valueTo:   valueTo,
	}
	return b
}

func (b *BaseTransform) Get(value any) any {
	if b.valueFrom == nil {
		return nil
	}
	return b.valueFrom(value)
}

func (b *BaseTransform) Mutable() bool {
	return b.valueTo != nil
}

func (b *BaseTransform) Set(value any) any {
	if b.valueTo == nil {
		return nil
	}
	return b.valueTo(value)
}

// isNil returns true if the value is nil
type isNil struct {
	*BaseTransform
}

func (_ isNil) NewTransformer() Transformer {
	return isNil{
		BaseTransform: NewBaseTransform(
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

func (_ isZero) NewTransformer() Transformer {
	return isZero{
		BaseTransform: NewBaseTransform(
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

func (_ length) NewTransformer() Transformer {
	return length{
		BaseTransform: NewBaseTransform(
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
