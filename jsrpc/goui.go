//go:build js

package jsrpc

import (
	"errors"
	"fmt"
	"slices"
	"syscall/js"

	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/jstypes"
	"github.com/CCorderZugcat/zugoui/observable"
)

// methods from the Browser server instance exported to window.goui

// JsObject returns the "zugoui" object which usually appears in the window object
func (b *Browser) JsObject() map[string]any {
	return map[string]any{
		"addEventListener": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}
			b.addEventListener(args[0].String(), args[1])
			return nil
		}),
		"removeEventListener": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}
			b.removeEventListener(args[0].String(), args[1])
			return nil
		}),
		"sendAction": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString); err != nil {
				return jsglue.Error(err)
			}
			b.server.Action("global." + args[0].String())
			return nil
		}),
		"form": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString); err != nil {
				return jsglue.Error(err)
			}

			return b.form(args[0].String())
		}),
	}
}

// addEventListener adds an event listener
func (b *Browser) addEventListener(typ string, listener js.Value) {
	b.lck.Lock()
	defer b.lck.Unlock()

	listeners := append(b.listeners[typ], listener)
	b.listeners[typ] = listeners
}

// removeEventListener removes an event listener
func (b *Browser) removeEventListener(typ string, listener js.Value) {
	b.lck.Lock()
	defer b.lck.Unlock()

	listeners := b.listeners[typ]

	if i := slices.IndexFunc(
		listeners,
		func(cb js.Value) bool {
			return cb.Equal(listener)
		},
	); i >= 0 {
		listeners = append(listeners[:i], listeners[i+1:]...)
		b.listeners[typ] = listeners
	}
}

// form allows browser code (e.g. react) to subscribe to fields in a form,
// and get the validation status, if the model has one.
// Caution: do not create multiple active instances of the same form.
// To avoid leaks, call release() on the returned object when done with it.
func (b *Browser) form(formID string) any {
	return jsglue.PromiseResult(func() (any, error) {
		for {
			v, ok := b.formIDs.Load(formID)
			if !ok {
				_, ok := <-b.formAdded
				if !ok {
					return nil, errors.New("closing")
				}
				continue
			}

			vb := v.(*valueBindings) // let this interface assertion panic
			return newForm(vb.source).JsObject(), nil
		}
	})
}

// form is an instance of a form
type form struct {
	source observable.Source
	funcs  jsglue.Funcs
	fields map[string][]string // key to id
	keys   map[string]string   // id to key
}

func newForm(source observable.Source) *form {
	f := &form{
		source: source,
		fields: make(map[string][]string),
		keys:   make(map[string]string),
	}

	for _, key := range source.Keys() {
		for _, tag := range source.Tag(key, "bind") {
			id, _ := idAndProperty(tag)
			f.fields[key] = append(f.fields[key], id)
			f.keys[id] = key
		}
	}

	return f
}

// unsubscriber returns an unsubscribe function to the js caller
func unsubscriber(cleanup func()) js.Func {
	var unsubscribe js.Func
	unsubscribe = js.FuncOf(func(js.Value, []js.Value) any {
		cleanup()
		unsubscribe.Release()
		return nil
	})
	return unsubscribe
}

func (f *form) JsObject() map[string]any {
	return map[string]any{
		"errors": f.funcs.FuncOf(func(js.Value, []js.Value) any {
			return f.errors()
		}),

		"subscribeErrors": f.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}

			observer := f.newErrorSubscription(args[0])
			return unsubscriber(func() {
				f.source.RemoveObserver("", observer)
			})
		}),

		"getValue": f.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString); err != nil {
				return jsglue.Error(err)
			}

			id := args[0].String()
			value, _ := jstypes.ValueOf(f.source.Value(f.keys[id]))

			return value
		}),

		"subscribe": f.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}

			observer := f.newSubscription("", args[0])
			return unsubscriber(func() {
				f.source.RemoveObserver("", observer)
			})

		}),

		"getSnapshot": f.funcs.FuncOf(func(js.Value, []js.Value) any {
			snapshot := make(map[string]any)

			for _, key := range f.source.Keys() {
				for _, id := range f.fields[key] {
					snapshot[id], _ = jstypes.ValueOf(f.source.Value(key))
				}
			}

			return snapshot
		}),

		"subscribeKey": f.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}

			id := args[0].String()
			key, ok := f.keys[id]
			if !ok {
				return jsglue.Error(fmt.Errorf("key %s not found", id))
			}

			observer := f.newSubscription(id, args[1])
			return unsubscriber(func() {
				f.source.RemoveObserver(key, observer)
			})
		}),

		"release": f.funcs.FuncOf(func(js.Value, []js.Value) any {
			f.funcs.Release()
			return nil
		}),
	}
}

// errors returns an object of errors by field id,
// if there is indeed a Source found for the given form.
func (f *form) errors() any {
	err := observable.ValidateSource(f.source)
	errors, ok := err.(observable.ValidationError)
	if !ok {
		return nil
	}

	result := make(map[string]any)

	for key, err := range errors {
		for _, id := range f.fields[key] {
			result[id] = err.Error()
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// newSubscription returns a subscription to the whole form or a field in it
func (f *form) newSubscription(id string, callback js.Value) *subscription {
	s := &subscription{form: f, callback: callback, id: id}
	f.source.AddObserver("", s)
	return s
}

// newErrorSubscription returns a subscription to form errors
func (f *form) newErrorSubscription(callback js.Value) *errorSubscription {
	e := &errorSubscription{form: f, callback: callback}
	f.source.AddObserver("", e)
	return e
}

type subscription struct {
	observable.NullObserver
	form     *form
	id       string
	callback js.Value
}

func (s *subscription) SetValue(key string, value any) {
	for _, id := range s.form.fields[key] {
		if id == "" || id == s.id {
			value, _ := jstypes.ValueOf(value)
			s.callback.Invoke(value)
			return
		}
	}
}

type errorSubscription struct {
	observable.NullObserver
	form     *form
	callback js.Value
}

func (e *errorSubscription) SetValue(key string, value any) {
	if errors := e.form.errors(); errors != nil {
		e.callback.Invoke(errors)
	} else {
		e.callback.Invoke(map[string]any{})
	}
}
