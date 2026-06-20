package inertiaprecognition

import (
	"net/http"
	"strings"

	"github.com/go-json-experiment/json"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
)

const wildcard = "*"

// ValidationError is the subset of inertia.ValidationError needed to serialize
// Precognition validation responses without depending on the root package.
type ValidationError interface {
	Field() string
	Error() string
}

// IsRequest reports whether r is a Precognition validation request.
func IsRequest(r *http.Request) bool {
	return r.Header.Get(inertiaheader.HeaderPrecognition) == inertiaheader.HeaderValueTrue
}

// WriteSuccess writes a successful Precognition validation response.
func WriteSuccess(w http.ResponseWriter) {
	writeHeaders(w)
	w.Header().Set(inertiaheader.HeaderPrecognitionSuccess, inertiaheader.HeaderValueTrue)
	w.WriteHeader(http.StatusNoContent)
}

// WriteValidationErrors writes validation errors for a Precognition request.
func WriteValidationErrors[E ValidationError](w http.ResponseWriter, errs []E) {
	writeHeaders(w)
	w.Header().Set(inertiaheader.HeaderContentType, inertiaheader.ContentTypeJSON)
	w.WriteHeader(http.StatusUnprocessableEntity)

	body := must.Must(json.Marshal(struct {
		Errors map[string][]string `json:"errors"`
	}{
		Errors: validationErrors(errs),
	}))

	_, _ = w.Write(body)
}

// FilterValidationErrors returns the validation errors requested by
// Precognition-Validate-Only. If the request does not specify fields, all
// errors are returned.
func FilterValidationErrors[E ValidationError](r *http.Request, errs []E) []E {
	fields := validateOnly(r)
	if len(fields) == 0 {
		return errs
	}

	filtered := make([]E, 0, len(errs))
	for _, err := range errs {
		if fieldRequested(err.Field(), fields) {
			filtered = append(filtered, err)
		}
	}

	return filtered
}

func writeHeaders(w http.ResponseWriter) {
	w.Header().Set(inertiaheader.HeaderPrecognition, inertiaheader.HeaderValueTrue)
	appendVary(w, inertiaheader.HeaderPrecognition)
}

func appendVary(w http.ResponseWriter, value string) {
	for _, header := range w.Header().Values(inertiaheader.HeaderVary) {
		for part := range strings.SplitSeq(header, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}

	w.Header().Add(inertiaheader.HeaderVary, value)
}

func validationErrors[E ValidationError](errs []E) map[string][]string {
	m := make(map[string][]string, len(errs))

	for _, err := range errs {
		m[err.Field()] = append(m[err.Field()], err.Error())
	}

	return m
}

func validateOnly(r *http.Request) []string {
	raw := r.Header.Get(inertiaheader.HeaderPrecognitionValidateOnly)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	fields := make([]string, 0, len(parts))

	for _, part := range parts {
		field := strings.TrimSpace(part)
		if field != "" {
			fields = append(fields, field)
		}
	}

	return fields
}

func fieldRequested(field string, requested []string) bool {
	for _, pattern := range requested {
		if pattern == field || wildcardMatch(pattern, field) {
			return true
		}
	}

	return false
}

func wildcardMatch(pattern string, field string) bool {
	patternParts := strings.Split(pattern, ".")

	fieldParts := strings.Split(field, ".")
	if len(patternParts) != len(fieldParts) {
		return false
	}

	for i, part := range patternParts {
		if part != wildcard && part != fieldParts[i] {
			return false
		}
	}

	return true
}
