package inertia

import (
	"fmt"
	"testing"
)

type A struct {
	Field3 Lazy   `inertia:"field3,optional"`
	Field4 Lazy   `inertia:"field4,deferred"`
	Field1 string `inertia:"field1"`
	Field2 int    `inertia:"field2"`
}

type H struct{}

func (h *H) Value() any {
	return "value"
}

func TestStructParser(t *testing.T) {
	props, err := ParseStruct(&A{
		Field3: LazyFunc(func() any { return "lazy" }),
		Field4: &H{},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(props) != 4 {
		t.Errorf("expected 3 fields, got %d", len(props))
	}

	for _, prop := range props {
		fmt.Printf("prop: %+v\n", prop)
	}
}
