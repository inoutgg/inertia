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

			assert.Equal(t, "key", prop.value.key)
			val, err := prop.resolveValue(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val123", val)
			assert.Equal(t, "default", prop.deferred.group)

			assert.True(t, prop.partial.lazy)
			assert.True(t, prop.partial.ignorable)
			assert.True(t, prop.deferred.enabled)
			assert.False(t, prop.merge.enabled)
		})

		t.Run("Custom group", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred("key", LazyFunc(func(context.Context) (any, error) {
				return "deferred-val", nil
			}), &DeferredOptions{
				Group: "custom",
			})

			assert.Equal(t, "key", prop.value.key)
			val, err := prop.resolveValue(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "deferred-val", val)
			assert.Equal(t, "custom", prop.deferred.group)

			assert.True(t, prop.partial.lazy)
			assert.True(t, prop.partial.ignorable)
			assert.True(t, prop.deferred.enabled)
			assert.False(t, prop.merge.enabled)
		})

		t.Run("Merge", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred(
				"key",
				LazyFunc(func(context.Context) (any, error) { return "val", nil }),
				&DeferredOptions{
					Merge: true,
				},
			)

			assert.Equal(t, "key", prop.value.key)
			val, err := prop.resolveValue(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.Equal(t, "default", prop.deferred.group)

			assert.True(t, prop.partial.lazy)
			assert.True(t, prop.partial.ignorable)
			assert.True(t, prop.deferred.enabled)
			assert.True(t, prop.merge.enabled)
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

			assert.Equal(t, "key", prop.value.key)
			val, err := prop.resolveValue(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.Equal(t, "default", prop.deferred.group)

			assert.True(t, prop.partial.lazy)
			assert.True(t, prop.partial.ignorable)
			assert.True(t, prop.deferred.enabled)
			assert.False(t, prop.merge.enabled)
			assert.True(t, prop.deferred.concurrent)
		})
	})

	t.Run("NewAlways", func(t *testing.T) {
		t.Parallel()

		prop := NewAlways("key", "val")

		assert.Equal(t, "key", prop.value.key)
		val, err := prop.resolveValue(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val", val)

		assert.False(t, prop.partial.lazy)
		assert.False(t, prop.partial.ignorable)
		assert.False(t, prop.deferred.enabled)
		assert.False(t, prop.merge.enabled)
	})

	t.Run("NewOptional", func(t *testing.T) {
		t.Parallel()

		prop := NewOptional("key", LazyFunc(func(context.Context) (any, error) { return "val", nil }))

		assert.Equal(t, "key", prop.value.key)
		val, err := prop.resolveValue(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val", val)

		assert.True(t, prop.partial.lazy)
		assert.True(t, prop.partial.ignorable)
		assert.False(t, prop.deferred.enabled)
		assert.False(t, prop.merge.enabled)
		assert.False(t, prop.deferred.concurrent)
	})

	t.Run("NewOnce", func(t *testing.T) {
		t.Parallel()

		// arrange
		expiresAt := int64(123)
		prop := NewOnce("key", LazyFunc(func(context.Context) (any, error) { return "val", nil }), &OnceOptions{
			Key:       "remembered-key",
			ExpiresAt: &expiresAt,
			Fresh:     true,
		})

		// act
		val, err := prop.resolveValue(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, "val", val)
		assert.True(t, prop.once.enabled)
		assert.Equal(t, "remembered-key", prop.once.key)
		assert.Equal(t, &expiresAt, prop.once.expiresAt)
		assert.True(t, prop.once.fresh)
	})

	t.Run("NewScroll", func(t *testing.T) {
		t.Parallel()

		// arrange
		previousPage := 1
		nextPage := 3
		currentPage := 2
		prop := NewScroll("users", []string{"one"}, NewScrollOptions("items", ScrollMetadata[int]{
			PageName:     "users",
			PreviousPage: &previousPage,
			NextPage:     &nextPage,
			CurrentPage:  &currentPage,
		}))

		// act
		val, err := prop.resolveValue(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, val)
		assert.True(t, prop.scroll.enabled)
		assert.True(t, prop.merge.enabled)
		assert.Equal(t, "users.items", prop.scroll.path)
		assert.Equal(t, "users", prop.scroll.meta.PageName)
		assert.Equal(t, &previousPage, prop.scroll.meta.PreviousPage)
	})

	t.Run("NewScroll with cursor metadata", func(t *testing.T) {
		t.Parallel()

		// arrange
		nextPage := "cursor-next"
		currentPage := "cursor-current"
		prop := NewScroll("users", []string{"one"}, NewScrollOptions("data", ScrollMetadata[string]{
			PageName:    "cursor",
			NextPage:    &nextPage,
			CurrentPage: &currentPage,
		}))

		// act
		val, err := prop.resolveValue(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, val)
		assert.Equal(t, "cursor", prop.scroll.meta.PageName)
		assert.Nil(t, prop.scroll.meta.PreviousPage)
		assert.Equal(t, &nextPage, prop.scroll.meta.NextPage)
	})

	t.Run("NewProp", func(t *testing.T) {
		t.Parallel()

		t.Run("Without options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", nil)

			assert.Equal(t, "key", prop.value.key)
			val, err := prop.resolveValue(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)

			assert.False(t, prop.partial.lazy)
			assert.True(t, prop.partial.ignorable)
			assert.False(t, prop.deferred.enabled)
			assert.False(t, prop.merge.enabled)
		})

		t.Run("With options", func(t *testing.T) {
			t.Parallel()

			// arrange
			prop := NewProp("key", "val", &PropOptions{Merge: true})

			// act
			val, err := prop.resolveValue(t.Context())

			// assert
			assert.Equal(t, "key", prop.value.key)
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.False(t, prop.partial.lazy)
			assert.True(t, prop.partial.ignorable)
			assert.False(t, prop.deferred.enabled)
			assert.True(t, prop.merge.enabled)
			assert.False(t, prop.deferred.concurrent)
		})

		t.Run("With v3 merge options", func(t *testing.T) {
			t.Parallel()

			// arrange
			prop := NewProp("key", "val", &PropOptions{
				Merge:     true,
				Prepend:   true,
				DeepMerge: true,
				MatchOn:   []string{"id"},
			})

			// act
			val, err := prop.resolveValue(t.Context())

			// assert
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.True(t, prop.merge.enabled)
			assert.True(t, prop.merge.prepend)
			assert.True(t, prop.merge.deepMerge)
			assert.Equal(t, []string{"id"}, prop.merge.matchOn)
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
		assert.Equal(t, "key1", props.Props()[0].value.key)

		val, err := props.Props()[0].resolveValue(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val1", val)
		assert.Equal(t, "key2", props.Props()[1].value.key)
		val, err = props.Props()[1].resolveValue(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val2", val)
	})
}
