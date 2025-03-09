package inertiavalidationerrors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"required,gt=0"`
}

// Double wrapped error
type CustomError struct {
	err error
}

func (d *CustomError) Error() string {
	return "wrapped: " + d.err.Error()
}

func (d *CustomError) Unwrap() error {
	return d.err
}

func setupValidator() (*validator.Validate, ut.Translator) {
	v := validator.New()

	// Create translator
	english := en.New()
	uni := ut.New(english, english)
	trans, _ := uni.GetTranslator("en")

	// Register default English translations
	_ = en_translations.RegisterDefaultTranslations(v, trans)

	return v, trans
}

func TestFromValidationErrors(t *testing.T) {
	v, trans := setupValidator()

	tests := []struct {
		name           string
		input          error
		shouldSucceed  bool
		expectedFields []string
	}{
		{
			name: "valid validation error",
			input: func() error {
				s := TestStruct{}
				return v.Struct(s)
			}(),
			shouldSucceed:  true,
			expectedFields: []string{"Name", "Email", "Age"},
		},
		{
			name: "wrapped validation error",
			input: func() error {
				s := TestStruct{}
				err := v.Struct(s)
				return fmt.Errorf("wrapped error: %w", err)
			}(),
			shouldSucceed:  true,
			expectedFields: []string{"Name", "Email", "Age"},
		},
		{
			name: "double wrapped validation error",
			input: func() error {
				s := TestStruct{}
				err := v.Struct(s)
				wrapped := fmt.Errorf("wrapped error: %w", err)
				return &CustomError{err: wrapped}
			}(),
			shouldSucceed:  true,
			expectedFields: []string{"Name", "Email", "Age"},
		},
		{
			name:          "non-validation error",
			input:         errors.New("just a regular error"),
			shouldSucceed: false,
		},
		{
			name: "wrapped non-validation error",
			input: func() error {
				err := errors.New("regular error")
				return fmt.Errorf("wrapped error: %w", err)
			}(),
			shouldSucceed: false,
		},
		{
			name:          "nil error",
			input:         nil,
			shouldSucceed: false,
		},
		{
			name:          "wrapped nil error",
			input:         fmt.Errorf("wrapped error: %w", nil),
			shouldSucceed: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := FromValidationErrors(tc.input, trans)

			assert.Equal(t, tc.shouldSucceed, ok)

			if tc.shouldSucceed {
				assert.Equal(t, len(tc.expectedFields), len(m))

				for _, field := range tc.expectedFields {
					assert.Contains(t, m, field)
					assert.NotEmpty(t, m[field])
				}
			} else {
				assert.Nil(t, m)
			}
		})
	}
}

func TestMap_WithErrorBag(t *testing.T) {
	m := Map{"Name": "Name is required"}

	assert.Equal(t, "", m.ErrorBag())

	withBag := m.WithErrorBag("login")

	assert.Equal(t, "login", withBag.ErrorBag())
	assert.Equal(t, 1, withBag.Len())

	// Original map should be unchanged
	assert.Equal(t, "", m.ErrorBag())
}
