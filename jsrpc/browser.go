//go:build js

package jsrpc

import (
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
	server    *Server // to the web server
	handles   sync.Map
	formIDs   sync.Map
	listeners map[string][]js.Value
	lck       sync.RWMutex
	funcs     jsglue.Funcs
	formAdded chan struct{}
}

type bindingSet map[string][]*input.ValueBinding

type valueBindings struct {
	source, up observable.MutableSource
	elementIDs []string
	property   string
	formID     string
	bindings   bindingSet
}

func (vb *valueBindings) Destroy() {
	for _, v := range vb.bindings {
		for _, b := range v {
			b.Destroy()
		}
	}

	vb.source.RemoveAllObservers()
	vb.up.RemoveAllObservers()
}

func (vb *valueBindings) Rebind() any {
	vcopy := *vb

	for _, v := range vcopy.bindings {
		for _, b := range v {
			b.Destroy()
		}
	}

	bindings := make(bindingSet)

	if len(vcopy.elementIDs) > 0 {
		for _, id := range vcopy.elementIDs {
			// bind single element to single value (no form logic)
			elem, err := input.Element(id)
			if err != nil {
				// dynamic pages can race, which is why we provide a rebind for these cases
				continue
			}
			binding, err := input.NewValueBinding(elem, vcopy.property, vcopy.source, "value")
			bindings["value"] = append(bindings["value"], binding)
		}
	} else {
		// bind structure with tagged fields
		for _, key := range vcopy.source.Keys() {
			elems := vcopy.source.Tag(key, "bind")
			for _, tag := range elems {
				id, property := idAndProperty(tag)

				elem, err := input.Element(id)
				if err != nil {
					continue
				}

				if vcopy.formID == "" {
					// opportunistically try to get the enclosing form
					form := elem.Call("closest", "form")
					if form.Type() == js.TypeObject {
						if id := form.Get("id"); id.Type() == js.TypeString {
							vcopy.formID = id.String()
						}
					}
				}

				binding, err := input.NewValueBinding(elem, property, vcopy.source, key)
				bindings[key] = append(bindings[key], binding)
			}
		}
	}

	return &vcopy
}

// perform the same action on each binding
func (vb *valueBindings) eachBindingFor(key string, fn func(observable.Observer)) {
	fn(vb.up)
	for it := range slices.Values(vb.bindings[key]) {
		fn(it)
	}
}

var nextID atomic.Int64

// New creates a new browser side rpc service instance
func New(server *Server) *Browser {
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

	// create client side data source
	model.Set(reflect.ValueOf(req.Model))
	source := observable.NewModelValue(model)

	// this observer sends user changes from browser to the server
	up := observable.NewWriter(source)
	observer := Observer{b.server, req.Action}
	up.AddObserver("", observer)

	property := req.Property
	if property == "" {
		property = "value"
	}

	vb := (&valueBindings{
		source:     source,
		up:         up,
		elementIDs: slices.Clone(req.ElementIDs),
		property:   property,
	}).Rebind()

	res.Handle = handle
	b.storeBindings(handle, vb.(*valueBindings))

	return nil
}

func (b *Browser) storeBindings(handle int64, bindings any) {
	switch bindings := bindings.(type) {
	case *valueBindings:
		formID := bindings.formID
		b.handles.Store(handle, bindings)

		if formID != "" {
			b.formIDs.Store(formID, bindings)
			select {
			case b.formAdded <- struct{}{}:
			default: // do not block
			}
		}
	default:
		b.handles.Store(handle, bindings)
	}
}

// Rebind allows the client to request the server bindings to be rebound.
// This is needed for dynamic pages after creating or re-creating elements.
func (b *Browser) Rebind() {
	b.formIDs.Clear()
	b.handles.Range(func(key, value any) bool {
		if rb, ok := value.(interface{ Rebind() any }); ok {
			b.storeBindings(key.(int64), rb.Rebind())
		}

		return true
	})
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
	handle := nextID.Add(1)

	clickBinding := input.NewClickBinding(req.ElementID, req.Action, func(action string) {
		b.server.Action(action)
	})
	res.Handle = handle

	b.storeBindings(handle, clickBinding)
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
