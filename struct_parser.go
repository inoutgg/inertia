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
	propDiscard    = "-"          //nolint:gochecknoglobals
	propOmitEmpty  = "omitempty"  //nolint:gochecknoglobals
	propMergeable  = "mergeable"  //nolint:gochecknoglobals
	propConcurrent = "concurrent" //nolint:gochecknoglobals
)

var lazyType = reflect.TypeOf((*Lazy)(nil)).Elem() //nolint:gochecknoglobals

// ParseStruct returns a Props set parsed from v.
//
// ParseStruct expects a struct pointer as input with JSON encodable fields.
// By default, all fields are ignored unless they are tagged with the "inertia" tag.
//
// The inertia tag follows the format "field_name,optional|deferred|always|<empty>,mergeable|<empty>,omitempty|<empty>"
// The tag can be used to control how the field is handled during the parsing process.
// The inertia tag contains a comma-separated list of options.
//
// The first item in the list denotes the field name.
//
// The second item in the list denotes the field's behavior and can be one of the following:
//   - "optional": The field is optional and will be included in the response if it is present.
//   - "deferred": The field is deferred and will be included in the response if it is present.
//   - "always": The field is always included in the response.
//   - empty string: The field is omitted from the response.
//
// The third positional item in the tag can be one of the following:
//   - "mergeable": The field is mergeable and will be merged with the existing value if it is present.
//   - empty string: The field is not mergeable.
//
// The fourth positional item in the tag can be one of the following:
//   - "omitempty": The field is omitted from the response if it is empty.
//   - empty string: The field is not omitted from the response if it is empty.
//
// By default, deferred (optional, deferred) fields are assigned to the
// default group "default". An optional "inertiagroup" tag can be used for
// grouping deferrable fields. If a non-deferrable field is tagged by "inertiagroup"
// an error will be returned.
// The argument of the "inertiagroup" tag denotes the group name into which the field belongs.
func ParseStruct(v any) (Props, error) {
	val := reflect.ValueOf(v)
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
		if inertiaTag == "" {
			continue
		}

		// Get the inertiaGroup tag, if any
		inertiaGroup := field.Tag.Get(TagInertiaGroup)

		fieldName := field.Name
		fieldType := ""
		mergeable := false
		concurrent := false

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

			// Fourth part is concurrent flag
			if len(parts) > 3 && parts[3] == propConcurrent {
				concurrent = true
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

		var prop Prop

		switch fieldType {
		case propTypeOptional:
			fn, err := toLazy(fieldVal)
			if err != nil {
				return nil, err
			}

			prop = NewOptional(fieldName, fn)
		case propTypeDeferred:
			fn, err := toLazy(fieldVal)
			if err != nil {
				return nil, err
			}

			prop = NewDeferred(
				fieldName,
				fn,
				&DeferredOptions{
					Merge:      mergeable,
					Group:      cmp.Or(inertiaGroup, DefaultDeferredGroup),
					Concurrent: concurrent,
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

// toLazy converts a reflect.Value to an Lazy
// if the value is Lazy convertible.
func toLazy(v reflect.Value) (Lazy, error) {
	val := v.Interface()
	if v.Kind() == reflect.Interface && v.Type().Implements(lazyType) {
		lazy, ok := val.(Lazy)
		if !ok {
			return nil, errors.New("inertiaframe: invalid lazy value")
		}

		return lazy, nil
	}

	if v.Kind() == reflect.Func {
		lazyFn, ok := val.(LazyFunc)
		if !ok {
			return nil, errors.New("inertiaframe: invalid lazy function")
		}

		return LazyFunc(lazyFn), nil
	}

	return nil, errors.New("inertiaframe: invalid lazy value")
}
