package scroll

import (
	"reflect"
	"strconv"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/controllers"
)

// Scroll controls a window of any array or slice
type Scroll struct {
	*observable.Observe
	observable.NullSource
	so               *scrollObserver
	data             observable.Source
	offset, capacity int
}

func init() {
	controllers.RegisterController("scroll", New)
}

type scrollObserver struct {
	observable.NullObserver
	s *Scroll
}

func (s *scrollObserver) SetValue(key string, value any) {
	index, err := strconv.Atoi(key)
	if err != nil {
		return
	}

	index -= s.s.offset
	if index < 0 || index >= s.s.capacity {
		return
	}

	s.s.Observe.SetValue(strconv.Itoa(index), value)
}

func (s *scrollObserver) InsertValueAt(index int, value any) {
	s.s.Observe.SetValue("canDown", s.s.canDown())
}

func (s *scrollObserver) RemoveValueAt(index int) {
	length, _ := s.s.data.Value("len").(int)
	s.s.offset = max(0, min(length-s.s.capacity, s.s.offset))
	s.s.Observe.SetValue("canUp", s.s.canUp())
	s.s.Observe.SetValue("canDown", s.s.canDown())
}

// New is the runtime pluggable form of NewScroll
func New(data observable.Source, args []string) observable.Source {
	var capacity int

	if len(args) > 0 {
		capacity, _ = strconv.Atoi(args[0])
	}

	return NewScroll(data, capacity)
}

// NewScroll creates a new Scroll instance
func NewScroll(data observable.Source, capacity int) *Scroll {
	s := &Scroll{
		Observe:  observable.New(),
		data:     data,
		capacity: capacity,
		so:       &scrollObserver{},
	}
	s.so.s = s
	data.AddObserver("", s.so)

	return s
}

// Up decrements the offset
func (s *Scroll) Up(string) {
	if s.canUp() {
		s.offset--
		s.updateAll()
	}
}

// Down increments the offset
func (s *Scroll) Down(string) {
	if s.canDown() {
		len, _ := s.data.Value("len").(int)
		s.offset = max(min(len-s.capacity, s.offset+1), 0)
		s.updateAll()
	}
}

// PageUp decrements offset by capacity
func (s *Scroll) PageUp(string) {
	if s.canUp() {
		s.offset = max(0, s.offset-s.capacity)
		s.updateAll()
	}
}

// PageDown increments offset by capacity
func (s *Scroll) PageDown(string) {
	if s.canDown() {
		len, _ := s.data.Value("len").(int)
		s.offset = max(min(len-1, s.offset+s.capacity), 0)
		s.updateAll()
	}
}

func (s *Scroll) updateAll() {
	for index := range s.capacity {
		s.Observe.SetValue(strconv.Itoa(index), s.data.ValueAt(index+s.offset))
	}
	s.Observe.SetValue("canUp", s.canUp())
	s.Observe.SetValue("canDown", s.canDown())
}

func (s *Scroll) Insert(string) {
	data, ok := s.data.(observable.MutableSource)
	if !ok {
		return
	}

	var zero reflect.Value
	e := data.Elem()

	if e.Kind() == reflect.Pointer {
		// try to avoid SetValue notifications with nil structs for gorpc limitations
		zero = reflect.New(e.Elem())
	} else {
		zero = reflect.New(e).Elem()
	}

	if !(zero.IsValid() || zero.CanInterface()) {
		return
	}

	data.InsertValueAt(s.offset, zero.Interface())
}

func (s *Scroll) Release() {
	s.data.RemoveObserver("", s.so)
	s.Observe.Release()
}

func (s *Scroll) Model() reflect.Value {
	return s.data.Model()
}

func (s *Scroll) Elem() reflect.Type {
	return s.data.Elem()
}

func (s *Scroll) SetValue(key string, value any) {
	m, ok := s.data.(observable.MutableSource)
	if !ok {
		return
	}

	index, err := strconv.Atoi(key)
	if err != nil {
		return
	}
	index -= s.offset
	if index < 0 || index >= s.capacity {
		return
	}
	m.SetValue(strconv.Itoa(index), value)
}

func (s *Scroll) Keys() []string {
	keys := make([]string, 0, s.capacity)
	for i := range s.capacity {
		keys = append(keys, strconv.Itoa(i))
	}
	return keys
}

func (s *Scroll) Value(key string) any {
	switch key {
	case "len":
		length, ok := s.data.Value("len").(int)
		if !ok {
			return nil
		}
		return min(length, s.capacity)

	case "cap":
		return s.capacity

	case "canUp":
		return s.canUp()

	case "canDown":
		return s.canDown()

	case "value":
		return s.data

	default:
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 || index >= s.capacity {
			return nil
		}
		return s.data.Value(strconv.Itoa(index + s.offset))
	}
}

func (s *Scroll) canUp() bool {
	return s.offset > 0
}

func (s *Scroll) canDown() bool {
	length, _ := s.data.Value("len").(int)
	return s.offset < (length - s.capacity)
}

func (s *Scroll) ValueAt(index int) any {
	if index < 0 || index >= s.capacity {
		return nil
	}
	return s.data.ValueAt(index + s.offset)
}

func (s *Scroll) Source() observable.Source {
	return s.data
}
