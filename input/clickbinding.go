//go:build js

package input

import "syscall/js"

// ClickBinding is a binding between any clickable element (usually a <button>) and an action.
type ClickBinding struct {
	listener js.Func
	elem     js.Value
	name     string
	action   func(string)
}

// NewClickBinding creates a new ClickBinding instance.
// When clicked, action is called with name as the parameter.
// The action should not block.
func NewClickBinding(elem js.Value, name string, action func(string)) *ClickBinding {
	c := &ClickBinding{
		elem:   elem,
		name:   name,
		action: action,
	}
	c.listener = js.FuncOf(c.eventListener)
	elem.Call("addEventListener", "click", c.listener, map[string]any{"passive": true})

	return c
}

// Destroy releases this binding.
// The event listener is removed and the js.Function instance released.
func (c *ClickBinding) Destroy() {
	c.elem.Call("removeEventListener", "click", c.listener)
	c.listener.Release()
}

func (c *ClickBinding) eventListener(_ js.Value, _ []js.Value) any {
	c.action(c.name)
	return nil
}
