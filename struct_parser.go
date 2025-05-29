package inertia

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	TagInertia      = "inertia"
	TagInertiaGroup = "inertiagroup"
)

var (
	propTypeOptional = "optional" //nolint:gochecknoglobals
	propTypeDeferred = "deferred" //nolint:gochecknoglobals
	propTypeAlways   = "always"   //nolint:gochecknoglobals
)

var (
	propDiscard   = "-"         //nolint:gochecknoglobals
	propOmitEmpty = "omitempty" //nolint:gochecknoglobals
	propMergeable = "mergeable" //nolint:gochecknoglobals
)

var lazyType reflect.Type //nolint:gochecknoglobals

//nolint:gochecknoinits
func init() {
	lazyType = reflect.TypeOf((*Lazy)(nil)).Elem()
}

// ParseFields parses the fields from the msg. msg is expected to be a
// tagged struct.
//
// The rules:
// inertia:"-" omits the field from the response.
// inertia:"field_name,optional|deferred|always|<empty>,mergeable|<empty>,omitempty|<empty>"
// inertiagroup:"group"
func ParseStruct(msg any) (Props, error) {
	val := reflect.ValueOf(msg)
	if val.Kind() != reflect.Ptr {
		return nil, errors.New("msg must be a pointer")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return nil, errors.New("msg must be a struct")
	}

	typ := val.Type()
	numFields := typ.NumField()
	props := make(Props, 0, numFields)

	for i := range numFields {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		inertiaTag := field.Tag.Get(TagInertia)
		if inertiaTag == "-" {
			continue
		}

		// Get the inertiaGroup tag, if any
		inertiaGroup := field.Tag.Get(TagInertiaGroup)

		fieldName := field.Name
		fieldType := ""
		mergeable := false

		// If tag is not empty, parse it
		if inertiaTag != "" {
			parts := strings.Split(inertiaTag, ",")

			if parts[0] != "" {
				fieldName = parts[0]
			}

			// Check if the field should be discarded.
			if fieldName == propDiscard {
				continue
			}

			// Second part is the field type (optional, deferred, always)
			if len(parts) > 1 {
				fieldType = parts[1]
			}

			// Third part is mergeable flag
			if len(parts) > 2 && parts[2] == propMergeable {
				mergeable = true
			}

			// Skip empty fields if omitempty is presented.
			if parts[len(parts)-1] == propOmitEmpty {
				if fieldVal.IsZero() {
					continue
				}
			}
		}

		// Check if field can be accessed
		if !fieldVal.CanInterface() {
			continue
		}

		// Add to the appropriate prop map
		if inertiaGroup != "" && fieldType != propTypeDeferred {
			return nil, errors.New("inertiaframe: cannot use group tag on non-deferred field")
		}

		var prop *Prop

		switch fieldType {
		case propTypeOptional:
			prop = NewOptional(fieldName, toLazy(fieldVal))
		case propTypeDeferred:
			prop = NewDeferred(
				fieldName,
				toLazy(fieldVal),
				&DeferredOptions{
					Merge: mergeable,
					Group: cmp.Or(inertiaGroup, DefaultDeferredGroup),
				},
			)
		case propTypeAlways:
			prop = NewAlways(fieldName, fieldVal.Interface())
		case "":
			prop = NewProp(
				fieldName,
				fieldVal.Interface(),
				&PropOptions{Merge: mergeable},
			)
		default:
			return nil, fmt.Errorf("inertiaframe: unknown field type %q", fieldType)
		}

		props = append(props, prop)
	}

	return props, nil
}

func toLazy(v reflect.Value) Lazy {
	val := v.Interface()
	if v.Kind() == reflect.Interface && v.Type().Implements(lazyType) {
		lazy, ok := val.(Lazy)
		if !ok {
			return nil
		}

		return lazy
	}

	if v.Kind() == reflect.Func {
		lazyFn, ok := val.(func() any)
		if !ok {
			return nil
		}

		return LazyFunc(lazyFn)
	}

	return nil
}
