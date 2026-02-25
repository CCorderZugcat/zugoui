//go:build js

package jsglue

import (
	"errors"
	"syscall/js"
)

var ErrArgs = errors.New("args")

// AssertArgs asserts the passed in args are sufficient and of the expected types
func AssertArgs(args []js.Value, types ...js.Type) error {
	if len(args) < len(types) {
		return ErrArgs
	}
	for n, arg := range args {
		if arg.Type() != types[n] {
			return ErrArgs
		}
	}
	return nil
}
