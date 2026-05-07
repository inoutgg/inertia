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
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val123", val)
			assert.Equal(t, "default", prop.deferred.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
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

			assert.Equal(t, "key", prop.key)
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "deferred-val", val)
			assert.Equal(t, "custom", prop.deferred.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
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

			assert.Equal(t, "key", prop.key)
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.Equal(t, "default", prop.deferred.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
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

			assert.Equal(t, "key", prop.key)
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.Equal(t, "default", prop.deferred.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.True(t, prop.deferred.enabled)
			assert.False(t, prop.merge.enabled)
			assert.True(t, prop.concurrent)
		})
	})

	t.Run("NewAlways", func(t *testing.T) {
		t.Parallel()

		prop := NewAlways("key", "val")

		assert.Equal(t, "key", prop.key)
		val, err := prop.Value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val", val)

		assert.False(t, prop.lazy)
		assert.False(t, prop.ignorable)
		assert.False(t, prop.deferred.enabled)
		assert.False(t, prop.merge.enabled)
	})

	t.Run("NewOptional", func(t *testing.T) {
		t.Parallel()

		prop := NewOptional("key", LazyFunc(func(context.Context) (any, error) { return "val", nil }))

		assert.Equal(t, "key", prop.key)
		val, err := prop.Value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val", val)

		assert.True(t, prop.lazy)
		assert.True(t, prop.ignorable)
		assert.False(t, prop.deferred.enabled)
		assert.False(t, prop.merge.enabled)
		assert.False(t, prop.concurrent)
	})

	t.Run("NewOnce", func(t *testing.T) {
		t.Parallel()

		// arrange
		expiresAt := int64(123)
		prop := NewOnce("key", LazyFunc(func(context.Context) (any, error) { return "val", nil }), &OnceOptions{
			Key:        "remembered-key",
			ExpiresAt:  &expiresAt,
			Fresh:      true,
			Concurrent: true,
		})

		// act
		val, err := prop.Value(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, "val", val)
		assert.True(t, prop.once.enabled)
		assert.Equal(t, "remembered-key", prop.once.key)
		assert.Equal(t, &expiresAt, prop.once.expiresAt)
		assert.True(t, prop.once.fresh)
		assert.True(t, prop.isConcurrent())
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
		val, err := prop.Value(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, val)
		assert.True(t, prop.scroll.enabled)
		assert.True(t, prop.merge.enabled)
		assert.Equal(t, "users.items", prop.scroll.path)
		assert.Equal(t, "users", prop.scroll.PageName)
		assert.Equal(t, &previousPage, prop.scroll.PreviousPage)
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
		val, err := prop.Value(t.Context())

		// assert
		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, val)
		assert.Equal(t, "cursor", prop.scroll.PageName)
		assert.Nil(t, prop.scroll.PreviousPage)
		assert.Equal(t, &nextPage, prop.scroll.NextPage)
	})

	t.Run("NewProp", func(t *testing.T) {
		t.Parallel()

		t.Run("Without options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", nil)

			assert.Equal(t, "key", prop.key)
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)

			assert.False(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.False(t, prop.deferred.enabled)
			assert.False(t, prop.merge.enabled)
		})

		t.Run("With options", func(t *testing.T) {
			t.Parallel()

			// arrange
			prop := NewProp("key", "val", &PropOptions{Merge: true})

			// act
			val, err := prop.Value(t.Context())

			// assert
			assert.Equal(t, "key", prop.key)
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.False(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.False(t, prop.deferred.enabled)
			assert.True(t, prop.merge.enabled)
			assert.False(t, prop.concurrent)
		})

		t.Run("With v3 prepend options", func(t *testing.T) {
			t.Parallel()

			// arrange
			prop := NewProp("key", "val", &PropOptions{
				Prepend: true,
				MatchOn: []string{"id"},
			})

			// act
			val, err := prop.Value(t.Context())

			// assert
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.True(t, prop.merge.enabled)
			assert.True(t, prop.merge.prepend)
			assert.False(t, prop.merge.deepMerge)
			assert.Equal(t, []string{"id"}, prop.merge.matchOn)
		})

		t.Run("With v3 deep merge options", func(t *testing.T) {
			t.Parallel()

			// arrange
			prop := NewProp("key", "val", &PropOptions{
				DeepMerge: true,
				MatchOn:   []string{"messages.id"},
			})

			// assert
			assert.True(t, prop.merge.enabled)
			assert.False(t, prop.merge.prepend)
			assert.True(t, prop.merge.deepMerge)
			assert.Equal(t, []string{"messages.id"}, prop.merge.matchOn)
		})

		t.Run("With conflicting merge options", func(t *testing.T) {
			t.Parallel()

			assert.Panics(t, func() {
				NewProp("key", "val", &PropOptions{
					Merge:     true,
					DeepMerge: true,
				})
			})
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

		val, err := props.Props()[0].Value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val1", val)
		assert.Equal(t, "key2", props.Props()[1].key)
		val, err = props.Props()[1].Value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val2", val)
	})
}

func TestPropCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("deferred merge once", func(t *testing.T) {
		t.Parallel()

		expiresAt := int64(123)
		prop := NewDeferred("users", LazyFunc(func(context.Context) (any, error) {
			return []string{"one"}, nil
		}), &DeferredOptions{
			Once: &OnceOptions{
				Key:        "remembered-users",
				ExpiresAt:  &expiresAt,
				Fresh:      true,
				Concurrent: false,
			},
			Group:      "attributes",
			MatchOn:    []string{"id"},
			Prepend:    true,
			Concurrent: true,
		})

		deferred, ok := prop.deferrable()
		require.True(t, ok)
		assert.Equal(t, "attributes", deferred.group)
		assert.False(t, prop.isFirstIgnorable())
		assert.False(t, prop.shouldIgnoreFilter())
		assert.True(t, prop.isConcurrent())

		merge, ok := prop.mergeable()
		require.True(t, ok)
		assert.True(t, merge.prepend)
		assert.False(t, merge.deepMerge)
		assert.Equal(t, []string{"id"}, merge.matchOn)

		once, ok := prop.onceable()
		require.True(t, ok)
		assert.Equal(t, "remembered-users", once.key)
		assert.Equal(t, &expiresAt, once.expiresAt)
		assert.True(t, once.fresh)
	})

	t.Run("conflicting deferred merge options", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			NewDeferred("users", LazyFunc(func(context.Context) (any, error) {
				return []string{"one"}, nil
			}), &DeferredOptions{
				Merge:   true,
				Prepend: true,
			})
		})
	})

	t.Run("always", func(t *testing.T) {
		t.Parallel()

		prop := NewAlways("auth", map[string]string{"name": "Roman"})

		assert.Equal(t, "auth", prop.Key())
		assert.True(t, prop.isFirstIgnorable())
		assert.True(t, prop.shouldIgnoreFilter())

		_, ok := prop.deferrable()
		assert.False(t, ok)
		_, ok = prop.mergeable()
		assert.False(t, ok)
		_, ok = prop.onceable()
		assert.False(t, ok)
	})

	t.Run("scroll", func(t *testing.T) {
		t.Parallel()

		nextPage := 2
		prop := NewScroll("users", []string{"one"}, NewScrollOptions("data", ScrollMetadata[int]{
			PageName: "page",
			NextPage: &nextPage,
		}))

		scroll, ok := prop.scrollable()
		require.True(t, ok)
		assert.Equal(t, "users.data", scroll.path)
		assert.Equal(t, "page", scroll.PageName)
		assert.Equal(t, &nextPage, scroll.NextPage)

		_, ok = prop.mergeable()
		assert.True(t, ok)
	})
}
