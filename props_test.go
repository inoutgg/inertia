package inertia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProps(t *testing.T) {
	t.Run("NewDeferred", func(t *testing.T) {
		t.Parallel()

		t.Run("Without options", func(t *testing.T) {
			t.Parallel()
			p := NewDeferred("key", func() any { return "val" }, nil)

			assert.Equal(t, "key", p.key)
			assert.Equal(t, "val", p.value())
			assert.Equal(t, p.group, "default")

			assert.True(t, p.lazy)
			assert.True(t, p.ignorable)
			assert.True(t, p.deferred)
			assert.False(t, p.mergeable)
		})

		t.Run("Custom group", func(t *testing.T) {
			t.Parallel()
			p := NewDeferred("key", func() any { return "val" }, &DeferredOptions{
				Group: "custom",
			})

			assert.Equal(t, "key", p.key)
			assert.Equal(t, "val", p.value())
			assert.Equal(t, p.group, "custom")

			assert.True(t, p.lazy)
			assert.True(t, p.ignorable)
			assert.True(t, p.deferred)
			assert.False(t, p.mergeable)
		})

		t.Run("Mergeable", func(t *testing.T) {
			t.Parallel()
			p := NewDeferred("key", func() any { return "val" }, &DeferredOptions{
				Merge: true,
			})

			assert.Equal(t, "key", p.key)
			assert.Equal(t, "val", p.value())
			assert.Equal(t, p.group, "default")

			assert.True(t, p.lazy)
			assert.True(t, p.ignorable)
			assert.True(t, p.deferred)
			assert.True(t, p.mergeable)
		})
	})

	t.Run("NewAlways", func(t *testing.T) {
		t.Parallel()
		p := NewAlways("key", "val")

		assert.Equal(t, "key", p.key)
		assert.Equal(t, "val", p.value())

		assert.False(t, p.lazy)
		assert.False(t, p.ignorable)
		assert.False(t, p.deferred)
		assert.False(t, p.mergeable)
	})

	t.Run("NewOptional", func(t *testing.T) {
		t.Parallel()
		p := NewOptional("key", func() any { return "val" })

		assert.Equal(t, "key", p.key)
		assert.Equal(t, "val", p.value())

		assert.True(t, p.lazy)
		assert.True(t, p.ignorable)
		assert.False(t, p.deferred)
		assert.False(t, p.mergeable)
	})

	t.Run("NewProp", func(t *testing.T) {
		t.Run("Without options", func(t *testing.T) {
			t.Parallel()
			p := NewProp("key", "val", nil)

			assert.Equal(t, "key", p.key)
			assert.Equal(t, "val", p.value())

			assert.False(t, p.lazy)
			assert.True(t, p.ignorable)
			assert.False(t, p.deferred)
			assert.False(t, p.mergeable)
		})

		t.Run("With options", func(t *testing.T) {
			t.Parallel()
			p := NewProp("key", "val", &PropOptions{Merge: true})

			assert.Equal(t, "key", p.key)
			assert.Equal(t, "val", p.value())

			assert.False(t, p.lazy)
			assert.True(t, p.ignorable)
			assert.False(t, p.deferred)
			assert.True(t, p.mergeable)
		})
	})
}
