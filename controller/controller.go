// Package controller manages bindings with a model, and provides mutation methods
package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/wsrpc"
)

var (
	ErrBadPair = errors.New("bad pair")
)

// Controller handles the interaction of observable bindings between the model and the browser.
// It also handles action bindings with clickable objects.
type Controller struct {
	Browser wsrpc.Browser

	server   *wsrpc.Server
	bindings map[string]*binding
	actions  map[string][]int64
	prefix   string // namespace with trailing dot
}

type binding struct {
	handles   []int64 // browser side binding handles
	action    string  // server side binding action (usually our key path)
	observing observable.Observable
}

// New creates a new Controller instance given the web server's RPC service and a connection to the browser.
// namespace specifies the browser's binding namespace to use with this Controller's instance.
func New(server *wsrpc.Server, browser wsrpc.Browser, namespace string) *Controller {
	return &Controller{
		server:   server,
		Browser:  browser,
		bindings: make(map[string]*binding),
		actions:  make(map[string][]int64),
		prefix:   namespace + ".",
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
		for _, h := range v {
			c.Browser.Unbind(h)
		}
	}
	clear(c.actions)
}

// HandleActions sets the callback for actions
func (c *Controller) HandleActions(handler func(action string)) {
	c.server.AddActionObserver(observable.NewActionObserver(func(_ string, value any) {
		fullName, _ := value.(string)

		if name, ok := strings.CutPrefix(fullName, c.prefix); ok {
			handler(name)
		} else if name, ok := strings.CutPrefix(fullName, "global."); ok {
			handler(name)
		}
	}))
}

// BindAction creates an action binding. Call HandleActions first.
func (c *Controller) BindAction(element, action string) error {
	handle, err := c.Browser.NewClickBinding(element, c.prefix+action)
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

// BindValues creates value bindings.
func (c *Controller) BindValues(
	name string,
	formID string,
	elements []string,
	source observable.Source,
) error {
	action := c.prefix + name

	if m, ok := source.(observable.MutableSource); ok {
		// NewWriter keeps the observers above us
		m = observable.NewWriter(m)
		c.server.AddValueObserver(action, m)

		// keep browswer updates from going back to the browser
		source = m
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

	source.AddObserver("", wsrpc.Observer{Browser: c.Browser, Handle: handle})

	return nil
}
