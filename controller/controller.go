// Package controller manages bindings with a model, and provides mutation methods
package controller

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/wsrpc"
)

var (
	ErrBadPair = errors.New("bad pair")
)

// Controller handles the interaction of observable bindings between the model and the browser.
// It also handles action bindings with clickable objects.
type Controller struct {
	Browswer wsrpc.Browser

	lck         sync.RWMutex
	server      *wsrpc.Server
	bindings    map[string]*binding
	actions     map[string][]int64
	controllers map[string]*Controller
	prefix      string // namespace with trailing dot
}

type binding struct {
	handles   []int64 // browser side binding handles
	action    string  // server side binding action (usually our key path)
	observing observable.Observable
}

// actionObserver is a function adapter to an [observable.Observer].
// actions only ever set the key "action" with the action name as the value.
type actionObserver func(action string)

var _ observable.Observer = actionObserver(nil)

func (a actionObserver) SetValue(key string, value any) {
	if key == "action" {
		a(value.(string))
	}
}

func (a actionObserver) InsertValueAt(int, any)  {}
func (a actionObserver) RemoveValueAt(int)       {}
func (a actionObserver) SetValueAt(int, any)     {}
func (a actionObserver) SetValueFor(string, any) {}
func (a actionObserver) RemoveValueFor(string)   {}

// New creates a new Controller instance given the web server's RPC service and a connection to the browser.
// namespace specifies the browser's binding namespace to use with this Controller's instance.
func New(server *wsrpc.Server, browser wsrpc.Browser, namespace string) *Controller {
	return &Controller{
		server:      server,
		Browswer:    browser,
		bindings:    make(map[string]*binding),
		actions:     make(map[string][]int64),
		controllers: make(map[string]*Controller),
		prefix:      namespace + ".",
	}
}

// Release removes all bindings
func (c *Controller) Release() {
	for k, v := range c.bindings {
		c.server.RemoveValueObservers(k)
		v.observing.RemoveAllObservers()
		for _, h := range v.handles {
			c.Browswer.Unbind(h)
		}
	}
	clear(c.bindings)

	c.server.RemoveActionObservers()
	for _, v := range c.actions {
		for _, h := range v {
			c.Browswer.Unbind(h)
		}
	}
	clear(c.actions)

	for _, v := range c.controllers {
		v.Release()
	}
}

// HandleActions sets the callback for actions
func (c *Controller) HandleActions(handler func(action string)) {
	c.server.AddActionObserver(actionObserver(func(fullName string) {
		if strings.HasPrefix(fullName, c.prefix) {
			handler(strings.Trim(fullName, c.prefix))
		} else if strings.HasPrefix(fullName, "global.") {
			handler(strings.Trim(fullName, "global."))
		}
	}))
}

// BindAction creates an action binding. Call HandleActions first.
func (c *Controller) BindAction(element, action string) error {
	c.lck.Lock()
	defer c.lck.Unlock()

	handle, err := c.Browswer.NewClickBinding(element, c.prefix+action)
	if err != nil {
		return err
	}

	c.actions[action] = append(c.actions[action], handle)
	return nil
}

// BindActions calls BindAction with multiple pairs of element,action as a convenience
func (c *Controller) BindActions(pairs ...string) error {
	for i := 0; i < len(pairs); i += 2 {
		if (i + 1) == len(pairs) {
			return fmt.Errorf("%w: variadic in groups of element, action pairs", ErrBadPair)
		}
		if err := c.BindAction(pairs[i], pairs[i+1]); err != nil {
			return err
		}
	}
	return nil
}

// BindValue creates an arbitrary value binding.
// If o is observing a struct with bind tags, elements may be nil or empty.
// To recursively bind all structures with bind tags, use BindModel isntead.
func (c *Controller) BindValue(
	name string,
	elements []string,
	property string,
	source observable.Source,
) error {
	c.lck.Lock()
	defer c.lck.Unlock()

	action := c.prefix + name

	if m, ok := source.(observable.MutableSource); ok {
		// NewWriter keeps the observers above us
		m = observable.NewWriter(m)
		c.server.AddValueObserver(action, m)

		// keep browswer updates from going back to the browser
		source = m
	}

	handle, err := c.Browswer.NewValueBinding(
		action,
		elements,
		property,
		source.Model().Interface(),
	)
	if err != nil {
		return err
	}

	b, ok := c.bindings[action]
	if ok {
		b.handles = append(b.handles, handle)
	} else {
		b = &binding{
			handles:   []int64{handle},
			action:    action,
			observing: source,
		}
		c.bindings[action] = b
	}

	source.AddObserver("", wsrpc.Observer{Browser: c.Browswer, Handle: handle})

	return nil
}

// BindModel recursively binds a structure with bind tags.
func (c *Controller) BindModel(name string, m observable.Source) error {
	if err := c.BindValue(name, nil, "value", m); err != nil {
		return err
	}
	if m.Model().Kind() == reflect.Struct {
		for _, key := range m.Keys() {
			elem := m.Value(key)

			source, ok := elem.(observable.MutableSource)
			if !ok {
				value := observable.MutableValue(reflect.ValueOf(elem))
				if !value.IsValid() {
					continue
				}
				if value.Kind() == reflect.Struct {
					source = observable.NewModelValue(value)
				}
			}

			if err := c.BindModel(name+"."+key, source); err != nil {
				return err
			}
		}
	}
	return nil
}
