// Package observer implements an observer pattern
package observable

import (
	"slices"
	"sync"
)

type Observer interface {
	SetValue(key string, value any)     // observes a SetValue mutation
	InsertValueAt(index int, value any) // observes an InsertValueAt mutation
	RemoveValueAt(index int)            // observes a RemoveValueAt mutation
	SetValueAt(index int, value any)    // observes a SetValueAt mutation
	SetValueFor(key string, value any)  // observes at SetValueFor mutation
	RemoveValueFor(key string)          // observes a RemoveValueFor mutation
}

type Observable interface {
	Updating() (done func())                      // do not observe changes on this object until done is called. Can be recursive.
	AddObserver(key string, observer Observer)    // adds observer for key. If key is empty, observes all keys.
	RemoveObserver(key string, observer Observer) // removes observer for key. If key is empty, only removes the all keys observer.
	Release()                                     // removes all observers, do not use again after calling.
}

// Observe allows management of observer hooks
type Observe struct {
	lck       sync.RWMutex
	observers map[string][]Observer
	updating  int
}

var _ Observer = (*Observe)(nil)

// New returns a new Observe instance
func New() *Observe {
	o := &Observe{
		observers: make(map[string][]Observer),
	}
	return o
}

// Updating indicates this key is being updated without notification until done is called.
func (o *Observe) Updating() (done func()) {
	o.lck.Lock()
	defer o.lck.Unlock()

	o.updating++

	return func() {
		o.lck.Lock()
		defer o.lck.Unlock()

		o.updating--
	}
}

// AddObserver adds an observer for a given key.
// If key is empty, then all keys are observed.
func (o *Observe) AddObserver(key string, observer Observer) {
	o.lck.Lock()
	defer o.lck.Unlock()

	observers := o.observers[key]
	observers = append(observers, observer)
	o.observers[key] = observers
}

// RemoveObservers removes an observer for a given key.
// If key is empty, only observers observing all keys are removed,
// not observers observing specifc keys.
func (o *Observe) RemoveObserver(key string, observer Observer) {
	o.lck.Lock()
	defer o.lck.Unlock()

	o.observers[key] = slices.DeleteFunc(o.observers[key], func(ko Observer) bool { return ko == observer })
}

// RemoveAllObservers removes all observers.
func (o *Observe) Release() {
	o.lck.Lock()
	defer o.lck.Unlock()

	clear(o.observers)
}

func (o *Observe) observersFor(key string) []Observer {
	o.lck.RLock()
	defer o.lck.RUnlock()

	if o.updating > 0 {
		return nil
	}

	observers := slices.Clone(o.observers[key])
	observers = append(observers, o.observers[""]...)
	return observers
}

func eachObserver(observers []Observer, fn func(Observer)) {
	for i := range slices.Values(observers) {
		fn(i)
	}
}

func (o *Observe) eachObserverFor(key string, fn func(Observer)) {
	eachObserver(o.observersFor(key), fn)
}

// SetValue observes a SetValue mutation.
func (o *Observe) SetValue(key string, value any) {
	o.eachObserverFor(key, func(o Observer) { o.SetValue(key, value) })
}

// InsertValueAt observes an InsertValueAt mutation.
func (o *Observe) InsertValueAt(index int, value any) {
	o.eachObserverFor("value", func(o Observer) { o.InsertValueAt(index, value) })
}

// RemoveValueAt observes a RemoveValueAt mutation.
func (o *Observe) RemoveValueAt(index int) {
	o.eachObserverFor("value", func(o Observer) { o.RemoveValueAt(index) })
}

// SetValueAt observes a SetValueAt mutation.
func (o *Observe) SetValueAt(index int, value any) {
	o.eachObserverFor("value", func(o Observer) { o.SetValueAt(index, value) })
}

// SetValueFor observes a SetValueFor mutation.
func (o *Observe) SetValueFor(key string, value any) {
	o.eachObserverFor(key, func(o Observer) { o.SetValueFor(key, value) })
}

// RemoveValueFor observes a RemoveValueFor mutation.
func (o *Observe) RemoveValueFor(key string) {
	o.eachObserverFor(key, func(o Observer) { o.RemoveValueFor(key) })
}

// NullObserver makes it easier to implement partial observers
type NullObserver struct{}

func (n NullObserver) SetValue(string, any)    {}
func (n NullObserver) InsertValueAt(int, any)  {}
func (n NullObserver) RemoveValueAt(int)       {}
func (n NullObserver) SetValueAt(int, any)     {}
func (n NullObserver) SetValueFor(string, any) {}
func (n NullObserver) RemoveValueFor(string)   {}

var _ Observer = NullObserver{}

type actionObserver struct {
	f func(key string, value any)
}

// NewActionObserver allows a simple callback on SetValue.
// Hidden behind a struct pointer rather than being a function type
// to allow comparable
func NewActionObserver(f func(key string, value any)) Observer {
	return &actionObserver{f: f}
}

func (a actionObserver) SetValue(key string, value any) { a.f(key, value) }
func (a actionObserver) InsertValueAt(int, any)         {}
func (a actionObserver) RemoveValueAt(int)              {}
func (a actionObserver) SetValueAt(int, any)            {}
func (a actionObserver) SetValueFor(string, any)        {}
func (a actionObserver) RemoveValueFor(string)          {}
