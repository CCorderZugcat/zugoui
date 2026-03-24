package observable

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownTransformer = errors.New("unknown transformer")
)

// Binding is a (potentially two way) binding between two Observables,
// where at least one is a MutableSource
type Binding struct {
	sourceBinding *observeAs
	destBinding   *observeAs
	x             Transformer
	source, dest  MutableSource
}

// observeAs is our fundamental mini-binding from
// sourceKeyPath in source to destKeyPath in dest.
type observeAs struct {
	NullObserver
	destKeyPath   string
	dest          MutableSource
	sourceKeyPath string
	source        Source
}

func newObserveAs(
	destKeyPath string,
	dest MutableSource,
	sourceKeyPath string,
	source Source,
) *observeAs {
	o := &observeAs{
		destKeyPath:   destKeyPath,
		dest:          dest,
		sourceKeyPath: sourceKeyPath,
		source:        source,
	}
	o.source.AddObserver(o.sourceKeyPath, o)

	return o
}

func (o *observeAs) Release() {
	o.source.RemoveObserver(o.sourceKeyPath, o)
}

func (o *observeAs) SetValue(key string, value any) {
	o.dest.SetValue(o.destKeyPath, value)
}

// NewBinding creates a new Binding
func NewBinding(
	destKeyPath string,
	dest MutableSource,
	sourceKeyPath string,
	source Source,
	xformName string,
) (*Binding, error) {
	_, mutableSource := source.(MutableSource)
	b := &Binding{
		dest: NewWriter(dest),
	}

	if xformName != "" {
		if b.x = NewTransformer(xformName, sourceKeyPath, source); b.x == nil {
			return nil, fmt.Errorf("%w: %s", ErrUnknownTransformer, xformName)
		}

		b.sourceBinding = newObserveAs(destKeyPath, b.dest, "value", b.x)
		b.dest.SetValue(destKeyPath, b.x.Value("value"))

		mutableSource = mutableSource && b.x.Mutable()
		if mutableSource {
			b.destBinding = newObserveAs("value", b.x, destKeyPath, b.dest)
		}
	} else {
		dest.SetValue(destKeyPath, source.Value(sourceKeyPath))

		if mutableSource {
			b.source = NewWriter(source.(MutableSource))
			b.sourceBinding = newObserveAs(destKeyPath, b.dest, sourceKeyPath, b.source)
			b.destBinding = newObserveAs(sourceKeyPath, b.source, destKeyPath, b.dest)
		} else {
			b.sourceBinding = newObserveAs(destKeyPath, b.dest, sourceKeyPath, source)
		}
	}

	return b, nil
}

func (b *Binding) Release() {
	if b.x != nil {
		b.x.Release()
	}
	if b.sourceBinding != nil {
		b.sourceBinding.Release()
	}
	if b.destBinding != nil {
		b.destBinding.Release()
	}
	if b.source != nil {
		b.source.Release()
	}
	if b.dest != nil {
		b.dest.Release()
	}
}
