package observabletest

type Observer struct {
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

func (o *Observer) InsertValueAt(index int, value any) {
	o.ch <- &Observer{Op: "insertValueAt", Index: index, Value: value}
}

func (o *Observer) RemoveValueAt(index int) {
	o.ch <- &Observer{Op: "removeValueAt", Index: index}
}

func (o *Observer) SetValueAt(index int, value any) {
	o.ch <- &Observer{Op: "setValueAt", Index: index, Value: value}
}

func (o *Observer) SetValueFor(key string, value any) {
	o.ch <- &Observer{Op: "setValueFor", Key: key, Value: value}
}

func (o *Observer) RemoveValueFor(key string) {
	o.ch <- &Observer{Op: "removeValueFor", Key: key}
}
