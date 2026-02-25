//go:build js

package input

import (
	"errors"
	"fmt"
	"reflect"
	"syscall/js"

	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/jstypes"
	"github.com/CCorderZugcat/zugoui/observable"
)

var ErrInvalidType = errors.New("invalid type")

// ValueBinding is a binding between an <input> or <fieldset> and a value of any representable go type (the model).
// <fieldset> is assumed to contain radio buttons of the same name.
type ValueBinding struct {
	observable.NullObserver
	listener js.Func                  // function for addEventListener on elem
	elem     js.Value                 // anchoring element
	property string                   // property of the elem
	source   observable.MutableSource // observerable from where key is (up changes from here, down changes to here)
	key      string                   // observerable key this control is bound to
	model    reflect.Value            // template instance of bound value
}

var _ observable.Observer = &ValueBinding{}

// NewValueBinding creates a new ValueBinding instance.
// For elem.property, create a binding to o.key
func NewValueBinding(elem js.Value, property string, source observable.MutableSource, key string) (*ValueBinding, error) {
	b := &ValueBinding{
		elem:     elem,
		property: property,
		key:      key,
		source:   observable.NewWriter(source),
	}

	model := source.Value(key)

	// create a template instance to receive the control's js value for conversion purposes
	b.model = reflect.New(reflect.ValueOf(model).Type())

	// set the control's initial value
	jsModel, ok := jstypes.ValueOf(model)
	if !ok {
		return nil, fmt.Errorf("%w. Cannot bind to type %T", ErrInvalidType, model)
	}

	Set(elem, property, js.ValueOf(jsModel))

	// binding observes the model
	b.source.AddObserver(key, b)

	// establish the event handler on the control
	// note that, of course, not all properties may be notified via "change"
	b.listener = js.FuncOf(b.eventHandler)
	elem.Call("addEventListener", "change", b.listener, map[string]any{"passive": true})

	return b, nil
}

// Destroy releases this binding.
// The event listener is removed from the element, and the js.Function instance is released.
func (b *ValueBinding) Destroy() {
	b.source.RemoveAllObservers()
	b.elem.Call("removeEventListener", "change", b.listener)
	b.listener.Release()
}

func (b *ValueBinding) eventHandler(_ js.Value, _ []js.Value) any {
	// convert js value to model's type
	jsglue.Set(b.model.Interface(), Value(b.elem))
	// observe the new change
	b.source.SetValue(b.key, b.model.Elem().Interface())

	return nil
}

// Observer interface

func (b *ValueBinding) SetValue(key string, value any) {
	if key != b.key {
		// we are in control of this logic, and if the keys don't match,
		// there is a serious code bug
		panic(fmt.Sprintf("SetValue called on wrong key %s, expected %s", key, b.key))
	}

	// set the UI control
	jsModel, ok := jstypes.ValueOf(value)
	if !ok {
		// we already established this before creating the ValueBinding
		panic(fmt.Sprintf("SetValue of un js'able value type %T", value))
	}
	Set(b.elem, b.property, js.ValueOf(jsModel))
}
