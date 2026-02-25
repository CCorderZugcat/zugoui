//go:build js

package jsglue

import "syscall/js"

// Error returns a js Error from err (returns nil if err is nil)
func Error(err error) any {
	if err == nil {
		return nil
	}
	return js.Global().Get("Error").New(err.Error())
}
