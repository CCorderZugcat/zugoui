package observable

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownTransformer = errors.New("unknown transformer")
)

// Binding is a potentially two way binding between two key paths.
// If the source is also mutable, the binding will be created full duplex.
// A transformer may optionally be specified.
type Binding struct {
	sbind, dbind *binding
	source, dest *PathObserver
}

// NewBinding creates a new Binding.
// If no transformer is desired, xformName is an empty string.
func NewBinding(
	sourcePath string,
	source Source,
	destPath string,
	dest MutableSource,
	xformName string,
) (*Binding, error) {
	b := &Binding{}

	_, mutable := source.(MutableSource)
	var get, set func(any) any

	if xformName != "" {
		xform := NewTransformer(xformName)
		if xform == nil {
			return nil, fmt.Errorf("%w: %s", ErrUnknownTransformer, xformName)
		}
		get = xform.Get

		mutable = mutable && xform.Mutable()
		if mutable {
			set = xform.Set
		}
	}

	b.source = NewPathObserver(sourcePath, source)
	b.dest = NewPathObserver(destPath, dest)

	b.sbind = newBinding(sourcePath, b.source, destPath, b.dest, get, true)

	if mutable {
		b.dbind = newBinding(destPath, b.dest, sourcePath, b.source, set, false)
	}

	return b, nil
}

// Release releases all resources associated with a Binding
func (b *Binding) Release() {
	if b.sbind != nil {
		b.sbind.Release()
	}
	if b.dbind != nil {
		b.dbind.Release()
	}
	if b.source != nil {
		b.source.Release()
	}
	if b.dest != nil {
		b.dest.Release()
	}
}

// binding is a half duplex fundamental direction used in an overall Binding object
type binding struct {
	NullObserver
	sourcePath string
	source     *PathObserver
	destPath   string
	dest       *PathObserver
	xform      func(any) any
}

func newBinding(
	sourcePath string,
	source *PathObserver,
	destPath string,
	dest *PathObserver,
	xform func(any) any,
	init bool,
) *binding {
	if xform == nil {
		xform = func(x any) any { return x }
	}

	b := &binding{
		sourcePath: sourcePath,
		source:     source,
		destPath:   destPath,
		dest:       dest,
		xform:      xform,
	}

	source.AddObserver(sourcePath, b)

	if init {
		dest.SetValue(destPath, xform(source.Value(sourcePath)))
	}

	return b
}

func (b *binding) Release() {
	b.source.RemoveObserver(b.sourcePath, b)
}

func (b *binding) SetValue(key string, value any) {
	b.dest.SetValue(b.destPath, b.xform(value))
}
