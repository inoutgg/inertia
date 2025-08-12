package inertiaprops

import "go.inout.gg/inertia"

var _ inertia.Proper = (*Map)(nil)

// Map is a map of key-value pairs that can be used as props
// in the interia adapter.
//
// If  additional options to the props are needed, use inertia.NewProp
// with inertia.Props instead.
type Map map[string]any

func (m Map) Props() []inertia.Prop {
	props := make([]inertia.Prop, 0, len(m))
	for k, v := range m {
		props = append(props, inertia.NewProp(k, v, nil))
	}

	return props
}

func (m Map) Len() int { return len(m) }
