//go:build js

package input

import (
	"slices"
	"strings"

	"github.com/CCorderZugcat/zugoui/observable"
)

// ValueBindings holds multiple bindings from a single model to fields,
// which may or may not yet exist in the document.
type ValueBindings struct {
	IDPaths map[string]string // element id path to key path

	ids      []string
	inputs   map[string]*Input
	source   observable.Source
	bindings []*observable.Binding
}

// NewValueBindings creates a new ValueBindings object.
// ids is optional if the bind attributes in source fully describe the id.name paths.
func NewValueBindings(ids []string, source observable.Source) (*ValueBindings, error) {
	vb := &ValueBindings{
		IDPaths: make(map[string]string),

		ids:    slices.Clone(ids),
		inputs: make(map[string]*Input),
		source: source,
	}
	if err := vb.Rebind(); err != nil {
		return nil, err
	}
	return vb, nil
}

// Release releases the ValueBindings resources
func (vb *ValueBindings) Release() {
	for _, b := range vb.bindings {
		b.Release()
	}
	for _, i := range vb.inputs {
		i.Release()
	}
	vb.bindings = vb.bindings[:0]
	clear(vb.inputs)
	clear(vb.IDPaths)
}

// Rebind attempts to (re)bind document elements
func (vb *ValueBindings) Rebind() error {
	vb.Release()

	if len(vb.ids) == 0 {
		if err := vb.rebind("", "", vb.source); err != nil {
			return err
		}
	} else {
		for _, id := range vb.ids {
			if err := vb.rebind(id, "", vb.source); err != nil {
				return err
			}
		}
	}

	return nil
}

func (vb *ValueBindings) rebind(idPath, keyPath string, source observable.Source) error {
	for _, key := range source.Keys() {
		keyPath := keyPath
		if keyPath != "" {
			keyPath = keyPath + "." + key
		} else {
			keyPath = key
		}

		idPath := idPath
		if source.Elem() != nil {
			idPath = idPath + "." + key
		}

		bindings := source.Tag(key, "bind")
		if len(bindings) == 0 {
			if source, ok := source.Value(key).(observable.Source); ok {
				if err := vb.rebind(idPath, keyPath, source); err != nil {
					return err
				}
			}
			continue
		}

		for _, binding := range bindings {
			idPath := idPath
			if idPath != "" {
				idPath = idPath + "." + binding
			} else {
				idPath = binding
			}

			idPath, xform, _ := strings.Cut(idPath, ";")
			idPath, property, _ := strings.Cut(idPath, ">")
			if property == "" {
				property = "value"
			}
			idPath = strings.TrimSuffix(idPath, ".")

			if source, ok := source.Value(key).(observable.Source); ok {
				if err := vb.rebind(idPath, keyPath, source); err != nil {
					return err
				}
				if source.Elem() != nil {
					continue // do not bind directly to the base of maps, arrays, or slices
				}
			}

			input, ok := vb.inputs[idPath]
			if !ok {
				element, err := Element(idPath)
				if err != nil {
					continue
				}
				input = NewInput(element)
				vb.inputs[idPath] = input
			}
			vb.IDPaths[idPath] = keyPath

			binding, err := observable.NewBinding(
				property, input,
				keyPath, vb.source,
				xform,
			)
			if err != nil {
				return err
			}

			vb.bindings = append(vb.bindings, binding)
		}
	}

	return nil
}
