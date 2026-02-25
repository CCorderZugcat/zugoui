package observable_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CCorderZugcat/zugoui/observable"
)

func TestMutableValue(t *testing.T) {
	type s struct{ field string }
	var ps *s

	types := []struct {
		name     string
		instance any
		expect   reflect.Type
	}{
		{
			name:     "nil interface",
			instance: nil,
			expect:   nil,
		},
		{
			name:     "non addressable",
			instance: 5,
			expect:   nil,
		},
		{
			name:     "addressable",
			instance: new(5),
			expect:   reflect.TypeFor[int](),
		},
		{
			name:     "adressable struct",
			instance: &s{},
			expect:   reflect.TypeFor[s](),
		},
		{
			name:     "nil pointer to struct",
			instance: &ps,
			expect:   reflect.TypeFor[s](),
		},
	}

	for _, typ := range types {
		t.Run(typ.name, func(t *testing.T) {
			v := observable.MutableValue(reflect.ValueOf(typ.instance))
			if typ.expect == nil {
				assert.Zero(t, v)
			} else {
				assert.Equalf(t, typ.expect, v.Type(), "%v != %v", typ.expect, v.Type())
			}
		})
	}
}
