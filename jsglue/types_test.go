//go:build js

package jsglue_test

import (
	"errors"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/jsglue"
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

func TestMap(t *testing.T) {
	m1 := map[string]int{
		"one": 1,
		"two": 2,
	}
	v, ok := jstypes.ValueOf(m1)
	require.True(t, ok)

	var m2 map[string]int
	ok = jsglue.Set(&m2, js.ValueOf(v))
	require.True(t, ok)

	assert.Equal(t, m1, m2)
}

func TestSlice(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	v, ok := jstypes.ValueOf(s1)
	require.True(t, ok)

	var s2 []int
	ok = jsglue.Set(&s2, js.ValueOf(v))
	require.True(t, ok)

	assert.Equal(t, s1, s2)
}

func TestArray(t *testing.T) {
	a1 := [2]int{1, 2}
	v, ok := jstypes.ValueOf(a1)
	require.True(t, ok)

	var a2 [5]int
	ok = jsglue.Set(&a2, js.ValueOf(v))
	require.True(t, ok)

	assert.Equal(t, a1[:2], a2[:2])
}

func TestNumber(t *testing.T) {
	i1 := 1
	f1 := 2.3

	vi, ok := jstypes.ValueOf(i1)
	require.True(t, ok)

	vf, ok := jstypes.ValueOf(f1)
	require.True(t, ok)

	var i2 int
	var f2 float64

	ok = jsglue.Set(&i2, js.ValueOf(vi))
	require.True(t, ok)

	ok = jsglue.Set(&f2, js.ValueOf(vf))
	require.True(t, ok)

	assert.Equal(t, i1, i2)
	assert.Equal(t, f1, f2)
}

func TestPointerToPointer(t *testing.T) {
	type model struct {
		Field1 string
	}

	v, ok := jstypes.ValueOf(&model{Field1: "hello"})
	require.True(t, ok)

	var m *model
	ok = jsglue.Set(&m, js.ValueOf(v))
	require.True(t, ok)

	assert.Equal(t, "hello", m.Field1)
}

func TestRoundTrip(t *testing.T) {
	i1 := &torture{
		String:  "hello",
		Int:     12,
		Uint:    34,
		Float32: 45.6,
		Float64: 7.89,
		Bool:    true,
		Map:     make(map[myString]*torture),
	}
	i1.Slice = []*torture{{Int: 1}, {Int: 2}}
	i1.Array[1] = &torture{Int: 3}
	i1.Map["foo"] = &torture{Int: 1}
	i1.Map["bar"] = &torture{Int: 2}

	v, ok := jstypes.ValueOf(i1)
	require.True(t, ok)

	jv := js.ValueOf(v)
	i2 := &torture{}

	ok = jsglue.Set(i2, jv)
	require.True(t, ok)

	assert.Equal(t, i1, i2)
}

type myColor int

var errBadColor = errors.New("bad color value")

func (m *myColor) UnmarshalText(txt []byte) error {
	switch string(txt) {
	case "red":
		*m = 0
	case "green":
		*m = 1
	case "blue":
		*m = 2
	default:
		return errBadColor
	}
	return nil
}

func TestTextUnmarshal(t *testing.T) {
	var c myColor
	ok := jsglue.Set(&c, js.ValueOf("green"))
	assert.True(t, ok)
	assert.Equal(t, myColor(1), c)
}
