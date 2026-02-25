package wsrpc

import (
	"errors"
	"sync"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/wsrpc/rpctypes"
)

var (
	ErrInvalidHandle = errors.New("invalid handle")
)

// methods called to the server from the client

// Server is the server side rpc service
type Server struct {
	lck             sync.RWMutex
	valueObservers  map[string]*observable.Observe
	actionObservers *observable.Observe
}

// New creates a new server side rpc instance
func New() *Server {
	return &Server{
		valueObservers:  make(map[string]*observable.Observe),
		actionObservers: observable.New(),
	}
}

// Action receives an action from click binding
func (s *Server) Action(req *rpctypes.ActionReq, _ *bool) error {
	s.actionObservers.SetValue("action", req.Action)
	return nil
}

// AddValueObserver adds a value observer for an action
func (s *Server) AddValueObserver(action string, observer observable.Observer) {
	s.lck.Lock()
	o, ok := s.valueObservers[action]
	if !ok {
		o = observable.New()
		s.valueObservers[action] = o
	}
	s.lck.Unlock()

	o.AddObserver("", observer)
}

// RemoveValueObservers removes all value observers for an action
func (s *Server) RemoveValueObservers(action string) {
	s.lck.Lock()
	defer s.lck.Unlock()

	delete(s.valueObservers, action)
}

// AddActionObserver adds an action observer
func (s *Server) AddActionObserver(observer observable.Observer) {
	s.actionObservers.AddObserver("action", observer)
}

// RemoveActionObservers removes all action observers for an action
func (s *Server) RemoveActionObservers() {
	s.actionObservers.RemoveAllObservers()
}

func (s *Server) observerAt(action string) *observable.Observe {
	s.lck.RLock() // protecting the map of observers, but not the observer itself
	defer s.lck.RUnlock()

	return s.valueObservers[action]
}

// SetValue updates a bound model value
func (s *Server) SetValue(req *rpctypes.SetValueReq, _ *bool) error {
	if o := s.observerAt(req.Action); o != nil {
		o.SetValue(req.Key, req.Value)
	}
	return nil
}

// SetValueAt
func (s *Server) SetValueAt(req *rpctypes.SetValueAtReq, _ *bool) error {
	if o := s.observerAt(req.Action); o != nil {
		o.SetValueAt(req.At, req.Value)
	}
	return nil
}

// InsertValueAt
func (s *Server) InsertValueAt(req *rpctypes.InsertValueAtReq, _ *bool) error {
	if o := s.observerAt(req.Action); o != nil {
		o.InsertValueAt(req.At, req.Value)
	}
	return nil
}

// RemoveValueAt
func (s *Server) RemoveValueAt(req *rpctypes.RemoveValueAtReq, _ *bool) error {
	if o := s.observerAt(req.Action); o != nil {
		o.RemoveValueAt(req.At)
	}
	return nil
}

// SetValueFor
func (s *Server) SetValueFor(req *rpctypes.SetValueForReq, _ *bool) error {
	if o := s.observerAt(req.Action); o != nil {
		o.SetValueFor(req.Key, req.Value)
	}
	return nil
}

// RemoveValueFor
func (s *Server) RemoveValueFor(req *rpctypes.RemoveValueForReq, _ *bool) error {
	if o := s.observerAt(req.Action); o != nil {
		o.RemoveValueFor(req.Key)
	}
	return nil
}
