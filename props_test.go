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

			assert.Equal(t, "key", prop.Key())
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val123", val)

			deferred, ok := prop.Deferrable()
			require.True(t, ok)
			assert.Equal(t, "default", deferred.group)
			assert.True(t, prop.IsFirstLoadIgnorable())
			assert.False(t, prop.BypassPartialFilters())
			assert.False(t, prop.Concurrent())

			_, ok = prop.Mergeable()
			assert.False(t, ok)
		})

		t.Run("Custom group", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred("key", LazyFunc(func(context.Context) (any, error) {
				return "deferred-val", nil
			}), &DeferredOpts{
				Group: "custom",
			})

			assert.Equal(t, "key", prop.Key())
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "deferred-val", val)

			deferred, ok := prop.Deferrable()
			require.True(t, ok)
			assert.Equal(t, "custom", deferred.group)
			assert.True(t, prop.IsFirstLoadIgnorable())
			assert.False(t, prop.BypassPartialFilters())

			_, ok = prop.Mergeable()
			assert.False(t, ok)
		})

		t.Run("Merge", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred(
				"key",
				LazyFunc(func(context.Context) (any, error) { return "val", nil }),
				&DeferredOpts{
					Merge: true,
				},
			)

			assert.Equal(t, "key", prop.Key())
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)

			deferred, ok := prop.Deferrable()
			require.True(t, ok)
			assert.Equal(t, "default", deferred.group)
			assert.True(t, prop.IsFirstLoadIgnorable())

			_, ok = prop.Mergeable()
			assert.True(t, ok)
		})

		t.Run("Concurrent", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred(
				"key",
				LazyFunc(func(context.Context) (any, error) { return "val", nil }),
				&DeferredOpts{
					Concurrent: true,
				},
			)

			assert.Equal(t, "key", prop.Key())
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)

			deferred, ok := prop.Deferrable()
			require.True(t, ok)
			assert.Equal(t, "default", deferred.group)
			assert.True(t, prop.IsFirstLoadIgnorable())
			assert.True(t, prop.Concurrent())

			_, ok = prop.Mergeable()
			assert.False(t, ok)
		})
	})

	t.Run("NewAlways", func(t *testing.T) {
		t.Parallel()

		t.Run("eager value", func(t *testing.T) {
			t.Parallel()

			prop := NewAlways("key", "val")

			assert.Equal(t, "key", prop.Key())
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)

			assert.False(t, prop.IsFirstLoadIgnorable())
			assert.True(t, prop.BypassPartialFilters())
			assert.False(t, prop.Concurrent())

			_, ok := prop.Deferrable()
			assert.False(t, ok)
			_, ok = prop.Mergeable()
			assert.False(t, ok)
		})

		t.Run("lazy value", func(t *testing.T) {
			t.Parallel()

			prop := NewAlways("key", LazyFunc(func(context.Context) (any, error) {
				return "lazy-val", nil
			}))

			assert.Equal(t, "key", prop.Key())
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "lazy-val", val)

			assert.False(t, prop.IsFirstLoadIgnorable())
			assert.True(t, prop.BypassPartialFilters())
			assert.False(t, prop.Concurrent())
		})
	})

	t.Run("NewOptional", func(t *testing.T) {
		t.Parallel()

		prop := NewOptional("key", LazyFunc(func(context.Context) (any, error) { return "val", nil }))

		assert.Equal(t, "key", prop.Key())
		val, err := prop.Value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val", val)

		assert.True(t, prop.IsFirstLoadIgnorable())
		assert.False(t, prop.BypassPartialFilters())
		assert.False(t, prop.Concurrent())

		_, ok := prop.Deferrable()
		assert.False(t, ok)
		_, ok = prop.Mergeable()
		assert.False(t, ok)
	})

	t.Run("NewOnce", func(t *testing.T) {
		t.Parallel()

		expiresAt := int64(123)
		prop := NewOnce("key", LazyFunc(func(context.Context) (any, error) { return "val", nil }), &OnceOpts{
			Key:        "remembered-key",
			ExpiresAt:  &expiresAt,
			Fresh:      true,
			Concurrent: true,
		})

		val, err := prop.Value(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "val", val)

		once, ok := prop.Onceable()
		require.True(t, ok)
		assert.Equal(t, "remembered-key", once.key)
		assert.Equal(t, &expiresAt, once.expiresAt)
		assert.True(t, once.fresh)
		assert.True(t, prop.Concurrent())
	})

	t.Run("NewScroll", func(t *testing.T) {
		t.Parallel()

		previousPage := 1
		nextPage := 3
		currentPage := 2
		prop := NewScroll("users", []string{"one"}, NewScrollOptions("items", ScrollMetadata[int]{
			PageName:     "users",
			PreviousPage: &previousPage,
			NextPage:     &nextPage,
			CurrentPage:  &currentPage,
		}))

		val, err := prop.Value(t.Context())

		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, val)

		scroll, ok := prop.Scrollable()
		require.True(t, ok)
		assert.Equal(t, "users.items", scroll.path)
		assert.Equal(t, "users", scroll.PageName)
		assert.Equal(t, &previousPage, scroll.PreviousPage)

		_, ok = prop.Mergeable()
		assert.True(t, ok)
	})

	t.Run("NewScroll with cursor metadata", func(t *testing.T) {
		t.Parallel()

		nextPage := "cursor-next"
		currentPage := "cursor-current"
		prop := NewScroll("users", []string{"one"}, NewScrollOptions("data", ScrollMetadata[string]{
			PageName:    "cursor",
			NextPage:    &nextPage,
			CurrentPage: &currentPage,
		}))

		val, err := prop.Value(t.Context())

		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, val)

		scroll, ok := prop.Scrollable()
		require.True(t, ok)
		assert.Equal(t, "cursor", scroll.PageName)
		assert.Nil(t, scroll.PreviousPage)
		assert.Equal(t, &nextPage, scroll.NextPage)
	})

	t.Run("NewProp", func(t *testing.T) {
		t.Parallel()

		t.Run("Without options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", nil)

			assert.Equal(t, "key", prop.Key())
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "val", val)

			assert.False(t, prop.IsFirstLoadIgnorable())
			assert.False(t, prop.BypassPartialFilters())
			_, ok := prop.Deferrable()
			assert.False(t, ok)
			_, ok = prop.Mergeable()
			assert.False(t, ok)
		})

		t.Run("Lazy value", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", LazyFunc(func(context.Context) (any, error) {
				return "lazy-val", nil
			}), nil)

			assert.Equal(t, "key", prop.Key())
			val, err := prop.Value(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "lazy-val", val)

			assert.False(t, prop.IsFirstLoadIgnorable())
			assert.False(t, prop.BypassPartialFilters())
		})

		t.Run("With options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", &PropOpts{Merge: true})

			val, err := prop.Value(t.Context())

			assert.Equal(t, "key", prop.Key())
			require.NoError(t, err)
			assert.Equal(t, "val", val)
			assert.False(t, prop.IsFirstLoadIgnorable())
			_, ok := prop.Deferrable()
			assert.False(t, ok)
			_, ok = prop.Mergeable()
			assert.True(t, ok)
			assert.False(t, prop.Concurrent())
		})

		t.Run("With v3 prepend options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", &PropOpts{
				Prepend: true,
				MatchOn: []string{"id"},
			})

			val, err := prop.Value(t.Context())

			require.NoError(t, err)
			assert.Equal(t, "val", val)

			merge, ok := prop.Mergeable()
			require.True(t, ok)
			assert.True(t, merge.prepend)
			assert.False(t, merge.deepMerge)
			assert.Equal(t, []string{"id"}, merge.matchOn)
		})

		t.Run("With v3 deep merge options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", &PropOpts{
				DeepMerge: true,
				MatchOn:   []string{"messages.id"},
			})

			merge, ok := prop.Mergeable()
			require.True(t, ok)
			assert.False(t, merge.prepend)
			assert.True(t, merge.deepMerge)
			assert.Equal(t, []string{"messages.id"}, merge.matchOn)
		})

		t.Run("With conflicting merge options", func(t *testing.T) {
			t.Parallel()

			assert.Panics(t, func() {
				NewProp("key", "val", &PropOpts{
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
		assert.Equal(t, "key1", props.Props()[0].Key())

		val, err := props.Props()[0].Value(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "val1", val)
		assert.Equal(t, "key2", props.Props()[1].Key())
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
		}), &DeferredOpts{
			Once: &OnceOpts{
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

		deferred, ok := prop.Deferrable()
		require.True(t, ok)
		assert.Equal(t, "attributes", deferred.group)
		assert.True(t, prop.IsFirstLoadIgnorable())
		assert.False(t, prop.BypassPartialFilters())
		assert.True(t, prop.Concurrent())

		merge, ok := prop.Mergeable()
		require.True(t, ok)
		assert.True(t, merge.prepend)
		assert.False(t, merge.deepMerge)
		assert.Equal(t, []string{"id"}, merge.matchOn)

		once, ok := prop.Onceable()
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
			}), &DeferredOpts{
				Merge:   true,
				Prepend: true,
			})
		})
	})

	t.Run("always", func(t *testing.T) {
		t.Parallel()

		prop := NewAlways("auth", map[string]string{"name": "Roman"})

		assert.Equal(t, "auth", prop.Key())
		assert.False(t, prop.IsFirstLoadIgnorable())
		assert.True(t, prop.BypassPartialFilters())

		_, ok := prop.Deferrable()
		assert.False(t, ok)
		_, ok = prop.Mergeable()
		assert.False(t, ok)
		_, ok = prop.Onceable()
		assert.False(t, ok)
	})

	t.Run("scroll", func(t *testing.T) {
		t.Parallel()

		nextPage := 2
		prop := NewScroll("users", []string{"one"}, NewScrollOptions("data", ScrollMetadata[int]{
			PageName: "page",
			NextPage: &nextPage,
		}))

		scroll, ok := prop.Scrollable()
		require.True(t, ok)
		assert.Equal(t, "users.data", scroll.path)
		assert.Equal(t, "page", scroll.PageName)
		assert.Equal(t, &nextPage, scroll.NextPage)

		_, ok = prop.Mergeable()
		assert.True(t, ok)
	})
}
