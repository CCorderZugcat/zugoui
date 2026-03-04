//go:build js

package jsrpc

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall/js"

	"github.com/CCorderZugcat/zugoui/input"
	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/wsrpc"
	"github.com/CCorderZugcat/zugoui/wsrpc/rpctypes"
)

// methods called to the client from the server

// Browser is the browser side rpc service
type Browser struct {
	handles   sync.Map
	formIDs   sync.Map
	server    Server // to the web server
	listeners map[string][]js.Value
	lck       sync.RWMutex
	funcs     jsglue.Funcs
	formAdded chan struct{}
}

type bindingSet map[string][]*input.ValueBinding

type valueBindings struct {
	source, up observable.MutableSource
	formID     string
	bindings   bindingSet
}

func (v *valueBindings) Destroy() {
	for _, v := range v.bindings {
		for _, b := range v {
			b.Destroy()
		}
	}

	v.source.RemoveAllObservers()
	v.up.RemoveAllObservers()
}

// perform the same action on each binding
func (v *valueBindings) eachBindingFor(key string, fn func(observable.Observer)) {
	fn(v.up)
	for it := range slices.Values(v.bindings[key]) {
		fn(it)
	}
}

var nextID atomic.Int64

// New creates a new browser side rpc service instance
func New(server Server) *Browser {
	b := &Browser{
		server:    server,
		listeners: make(map[string][]js.Value),
		formAdded: make(chan struct{}, 1), // channel size of 1 presumes one thread in host js
	}

	return b
}

// Destroy removes all bindings
func (b *Browser) Destroy() {
	close(b.formAdded)
	b.handles.Range(func(_ any, v any) bool {
		if v != nil {
			v.(interface{ Destroy() }).Destroy()
		}
		return true
	})
	b.funcs.Release()
}

func idAndProperty(tag string) (id, property string) {
	n := strings.IndexRune(tag, '>')
	if n < 0 {
		return tag, "value"
	}
	return tag[:n], tag[n+1:]
}

// DispatchEvent causes and event to be sent to all listeners in the browser
func (b *Browser) DispatchEvent(req *rpctypes.DispatchEventReq, _ *bool) error {
	ev := jsglue.NewCustomEvent(req.Type, req.Detail)

	b.lck.RLock()
	cbs := slices.Clone(b.listeners[req.Type])
	b.lck.RUnlock()

	for _, cb := range cbs {
		cb.Invoke(ev)
	}
	return nil
}

// NewValueBinding creates new value bindings.
// A single binding may be made if ElementID is specified.
// The change callback is proxied to [server.UpdateValue].
// If binding a model to a collection of controls in a form,
// keep them in the same form element for validation to work properly, if using validation.
func (b *Browser) NewValueBinding(req *rpctypes.NewValueBindingReq, res *rpctypes.NewValueBindingRes) error {
	// rpc will hand us an un-settable reference, store it in our own pointer
	modelValue := reflect.ValueOf(req.Model)
	modelStorage := reflect.New(modelValue.Type())
	modelStorage.Elem().Set(modelValue)
	model := reflect.New(reflect.ValueOf(req.Model).Type()).Elem()

	handle := nextID.Add(1)
	bindings := make(bindingSet) // key:[]binding (each ui element of same key)

	// create client side data source
	model.Set(reflect.ValueOf(req.Model))
	source := observable.NewModelValue(model)

	// this observer sends user changes from browser to the server
	up := observable.NewWriter(source)
	observer := Observer{b.server, req.Action}
	up.AddObserver("", observer)

	var formID string

	if len(req.ElementIDs) > 0 {
		for _, id := range req.ElementIDs {
			// bind single element to single value (no form logic)
			elem, err := input.Element(id)
			if err != nil {
				fmt.Printf("warning: could not find element %s\n", id)
				continue
			}
			binding, err := input.NewValueBinding(elem, req.Property, source, "value")
			bindings["value"] = append(bindings["value"], binding)
		}
	} else {
		// bind structure with tagged fields
		for _, key := range source.Keys() {
			elems := source.Tag(key, "bind")
			for _, tag := range elems {
				id, property := idAndProperty(tag)

				elem, err := input.Element(id)
				if err != nil {
					fmt.Printf("warning: could not find element %s\n", id)
					continue
				}

				if formID == "" {
					// opportunistically try to get the enclosing form
					form := elem.Call("closest", "form")
					if form.Type() == js.TypeObject {
						if id := form.Get("id"); id.Type() == js.TypeString {
							formID = id.String()
						}
					}
				}

				binding, err := input.NewValueBinding(elem, property, source, key)
				bindings[key] = append(bindings[key], binding)
			}
		}
	}

	res.Handle = handle

	// this association allows the server to update us, the browser
	v := &valueBindings{source: source, up: up, formID: formID, bindings: bindings}
	b.handles.Store(handle, v)

	if formID != "" {
		// this presumes a single threaded js engine, otherwise there is a potential
		// for a race condition between other go routine not seeing the form,
		// us storing the form, then seeing nobody waiting on the channel, then other
		// go routine waiting on the channel.
		b.formIDs.Store(formID, v)

		select {
		case b.formAdded <- struct{}{}:
		default: // do not block
		}
	}

	return nil
}

// Unbind releases a binding
func (b *Browser) Unbind(req *rpctypes.UnbindReq, _ *bool) error {
	h, ok := b.handles.LoadAndDelete(req.Handle)
	if !ok || h == nil {
		return wsrpc.ErrInvalidHandle
	}

	switch h := h.(type) {
	case *valueBindings:
		if h.formID != "" {
			b.formIDs.CompareAndDelete(h.formID, h)
		}
		h.Destroy()
	case interface{ Destroy() }:
		h.Destroy()
	default:
		panic("impossible handle stored. All handles must have Destroy()")
	}

	return nil
}

// resolveHandle finds a handle by ID and of type T,
// returning an appropriate error if neither are met.
func resolveHandle[T any](b *Browser, handle int64) (result T, err error) {
	h, ok := b.handles.Load(handle)
	if !ok {
		// handle not found
		return result, wsrpc.ErrInvalidHandle
	}

	result, ok = h.(T)
	if !ok {
		// handle to something else
		return result, wsrpc.ErrInvalidHandle
	}

	return result, nil
}

// eachBindingFor execute the same function for every binding associated with handles for the key.
// returns and appopriate error if the handle is not found.
// does nothing successfully if the key is not found.
func (b *Browser) eachBindingFor(handle int64, key string, fn func(observable.Observer)) error {
	bindings, err := resolveHandle[*valueBindings](b, handle)
	if err != nil {
		return err
	}

	bindings.eachBindingFor(key, fn)
	return nil
}

// NewClickBinding returns a new input binding.
// This is an rpc version of [input.NewClickBinding].
// The action is proxied to [server.Action].
func (b *Browser) NewClickBinding(req *rpctypes.NewClickBindingReq, res *rpctypes.NewClickBindingRes) error {
	elem, err := input.Element(req.ElementID)
	if err != nil {
		return err
	}

	handle := nextID.Add(1)

	clickBinding := input.NewClickBinding(elem, req.Action, func(action string) {
		b.server.Action(action)
	})
	res.Handle = handle

	b.handles.Store(handle, clickBinding)
	return nil
}

// Mutable rpc handlers

func (b *Browser) SetValue(req *rpctypes.SetValueReq, _ *bool) error {
	return b.eachBindingFor(
		req.Handle,
		req.Key,
		func(b observable.Observer) { b.SetValue(req.Key, req.Value) },
	)
}

func (b *Browser) InsertValueAt(req *rpctypes.InsertValueAtReq, _ *bool) error {
	return b.eachBindingFor(
		req.Handle,
		"value",
		func(b observable.Observer) { b.InsertValueAt(req.At, req.Value) },
	)
}

func (b *Browser) RemoveValueAt(req *rpctypes.RemoveValueAtReq, _ *bool) error {
	return b.eachBindingFor(
		req.Handle,
		"value",
		func(b observable.Observer) { b.RemoveValueAt(req.At) },
	)
}

func (b *Browser) SetValueAt(req *rpctypes.SetValueAtReq, _ *bool) error {
	return b.eachBindingFor(
		req.Handle,
		"value",
		func(b observable.Observer) { b.SetValueAt(req.At, req.Value) },
	)
}

func (b *Browser) SetValueFor(req *rpctypes.SetValueForReq, _ *bool) error {
	return b.eachBindingFor(
		req.Handle,
		req.Key,
		func(b observable.Observer) { b.SetValueFor(req.Key, req.Value) },
	)
}

func (b *Browser) RemoveValueFor(req *rpctypes.RemoveValueForReq, _ *bool) error {
	return b.eachBindingFor(
		req.Handle,
		req.Key,
		func(b observable.Observer) { b.RemoveValueFor(req.Key) },
	)
}
