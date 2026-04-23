//go:build js

package input_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/formtest"
	"github.com/CCorderZugcat/zugoui/input"
	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/observable/controllers"
)

func TestInput(t *testing.T) {
	formtest.SetBody(t, fsys, "value.html")

	elem, err := input.Element("field1")
	require.NoError(t, err)

	i := input.NewInput(elem)
	defer i.Release()

	assert.Equal(t, "bread", i.Value("value"))

	changes := make(map[string]any)
	m := controllers.New(changes)
	require.NotNil(t, m)
	defer m.Release()

	i.AddObserver("value", m)

	elem.Set("value", "toast")
	jsglue.DispatchEvent(elem, "change", map[string]any{"bubbles": true})

	assert.Equal(t, "toast", i.Value("value"))
	assert.Equal(t, "toast", changes["value"])
}
