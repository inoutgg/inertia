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

		t.Run("Merge", func(t *testing.T) {
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
		val, err := prop.value(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, "val", val)
		assert.True(t, prop.once)
		assert.Equal(t, "remembered-key", prop.onceKey)
		assert.Equal(t, &expiresAt, prop.expiresAt)
		assert.True(t, prop.fresh)
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
		val, err := prop.value(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, val)
		assert.True(t, prop.scroll)
		assert.True(t, prop.mergeable)
		assert.Equal(t, "users.items", prop.scrollPath)
		assert.Equal(t, "users", prop.scrollMeta.PageName)
		assert.Equal(t, &previousPage, prop.scrollMeta.PreviousPage)
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
		val, err := prop.value(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, val)
		assert.Equal(t, "cursor", prop.scrollMeta.PageName)
		assert.Nil(t, prop.scrollMeta.PreviousPage)
		assert.Equal(t, &nextPage, prop.scrollMeta.NextPage)
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

			// arrange
			prop := NewProp("key", "val", &PropOptions{Merge: true})

			// act
			val, err := prop.value(t.Context())

			// assert
			assert.Equal(t, "key", prop.key)
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.False(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.False(t, prop.deferred)
			assert.True(t, prop.mergeable)
			assert.False(t, prop.concurrent)
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
			val, err := prop.value(t.Context())

			// assert
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.True(t, prop.mergeable)
			assert.True(t, prop.prepend)
			assert.True(t, prop.deepMerge)
			assert.Equal(t, []string{"id"}, prop.matchOn)
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
