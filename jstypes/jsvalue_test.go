package jstypes_test

import (
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/jstypes"
)

type myString string

type torture struct {
	String  string
	Int     int
	Uint    uint
	Float32 float32
	Float64 float64
	Bool    bool
	Array   [2]*torture
	Slice   []*torture
	Map     map[myString]*torture
}

func TestValueOf(t *testing.T) {
	i := &torture{
		String:  "hello",
		Int:     12,
		Uint:    34,
		Float32: 45.6,
		Float64: 7.89,
		Bool:    true,
		Map:     make(map[myString]*torture),
	}
	i.Slice = []*torture{{Int: 1}, {Int: 2}}
	i.Array[1] = &torture{Int: 3}
	i.Map["foo"] = &torture{Int: 1}
	i.Map["bar"] = &torture{Int: 2}

	v, ok := jstypes.ValueOf(i)
	require.True(t, ok)
	t.Logf("%+v", v)

	m := v.(map[string]any)
	assert.Equal(t, 12, m["Int"])
	assert.True(t, m["Bool"].(bool))

	ar := m["Slice"].([]any)
	assert.Equal(t, 1, ar[0].(map[string]any)["Int"])

	ar = m["Array"].([]any)
	assert.Equal(t, 3, ar[1].(map[string]any)["Int"])
	assert.Nil(t, ar[0])

	mm := m["Map"].(map[string]any)
	assert.Equal(t, 2, mm["bar"].(map[string]any)["Int"])
}

func TestIters(t *testing.T) {
	ints := []int{1, 2, 3, 4, 5}
	v, ok := jstypes.ValueOf(slices.Values(ints))
	require.True(t, ok)

	assert.Equal(t, []any{1, 2, 3, 4, 5}, v)

	keyValues := map[string]int{"one": 1, "two": 2}
	v, ok = jstypes.ValueOf(maps.All(keyValues))

	assert.Equal(t, v, map[string]any{"one": 1, "two": 2})
}

type myColor int

var errBadColor = errors.New("bad color value")

func (m myColor) MarshalText() ([]byte, error) {
	switch m {
	case 0:
		return []byte("red"), nil
	case 1:
		return []byte("green"), nil
	case 2:
		return []byte("blue"), nil
	}
	return nil, errBadColor
}

func TestTextMarshaler(t *testing.T) {
	v, ok := jstypes.ValueOf(myColor(1))
	assert.True(t, ok)
	assert.Equal(t, "green", v)

	_, ok = jstypes.ValueOf(myColor(-1))
	assert.False(t, ok)
}
