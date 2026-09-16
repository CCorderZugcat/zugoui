//go:build js

package input_test

import (
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zugcat/zugoui/formtest"
	"github.com/zugcat/zugoui/input"
	"github.com/zugcat/zugoui/jsglue"
	"github.com/zugcat/zugoui/observable"
	"github.com/zugcat/zugoui/observable/controllers"
	"github.com/zugcat/zugoui/observable/controllers/scroll"
)

type sandwich struct {
	Product  string `bind:"product"`
	Toasted  bool   `bind:"toasted"`
	Cheese   string `bind:"cheese"`
	Quantity int    `bind:"quantity"`
	TextMe   string `bind:"textme>innerText"`
}

func TestValueBindings(t *testing.T) {
	formtest.SetBody(t, fsys, "value.html")

	s := &sandwich{
		Product:  "toast",
		Quantity: 5,
		TextMe:   "inner text",
		Cheese:   "cheddar",
	}
	m := controllers.New(s)
	require.NotNil(t, m)
	defer m.Release()

	vb, err := input.NewValueBindings([]string{"sandwich"}, m)
	require.NoError(t, err)
	defer vb.Release()

	elem, err := input.Element("sandwich.quantity")
	require.NoError(t, err)

	input.Set(elem, "value", js.ValueOf(10))
	jsglue.DispatchEvent(elem, "input", map[string]any{"bubbles": true})

	assert.Equal(t, 10, s.Quantity)

	m.SetValue("TextMe", "this is some text")
	elem, err = input.Element("sandwich.textme")
	require.NoError(t, err)
	assert.Equal(t, "this is some text", elem.Get("innerText").String())

	elem, err = input.Element("checkme")
	require.NoError(t, err)

	jsglue.DispatchMouseEvent(elem, "click", map[string]any{
		"bubbles":    true,
		"cancelable": true,
		"view":       js.Global().Get("window"),
		"button":     0,
	})

	assert.Equal(t, "shiny", s.Cheese)
}

func TestArrayBindings(t *testing.T) {
	formtest.SetBody(t, fsys, "array.html")

	type topping struct {
		Topping string `bind:"topping,>hidden;isZero"`
	}
	type pizza struct {
		Size     string      `bind:"size"`
		Toppings [3]*topping `bind:"toppings"`
	}

	p := &pizza{
		Size: "medium",
		Toppings: [3]*topping{
			{
				Topping: "raisins",
			},
			nil,
			nil,
		},
	}
	m := controllers.New(p)
	require.NotNil(t, m)
	defer m.Release()

	vb, err := input.NewValueBindings([]string{"pizza"}, m)
	require.NoError(t, err)
	defer vb.Release()

	elem, err := input.Element("pizza.toppings.1")
	require.NoError(t, err)
	assert.True(t, elem.Get("hidden").Truthy())

	elem, err = input.Element("pizza.toppings.0")
	require.NoError(t, err)
	assert.False(t, elem.Get("hidden").Truthy())

	elem, err = input.Element("pizza.toppings.0.topping")
	require.NoError(t, err)
	assert.Equal(t, "raisins", input.Value(elem).String())

	m.SetValue("Size", "large")
	elem, err = input.Element("pizza.size")
	require.NoError(t, err)
	assert.Equal(t, "large", input.Value(elem).String())

	elem, err = input.Element("pizza.toppings.0.topping")
	require.NoError(t, err)
	elem.Set("value", "broccoli")
	jsglue.DispatchEvent(elem, "change", map[string]any{"bubbles": true})

	assert.Equal(t, "broccoli", p.Toppings[0].Topping)

	elem, err = input.Element("pizza.toppings.1")
	require.NoError(t, err)
	assert.True(t, elem.Get("hidden").Truthy())

	require.NoError(t, observable.SetKeyPath(m, "Toppings.1.Topping", "Nutella"))
	assert.False(t, elem.Get("hidden").Truthy())
	elem, err = input.Element("pizza.toppings.1.topping")
	require.NoError(t, err)
	assert.Equal(t, "Nutella", input.Value(elem).String())

	t.Log(p.Toppings[1].Topping)
}

func TestScrollBindings(t *testing.T) {
	formtest.SetBody(t, fsys, "array.html")

	type topping struct {
		Topping string `bind:"topping,>hidden;isZero"`
	}
	type pizza struct {
		Size     string     `bind:"size"`
		Toppings []*topping `bind:"toppings" controller:"scroll,3"`
	}

	p := &pizza{
		Size: "medium",
		Toppings: []*topping{
			{
				Topping: "raisins",
			},
		},
	}
	m := controllers.New(p)
	require.NotNil(t, m)
	defer m.Release()

	vb, err := input.NewValueBindings([]string{"pizza"}, m)
	require.NoError(t, err)
	defer vb.Release()

	elem, err := input.Element("pizza.toppings.1")
	require.NoError(t, err)
	assert.True(t, elem.Get("hidden").Truthy())

	s := m.Value("Toppings").(*scroll.Scroll)
	s.Insert("insert")

	assert.False(t, elem.Get("hidden").Truthy())
	t.Log(p.Toppings[1].Topping)
}
