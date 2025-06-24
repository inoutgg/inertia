//nolint:goconst
package inertia

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProps(t *testing.T) {
	t.Parallel()

	t.Run("NewDeferred", func(t *testing.T) {
		t.Parallel()

		t.Run("Without options", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred(
				"key",
				LazyFunc(func(context.Context) (any, error) { return "val123", nil }),
				nil,
			)

			assert.Equal(t, "key", prop.key)
			val, err := prop.value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val123", val)
			assert.Equal(t, "default", prop.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.True(t, prop.deferred)
			assert.False(t, prop.mergeable)
		})

		t.Run("Custom group", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred("key", LazyFunc(func(context.Context) (any, error) {
				return "deferred-val", nil
			}), &DeferredOptions{
				Group: "custom",
			})

			assert.Equal(t, "key", prop.key)
			val, err := prop.value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "deferred-val", val)
			assert.Equal(t, "custom", prop.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.True(t, prop.deferred)
			assert.False(t, prop.mergeable)
		})

		t.Run("Mergeable", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred(
				"key",
				LazyFunc(func(context.Context) (any, error) { return "val", nil }),
				&DeferredOptions{
					Merge: true,
				},
			)

			assert.Equal(t, "key", prop.key)
			val, err := prop.value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.Equal(t, "default", prop.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.True(t, prop.deferred)
			assert.True(t, prop.mergeable)
		})

		t.Run("Concurrent", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred(
				"key",
				LazyFunc(func(context.Context) (any, error) { return "val", nil }),
				&DeferredOptions{
					Concurrent: true,
				},
			)

			assert.Equal(t, "key", prop.key)
			val, err := prop.value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.Equal(t, "default", prop.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.True(t, prop.deferred)
			assert.False(t, prop.mergeable)
			assert.True(t, prop.concurrent)
		})
	})

	t.Run("NewAlways", func(t *testing.T) {
		t.Parallel()

		prop := NewAlways("key", "val")

		assert.Equal(t, "key", prop.key)
		val, err := prop.value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val", val)

		assert.False(t, prop.lazy)
		assert.False(t, prop.ignorable)
		assert.False(t, prop.deferred)
		assert.False(t, prop.mergeable)
	})

	t.Run("NewOptional", func(t *testing.T) {
		t.Parallel()

		prop := NewOptional("key", LazyFunc(func(context.Context) (any, error) { return "val", nil }))

		assert.Equal(t, "key", prop.key)
		val, err := prop.value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val", val)

		assert.True(t, prop.lazy)
		assert.True(t, prop.ignorable)
		assert.False(t, prop.deferred)
		assert.False(t, prop.mergeable)
		assert.False(t, prop.concurrent)
	})

	t.Run("NewProp", func(t *testing.T) {
		t.Parallel()

		t.Run("Without options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", nil)

			assert.Equal(t, "key", prop.key)
			val, err := prop.value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)

			assert.False(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.False(t, prop.deferred)
			assert.False(t, prop.mergeable)
		})

		t.Run("With options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", &PropOptions{Merge: true})

			assert.Equal(t, "key", prop.key)
			val, err := prop.value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)

			assert.False(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.False(t, prop.deferred)
			assert.True(t, prop.mergeable)
			assert.False(t, prop.concurrent)
		})
	})
}

func TestPropsCollections(t *testing.T) {
	t.Parallel()

	t.Run("Props", func(t *testing.T) {
		t.Parallel()

		props := Props{
			NewProp("key1", "val1", nil),
			NewProp("key2", "val2", nil),
		}

		assert.Equal(t, 2, props.Len())
		assert.Len(t, props.Props(), 2)
		assert.Equal(t, "key1", props.Props()[0].key)

		val, err := props.Props()[0].value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val1", val)
		assert.Equal(t, "key2", props.Props()[1].key)
		val, err = props.Props()[1].value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val2", val)
	})
}
