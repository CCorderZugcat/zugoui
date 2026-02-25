//go:build js

package jsglue

import "syscall/js"

var Event = js.Global().Get("Event")
var CustomEvent = js.Global().Get("CustomEvent")

// DispatchEvent dispatches an Event to a target
func DispatchEvent(target js.Value, name string, options map[string]any) {
	ev := Event.New(name, options)
	target.Call("dispatchEvent", ev)
}

func NewCustomEvent(name string, detail any) js.Value {
	return CustomEvent.New(name, map[string]any{
		"detail": detail,
	})
}
