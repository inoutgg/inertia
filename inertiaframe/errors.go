package inertiaframe

import (
	"fmt"
)

var _ error = (*InertiaError)(nil)

type InertiaError struct {
	Cause error `json:"-"`
}

func (e *InertiaError) Unwrap() error { return e.Cause }

func (e *InertiaError) Error() string {
	if e.Cause == nil {
		return "inertiaframe: unknown error"
	}
	return fmt.Sprintf("inertiaframe: %v", e.Cause)
}
