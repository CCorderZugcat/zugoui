//go:build js

package input

import (
	"errors"
	"reflect"
	"sync"
	"syscall/js"
	"time"

	"github.com/CCorderZugcat/zugoui/jsglue"
	"github.com/CCorderZugcat/zugoui/jstypes"
	"github.com/CCorderZugcat/zugoui/observable"
)

var ErrInvalidType = errors.New("invalid type")

// Input is a bindable proxy to an <input> or <fieldset>.
// <fieldset> is assumed to contain radio buttons of the same name.
type Input struct {
	lck sync.RWMutex
	observable.NullObserver
	observable.NullSource
	input, change js.Func // function for addEventListener on elem
	listening     bool
	elem          js.Value                // anchoring element
	properties    map[string]reflect.Type // properties we are interested in
	o             *observable.Observe
	inputCh       chan struct{}
}

var _ observable.Observable = &Input{}

// NewInput creates a new Input instance.
func NewInput(elem js.Value) *Input {
	i := &Input{
		elem:       elem,
		properties: make(map[string]reflect.Type),
		o:          observable.New(),
		inputCh:    make(chan struct{}, 1),
	}
	i.input = js.FuncOf(i.inputHandler)
	i.change = js.FuncOf(i.changeHandler)

	return i
}

// Release releases this Input.
// The event listener is removed from the element, and the js.Function instance is released.
func (i *Input) Release() {
	close(i.inputCh)

	i.o.Release()
	if i.listening {
		i.elem.Call("removeEventListener", "change", i.change)
		i.elem.Call("removeEventListener", "input", i.input)
	}
	i.input.Release()
	i.change.Release()
}

func (i *Input) changeHandler(_ js.Value, _ []js.Value) any {
	i.lck.RLock()
	defer i.lck.RUnlock()

	for key := range i.properties {
		i.o.SetValue(key, i.Value(key))
	}
	return nil
}

func (i *Input) inputHandler(_ js.Value, _ []js.Value) any {
	select {
	case i.inputCh <- struct{}{}:
	default: // do not block
	}

	return nil
}

func (i *Input) SetValue(key string, value any) {
	if value != nil {
		i.lck.Lock()
		i.properties[key] = reflect.ValueOf(value).Type()
		i.lck.Unlock()
	}

	if v, ok := jstypes.ValueOf(value); ok {
		Set(i.elem, key, js.ValueOf(v))
	}
}

func (i *Input) Value(key string) any {
	i.lck.RLock()
	defer i.lck.RUnlock()

	typ, ok := i.properties[key]
	if !ok {
		typ = reflect.TypeFor[string]()
	}

	v := reflect.New(typ).Elem()
	if !jsglue.SetValue(v, Value(i.elem)) {
		return nil
	}

	return v.Interface()
}

func (i *Input) Updating() func() {
	return i.o.Updating()
}

func (i *Input) AddObserver(key string, observer observable.Observer) {
	i.lck.Lock()
	defer i.lck.Unlock()

	pkey := key
	if pkey == "" {
		pkey = "value"
	}
	if _, ok := i.properties[pkey]; !ok {
		i.properties[pkey] = reflect.TypeFor[string]()
	}

	if !i.listening {
		i.listening = true

		go func() {
			for range i.inputCh {
				i.changeHandler(js.ValueOf(nil), nil)
				time.Sleep(time.Second)
			}
		}()

		i.elem.Call("addEventListener", "change", i.change)
		i.elem.Call("addEventListener", "input", i.input)
	}

	i.o.AddObserver(key, observer)
}

func (i *Input) RemoveObserver(key string, observer observable.Observer) {
	i.o.RemoveObserver(key, observer)
}
