package inertia

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validTplContent = `<!doctype html>
<html>
<head>{{ .InertiaHead }}</head>
<body>{{ .InertiaBody }}</body>
</html>
`

func TestFromFS(t *testing.T) {
	t.Parallel()

	t.Run("it should return renderer when fsys and path are valid", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"templates/app.html": &fstest.MapFile{
				Data: []byte(validTplContent),
			},
		}

		r, err := FromFS(fsys, "templates/app.html", nil)

		require.NoError(t, err)
		assert.NotNil(t, r)
	})

	t.Run("it should return wrapped error when template file is not found", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"templates/app.html": &fstest.MapFile{
				Data: []byte(validTplContent),
			},
		}

		r, err := FromFS(fsys, "templates/missing.html", nil)

		require.Error(t, err)
		require.ErrorContains(t, err, "inertia: failed to parse templates")
		assert.Nil(t, r)
	})

	t.Run("it should return wrapped error when template syntax is invalid", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"templates/app.html": &fstest.MapFile{
				Data: []byte(`{{ .InertiaBody `),
			},
		}

		r, err := FromFS(fsys, "templates/app.html", nil)

		require.Error(t, err)
		require.ErrorContains(t, err, "inertia: failed to parse templates")
		assert.Nil(t, r)
	})
}

func TestMustFromFS(t *testing.T) {
	t.Parallel()

	t.Run("it should return renderer when from fs succeeds", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"templates/app.html": &fstest.MapFile{
				Data: []byte(validTplContent),
			},
		}

		r := MustFromFS(fsys, "templates/app.html", nil)

		assert.NotNil(t, r)
	})

	t.Run("it should panic when from fs fails", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"templates/app.html": &fstest.MapFile{
				Data: []byte(validTplContent),
			},
		}

		assert.Panics(t, func() {
			MustFromFS(fsys, "templates/missing.html", nil)
		})
	})
}
