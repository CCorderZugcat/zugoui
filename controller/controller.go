// Package controller manages bindings with a model, and provides mutation methods
package controller

import (
	"strings"
	"sync"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/wsrpc"
)

// Controller handles the interaction of observable bindings between the model and the browser.
// It also handles action bindings with clickable objects.
type Controller struct {
	Browser wsrpc.Browser

	lck      sync.RWMutex
	server   *wsrpc.Server
	bindings map[string]*binding
	actions  map[string]*actionBinding
	prefix   string // namespace with trailing dot
}

type binding struct {
	handles   []int64 // browser side binding handles
	action    string  // server side binding action (usually our key path)
	observing observable.Observable
}

type actionBinding struct {
	handle int64
	target func(string)
}

// New creates a new Controller instance given the web server's RPC service and a connection to the browser.
// namespace specifies the browser's binding namespace to use with this Controller's instance.
func New(server *wsrpc.Server, browser wsrpc.Browser, namespace string) *Controller {
	c := &Controller{
		server:   server,
		Browser:  browser,
		bindings: make(map[string]*binding),
		actions:  make(map[string]*actionBinding),
		prefix:   namespace + ".",
	}
	c.server.AddActionObserver(observable.NewActionObserver(func(_ string, value any) {
		name, ok := value.(string)
		if !ok {
			return
		}
		c.action(name)
	}))

	return c
}

func (c *Controller) action(action string) {
	name, ok := strings.CutPrefix(action, c.prefix)
	if !ok {
		name, ok = strings.CutPrefix(action, "global.")
	}
	if !ok {
		return
	}

	c.lck.RLock()
	b, ok := c.actions[name]
	c.lck.RUnlock()

	if ok {
		b.target(name)
	}
}

// Release removes all bindings
func (c *Controller) Release() {
	for k, v := range c.bindings {
		c.server.RemoveValueObservers(k)
		v.observing.Release()
		for _, h := range v.handles {
			c.Browser.Unbind(h)
		}
	}
	clear(c.bindings)

	c.server.ReleaseActionObservers()
	for _, v := range c.actions {
		c.Browser.Unbind(v.handle)
	}
	clear(c.actions)
}

func (c *Controller) BindAction(element, action string, target func(string)) (err error) {
	c.lck.Lock()
	defer c.lck.Unlock()

	b, ok := c.actions[action]
	if !ok {
		b = &actionBinding{}
	} else {
		c.Browser.Unbind(b.handle)
		delete(c.actions, action)
	}

	b.handle, err = c.Browser.NewClickBinding(element, c.prefix+action)
	if err != nil {
		return err
	}
	b.target = target

	c.actions[action] = b
	return nil
}

// BindValues creates value bindings.
func (c *Controller) BindValues(
	name string,
	formID string,
	elements []string,
	source observable.Source,
) error {
	c.lck.Lock()
	defer c.lck.Unlock()

	action := c.prefix + name
	o := observable.NewPathObserver("*", source)

	if _, mutable := source.(observable.MutableSource); mutable {
		c.server.AddValueObserver(action, o)
	}

	handle, err := c.Browser.NewValueBinding(
		action,
		formID,
		elements,
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

	o.AddObserver("", wsrpc.Observer{Browser: c.Browser, Handle: handle})
	return nil
}
