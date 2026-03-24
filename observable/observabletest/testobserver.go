package observabletest

import "github.com/CCorderZugcat/zugoui/observable"

type Observer struct {
	observable.NullObserver
	ch    chan *Observer
	Op    string
	Index int
	Key   string
	Value any
}

func New() (*Observer, chan *Observer) {
	ch := make(chan *Observer, 1)
	return &Observer{ch: ch}, ch
}

func (o *Observer) SetValue(key string, value any) {
	o.ch <- &Observer{Op: "setValue", Key: key, Value: value}
}
