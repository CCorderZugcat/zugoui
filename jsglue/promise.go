//go:build js

package jsglue

import "syscall/js"

// Promise returns a Promise js.Value.
// work is called in a go routine, and upon exit, the promise is fullfilled.
// If work returns a non-nil error, reject.
// Othrwise, resolve with the returned argumenets.
// The return slice is expanded into a variadic for resolve.
// To return nil, simply return nil or an empty slice.
//
// To return a single value which itself is a slice, then remember to have that as the first element of the result.
// That is: []any{[]string} to fullfill the promise with a single arg which is an array of String.
func Promise(work func() ([]any, error)) js.Value {
	return js.Global().Get("Promise").New(
		FuncOfOnce(
			func(this js.Value, args []js.Value) any {
				resolve, reject := args[0], args[1]
				go func() {
					results, err := work()
					complete(resolve, reject, err, results)
				}()
				return nil
			},
		),
	)
}

// PromiseResult is a convenience around Promise for functions returning a single result
func PromiseResult(work func() (any, error)) js.Value {
	return Promise(func() ([]any, error) {
		result, err := work()
		return []any{result}, err
	})
}

// SyncPromise returns a promise immediately completing with the supplied arguments or error
func SyncPromise(err error, results ...any) js.Value {
	return js.Global().Get("Promise").New(
		FuncOfOnce(
			func(this js.Value, args []js.Value) any {
				resolve, reject := args[0], args[1]
				complete(resolve, reject, err, results)
				return nil
			},
		),
	)
}

func complete(resolve, reject js.Value, err error, results []any) {
	if err != nil {
		reject.Invoke(Error(err))
	} else {
		if len(results) == 0 {
			resolve.Invoke(nil)
		} else {
			resolve.Invoke(results...)
		}
	}
}
