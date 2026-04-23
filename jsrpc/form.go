//go:build js

package jsrpc

import (
	"fmt"
	"maps"
	"syscall/js"

	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/jstypes"
	"github.com/CCorderZugcat/zugoui/observable"
)

// form is an instance of a form
type form struct {
	source observable.Source
	funcs  jsglue.Funcs
	fields map[string][]string // key to id
	keys   map[string]string   // id to key
}

func newForm(vb *valueBindings) *form {
	f := &form{
		source: vb.source,
		fields: make(map[string][]string),
		keys:   maps.Clone(vb.bindings.IDPaths),
	}

	for k, v := range f.keys {
		f.fields[v] = append(f.fields[v], k)
	}

	return f
}

func (f *form) JsObject() map[string]any {
	return map[string]any{
		// errors(void) => Record<string,{message: string, type: string}>
		"errors": f.funcs.FuncOf(func(js.Value, []js.Value) any {
			return f.errors()
		}),

		// subscribeErrors(cb: (Record<string, {message: string, type: string}>) => void): () => void
		"subscribeErrors": f.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}

			cb := args[0]
			cb.Invoke(f.errors())

			observer := f.newErrorSubscription(cb)
			return unsubscriber(observer.Release)
		}),

		// getValue(key: string): any
		"getValue": f.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString); err != nil {
				return jsglue.Error(err)
			}

			id := args[0].String()
			value, _ := jstypes.ValueOf(f.source.Value(f.keys[id]))

			return value
		}),

		// subscribe(cb: (Record<string, any>) => void): () => void
		"subscribe": f.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}

			observer := f.newSubscription("", "", args[0])
			return unsubscriber(func() {
				f.source.RemoveObserver("", observer)
			})

		}),

		// getSnapshot(): Record<string, any>
		"getSnapshot": f.funcs.FuncOf(func(js.Value, []js.Value) any {
			snapshot := make(map[string]any)

			for k, v := range f.keys {
				snapshot[k], _ = jstypes.ValueOf(f.source.Value(v))
			}

			return snapshot
		}),

		// subscribeKey(key: string, cb: (any) => void): () => void
		"subscribeKey": f.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}

			id := args[0].String()
			key, ok := f.keys[id]
			if !ok {
				return jsglue.Error(fmt.Errorf("key %s not found", id))
			}

			observer := f.newSubscription(id, key, args[1])
			return unsubscriber(observer.Release)
		}),

		// release(void): void
		"release": f.funcs.FuncOf(func(js.Value, []js.Value) any {
			f.release()
			return nil
		}),
	}
}

func (f *form) release() {
	f.funcs.Release()
}

// errors returns an object of errors by field id,
// if there is indeed a Source found for the given form.
func (f *form) errors() any {
	err := observable.ValidateSource(f.source)
	errors, ok := err.(observable.ValidationError)
	if !ok {
		return noErrors
	}

	result := make(map[string]any)

	for key, err := range errors {
		for _, id := range f.fields[key] {
			result[id] = map[string]any{
				"message": err.Error(),
				"type":    "server",
			}
		}
	}

	if len(result) == 0 {
		return noErrors
	}

	return result
}

// newSubscription returns a subscription to the whole form or a field in it
func (f *form) newSubscription(id, key string, callback js.Value) *subscription {
	s := &subscription{form: f, callback: callback, id: id}
	if key == "" {
		key = "*"
	}
	s.source = observable.NewPathObserver(key, f.source)
	s.source.AddObserver("", s)
	return s
}

// newErrorSubscription returns a subscription to form errors
func (f *form) newErrorSubscription(callback js.Value) *errorSubscription {
	e := &errorSubscription{form: f, callback: callback}
	e.source = observable.NewPathObserver("*", f.source)
	e.source.AddObserver("", e)
	return e
}

type subscription struct {
	observable.NullObserver
	form     *form
	id       string
	callback js.Value
	source   *observable.PathObserver
}

func (s *subscription) SetValue(key string, value any) {
	s.callback.Invoke()
}

func (s *subscription) Release() {
	s.source.Release()
}

type errorSubscription struct {
	observable.NullObserver
	form     *form
	callback js.Value
	source   *observable.PathObserver
}

func (e *errorSubscription) SetValue(key string, value any) {
	e.callback.Invoke(e.form.errors())
}

func (e *errorSubscription) Release() {
	e.source.Release()
}
