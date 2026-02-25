//go:build js

package input_test

import (
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/formtest"
	"github.com/CCorderZugcat/zugoui/input"
)

func TestInput(t *testing.T) {
	formtest.SetBody(t, fsys, "value.html")

	field1, err := input.Element("field1")
	require.NoError(t, err)

	field2, err := input.Element("field2")
	require.NoError(t, err)

	field3, err := input.Element("field3")
	require.NoError(t, err)

	assert.Equal(t, "bread", input.Value(field1).String())

	assert.False(t, input.Value(field2).Truthy())
	assert.Equal(t, js.TypeBoolean, input.Value(field2).Type())

	assert.Equal(t, "swiss", input.Value(field3).String())

	// change the cheese
	assert.True(t, input.Set(field3, "value", js.ValueOf("cheddar")))
	assert.Equal(t, "cheddar", input.Value(field3).String())

	// toast the bread
	assert.True(t, input.Set(field2, "value", js.ValueOf(true)))
	assert.True(t, input.Value(field2).Truthy())

	// name the sandwich
	assert.True(t, input.Set(field1, "value", js.ValueOf("fred")))
	assert.Equal(t, "fred", input.Value(field1).String())
}
