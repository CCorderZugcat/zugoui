//go:build js

package input_test

import (
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/formtest"
	"github.com/CCorderZugcat/zugoui/input"
	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/observabletest"
)

func TestSimpleValueBinding(t *testing.T) {
	formtest.SetBody(t, fsys, "value.html")

	field1, err := input.Element("field1")
	require.NoError(t, err)

	field4, err := input.Element("field4")
	require.NoError(t, err)

	model := &struct {
		Product  string
		Quantity int
	}{
		Product:  "initial value",
		Quantity: 5,
	}
	o := observable.NewModel(model)

	ov, ch := observabletest.New()
	defer close(ch)
	o.AddObserver("", ov)

	b, err := input.NewValueBinding(field1, "value", o, "Product")
	require.NoError(t, err)
	defer b.Destroy()

	b2, err := input.NewValueBinding(field4, "value", o, "Quantity")
	require.NoError(t, err)
	defer b2.Destroy()

	field1.Set("value", "Goodbye")
	jsglue.DispatchEvent(field1, "change", map[string]any{"bubbles": true})

	ov1 := <-ch
	assert.Equal(t, "Goodbye", ov1.Value)

	field4.Set("value", 3)
	jsglue.DispatchEvent(field4, "change", map[string]any{"bubbles": true})

	ov1 = <-ch
	assert.Equal(t, 3, ov1.Value)

	assert.Equal(t, "Goodbye", model.Product)

	t.Log(model.Product)
}

func TestRadio(t *testing.T) {
	formtest.SetBody(t, fsys, "value.html")

	field3, err := input.Element("field3")
	require.NoError(t, err)

	cheese := "swiss"
	o := observable.NewModel(&cheese)
	b, err := input.NewValueBinding(field3, "value", o, "value")
	require.NoError(t, err)
	defer b.Destroy()

	ov, ch := observabletest.New()
	defer close(ch)
	o.AddObserver("value", ov)

	cheddar := js.Global().Get("document").Call("querySelector", `#field3 input[value="cheddar"]`)
	require.Equal(t, js.TypeObject, cheddar.Type())

	cheddar.Call("click")

	ov1 := <-ch

	assert.Equal(t, "cheddar", cheese)
	assert.Equal(t, "cheddar", ov1.Value)

	t.Log(ov1.Value)
}
