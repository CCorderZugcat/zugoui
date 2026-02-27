//go:build js

package jsrpc

import (
	"errors"
	"slices"
	"syscall/js"

	"github.com/CCorderZugcat/zugoui/jsglue"
)

var ErrClosing = errors.New("closing")

var noErrors = js.ValueOf(map[string]any{}) // same instance to return for no errors

// methods from the Browser server instance exported to window.goui

// JsObject returns the "zugoui" object which usually appears in the window object
func (b *Browser) JsObject() map[string]any {
	return map[string]any{
		// addEventListener(type: string, listener: (ev)=>void): void
		"addEventListener": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}
			b.addEventListener(args[0].String(), args[1])
			return nil
		}),

		// removeEventListener(type: string, listener: (ev)=>void): void
		"removeEventListener": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}
			b.removeEventListener(args[0].String(), args[1])
			return nil
		}),

		// sendAction(action: string): void
		"sendAction": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString); err != nil {
				return jsglue.Error(err)
			}
			b.server.Action("global." + args[0].String())
			return nil
		}),

		// form(name: string): Promise<Form>
		"form": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString); err != nil {
				return jsglue.Error(err)
			}

			return b.form(args[0].String())
		}),

		// convenience react friendly datasource for field errors
		// errors(formName: string): { subscribe: (onEventChange: () => void) => void, getSnapshot: () => Record<string, any> }
		"errors": b.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeString); err != nil {
				return jsglue.Error(err)
			}
			return newErrorSource(b, args[0].String()).JsObject()
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

func (b *Browser) bindingsFor(formID string) (*valueBindings, error) {
	for {
		v, ok := b.formIDs.Load(formID)
		if !ok {
			if _, ok = <-b.formAdded; !ok {
				return nil, ErrClosing
			}
			continue
		}
		return v.(*valueBindings), nil
	}
}

// form allows browser code (e.g. react) to subscribe to fields in a form,
// and get the validation status, if the model has one.
// To avoid leaks, call release() on the returned object when done with it.
func (b *Browser) form(formID string) any {
	return jsglue.PromiseResult(func() (any, error) {
		vb, err := b.bindingsFor(formID)
		if err != nil {
			return nil, err
		}
		return newForm(vb.source).JsObject(), nil
	})
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
