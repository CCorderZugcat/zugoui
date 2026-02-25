//go:build js

package jsglue

import (
	"iter"
	"syscall/js"
)

var symbolIter = js.Global().Get("Symbol").Get("iterator")

// Iter iterates a js iterator or iterable
func Iter(iter js.Value) iter.Seq[js.Value] {
	return func(yield func(js.Value) bool) {
		if iter.Type() != js.TypeObject {
			return
		}

		values := "values"
		if symbolIter.Type() == js.TypeString {
			values = symbolIter.String()
		}

		if iter.Get(values).Type() == js.TypeFunction {
			iter = iter.Call(values)
		}

		for {
			next := iter.Call("next")

			if next.Get("done").Truthy() {
				return
			}

			if !yield(next.Get("value")) {
				ret := iter.Get("return")
				if ret.Type() == js.TypeFunction {
					iter.Call("return")
				}
				return
			}
		}
	}
}
