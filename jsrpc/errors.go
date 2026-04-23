//go:build js

package jsrpc

import (
	"sync/atomic"
	"syscall/js"

	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/observable"
)

// errorSource object is a react friendly datasource for validation errors
type errorSource struct {
	b        *Browser
	formID   string
	funcs    jsglue.Funcs
	form     *form
	state    atomic.Value
	resolved chan (struct{}) // form is valid if closed
}

func newErrorSource(b *Browser, formID string) *errorSource {
	e := &errorSource{
		b:        b,
		formID:   formID,
		resolved: make(chan struct{}),
	}

	go func() {
		defer close(e.resolved)

		vb, err := b.bindingsFor(formID)
		if err != nil {
			return
		}

		e.form = newForm(vb)
		e.state.Store(js.ValueOf(e.form.errors()))
	}()

	return e
}

func (e *errorSource) JsObject() map[string]any {
	return map[string]any{
		// subscribe(cb: () => void): () => void
		"subscribe": e.funcs.FuncOf(func(_ js.Value, args []js.Value) any {
			if err := jsglue.AssertArgs(args, js.TypeFunction); err != nil {
				return jsglue.Error(err)
			}

			cleanup := e.subscribe(args[0])
			return unsubscriber(cleanup)
		}),

		// getSnapshot(): Record<string, {message: string, type: string}>
		"getSnapshot": e.funcs.FuncOf(func(js.Value, []js.Value) any {
			if e := e.state.Load(); e != nil {
				return e
			}
			return noErrors
		}),

		// release(void): void
		"release": e.funcs.FuncOf(func(js.Value, []js.Value) any {
			e.release()
			return nil
		}),
	}
}

func (e *errorSource) release() {
	e.funcs.Release()
}

func (e *errorSource) subscribe(cb js.Value) func() {
	done := make(chan struct{})

	go func(cb js.Value) {
		var f *form

		select {
		case <-e.resolved:
			f = e.form
		case <-done:
		}

		if f == nil {
			return
		}

		cb.Invoke() // initial state

		o := observable.NewActionObserver(func(string, any) {
			e.state.Store(js.ValueOf(f.errors()))
			cb.Invoke()
		})

		so := observable.NewPathObserver("*", f.source)
		so.AddObserver("", o)
		<-done
		so.Release()
	}(cb)

	return func() {
		close(done)
	}
}
