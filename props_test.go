package inertia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProps(t *testing.T) {
	t.Parallel()

	t.Run("NewDeferred", func(t *testing.T) {
		t.Parallel()

		t.Run("Without options", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred("key", LazyFunc(func() any { return "val123" }), nil)

			assert.Equal(t, "key", prop.key)
			assert.Equal(t, "val123", prop.value())
			assert.Equal(t, "default", prop.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.True(t, prop.deferred)
			assert.False(t, prop.mergeable)
		})

		t.Run("Custom group", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred("key", LazyFunc(func() any {
				return "deferred-val"
			}), &DeferredOptions{
				Group: "custom",
			})

			assert.Equal(t, "key", prop.key)
			assert.Equal(t, "deferred-val", prop.value())
			assert.Equal(t, "custom", prop.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.True(t, prop.deferred)
			assert.False(t, prop.mergeable)
		})

		t.Run("Mergeable", func(t *testing.T) {
			t.Parallel()

			prop := NewDeferred("key", LazyFunc(func() any { return "val" }), &DeferredOptions{
				Merge: true,
			})

			assert.Equal(t, "key", prop.key)
			assert.Equal(t, "val", prop.value())
			assert.Equal(t, "default", prop.group)

			assert.True(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.True(t, prop.deferred)
			assert.True(t, prop.mergeable)
		})
	})

	t.Run("NewAlways", func(t *testing.T) {
		t.Parallel()

		prop := NewAlways("key", "val")

		assert.Equal(t, "key", prop.key)
		assert.Equal(t, "val", prop.value())

		assert.False(t, prop.lazy)
		assert.False(t, prop.ignorable)
		assert.False(t, prop.deferred)
		assert.False(t, prop.mergeable)
	})

	t.Run("NewOptional", func(t *testing.T) {
		t.Parallel()

		prop := NewOptional("key", LazyFunc(func() any { return "val" }))

		assert.Equal(t, "key", prop.key)
		assert.Equal(t, "val", prop.value())

		assert.True(t, prop.lazy)
		assert.True(t, prop.ignorable)
		assert.False(t, prop.deferred)
		assert.False(t, prop.mergeable)
	})

	t.Run("NewProp", func(t *testing.T) {
		t.Parallel()

		t.Run("Without options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", nil)

			assert.Equal(t, "key", prop.key)
			assert.Equal(t, "val", prop.value())

			assert.False(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.False(t, prop.deferred)
			assert.False(t, prop.mergeable)
		})

		t.Run("With options", func(t *testing.T) {
			t.Parallel()

			prop := NewProp("key", "val", &PropOptions{Merge: true})

			assert.Equal(t, "key", prop.key)
			assert.Equal(t, "val", prop.value())

			assert.False(t, prop.lazy)
			assert.True(t, prop.ignorable)
			assert.False(t, prop.deferred)
			assert.True(t, prop.mergeable)
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
		assert.Equal(t, "val1", props.Props()[0].value())
		assert.Equal(t, "key2", props.Props()[1].key)
		assert.Equal(t, "val2", props.Props()[1].value())
	})
}
