//go:build js

package input

import "syscall/js"

// ClickBinding is a binding between any clickable element (usually a <button>) and an action.
type ClickBinding struct {
	id       string
	listener js.Func
	elem     js.Value
	name     string
	action   func(string)
}

// NewClickBinding creates a new ClickBinding instance.
// When clicked, action is called with name as the parameter.
// The action should not block.
func NewClickBinding(id string, name string, action func(string)) *ClickBinding {
	c := &ClickBinding{
		id:     id,
		name:   name,
		action: action,
	}
	c.listener = js.FuncOf(c.eventListener)
	c.bind() // attempt if the element is there

	return c
}

func (c *ClickBinding) bind() {
	elem, err := Element(c.id)
	if err != nil {
		return
	}
	c.elem = elem
	elem.Call("addEventListener", "click", c.listener, map[string]any{"passive": true})
}

// Rebind follows the Rebind interface allowing this binding to
// hook back up to a dynamically rendered document or reconnection.
func (c *ClickBinding) Rebind() {
	if c.elem.Type() == js.TypeObject {
		c.elem.Call("removeEventListener", "click", c.listener)
	}

	c.bind()
}

// Release releases this binding.
// The event listener is removed and the js.Function instance released.
func (c *ClickBinding) Release() {
	c.elem.Call("removeEventListener", "click", c.listener)
	c.listener.Release()
}

func (c *ClickBinding) eventListener(_ js.Value, _ []js.Value) any {
	c.action(c.name)
	return nil
}
