//go:build js

package input

import (
	"errors"
	"fmt"
	"syscall/js"

	"github.com/CCorderZugcat/zugoui/jsglue"
)

var ErrNoElement = errors.New("element not found")

var document = js.Global().Get("document")

// Element finds an element by id
func Element(id string) (js.Value, error) {
	elem := document.Call("querySelector", fmt.Sprintf("#%s", id))
	if elem.Type() != js.TypeObject {
		return js.Undefined(), fmt.Errorf("%w: id %s", ErrNoElement, id)
	}
	return elem, nil
}

// Value does a best effort conversion from the input or fieldgroup's buttons to the intended value
func Value(elem js.Value) js.Value {
	typ := elem.Get("type")
	if typ.Type() != js.TypeString {
		return js.Null()
	}

	switch typ.String() {
	case "fieldset": // presume it's full of buttons
		// also presume we did not put two sets of names in it
		v := elem.Call("querySelector", `input[type="radio"]:checked`)
		if v.IsNull() {
			return js.Null()
		}
		return v.Get("value")

	case "checkbox":
		return elem.Get("checked")

	default: // go with value and hope for the best
		return elem.Get("value")
	}
}

// Set sets the property of a control.
// If the property is "value", special case radio buttons and check boxes.
func Set(elem js.Value, property string, value js.Value) bool {
	if property != "value" {
		elem.Set(property, value)
		return true
	}

	typ := elem.Get("type")
	if typ.Type() != js.TypeString {
		return false
	}

	switch typ.String() {
	case "fieldset":
		v := elem.Call("querySelectorAll", fmt.Sprintf(`input[type="radio"]:not([value="%v"])`, value))
		for elem := range jsglue.Iter(v) {
			elem.Set("checked", false)
		}
		v = elem.Call("querySelector", fmt.Sprintf(`input[type="radio"][value="%v"]`, value))
		if v.IsNull() {
			return false
		}
		v.Set("checked", true)

	case "checkbox":
		elem.Set("checked", value.Truthy())

	default:
		elem.Set("value", value)
	}

	return true
}
