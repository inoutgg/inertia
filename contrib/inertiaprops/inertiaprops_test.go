package inertiaprops

import (
	"testing"
)

func TestMap_Props(t *testing.T) {
	tests := []struct {
		name    string
		m       Map
		wantLen int
	}{
		{
			name:    "empty map",
			m:       Map{},
			wantLen: 0,
		},
		{
			name: "single string value",
			m: Map{
				"name": "John",
			},
			wantLen: 1,
		},
		{
			name: "multiple values of different types",
			m: Map{
				"name":   "John",
				"age":    30,
				"active": true,
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.Props()
			l := tt.m.Len()

			if len(got) != tt.wantLen || l != tt.wantLen {
				t.Errorf("Map.Props() got length = %v, want length = %v", len(got), tt.wantLen)
			}
		})
	}
}
