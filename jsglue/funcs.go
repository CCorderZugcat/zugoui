//go:build js

package jsglue

import "syscall/js"

// Funcs is a simple collection of [js.Func] instances
type Funcs []js.Func

// Release releases all the functions
func (f Funcs) Release() {
	for _, f := range f {
		f.Release()
	}
}

// FuncOf is a covenience calling [js.FuncOf] and adding to f
func (f *Funcs) FuncOf(fn func(js.Value, []js.Value) any) js.Func {
	jsfn := js.FuncOf(fn)
	(*f) = append((*f), jsfn)
	return jsfn
}

// Close calls dtor, and if it does not return an error, proceeds to Release f
func (f Funcs) Close(dtor func() error) error {
	if dtor != nil {
		if err := dtor(); err != nil {
			return err
		}
	}
	f.Release()
	return nil
}

// JsDtor adds a function to f which will release f if dtor does not return an error,
// else returns that error without releasing. It is presumed the returned function
// will also be part of f.
func (f *Funcs) JsDtor(dtor func() error) js.Func {
	return js.FuncOf(
		func(this js.Value, args []js.Value) any {
			if dtor != nil {
				if err := dtor(); err != nil {
					return Error(err)
				}
			}
			f.Release()
			return nil
		},
	)
}

// FuncOfOnce returns a single use function that self releases once called.
func FuncOfOnce(fn func(js.Value, []js.Value) any) js.Func {
	var jsfn js.Func

	jsfn = js.FuncOf(
		func(this js.Value, args []js.Value) any {
			jsfn.Release()
			return fn(this, args)
		},
	)

	return jsfn
}
