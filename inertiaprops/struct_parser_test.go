package inertiaprops

import (
	"fmt"
	"testing"

	"go.inout.gg/inertia"
)

type A struct {
	Field3 inertia.Lazy `inertia:"field3,optional"`
	Field4 inertia.Lazy `inertia:"field4,deferred"`
	Field1 string       `inertia:"field1"`
	Field2 int          `inertia:"field2"`
}

type H struct{}

func (h *H) Value() any {
	return "value"
}

func TestStructParser(t *testing.T) {
	props, err := ParseStruct(&A{
		Field3: inertia.LazyFunc(func() any { return "lazy" }),
		Field4: &H{},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(props) != 4 {
		t.Errorf("expected 3 fields, got %d", len(props))
	}

	for _, prop := range props {
		//nolint:forbidigo
		fmt.Printf("prop: %+v\n", prop)
	}
}
