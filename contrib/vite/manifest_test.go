package vite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifest(t *testing.T) {
	content, err := os.ReadFile("testdata/manifest.json")
	require.NoError(t, err, "Failed to read test manifest file")

	manifest, err := ParseManifest(content)
	require.NoError(t, err, "ParseManifest failed")

	t.Run("successful parsing", func(t *testing.T) {
		sharedJS, ok := manifest.raw["_shared-B7PI925R.js"]
		require.True(t, ok, "Expected manifest to contain '_shared-B7PI925R.js' entry")
		assert.Equal(t, &ManifestEntry{
			File: "assets/shared-B7PI925R.js",
			Name: "shared",
			CSS:  []string{"assets/shared-ChJ_j-JJ.css"},
		}, sharedJS)

		bar, ok := manifest.raw["views/bar.js"]
		require.True(t, ok, "Expected manifest to contain 'views/bar.js' entry")
		assert.Equal(t, &ManifestEntry{
			File:           "assets/bar-gkvgaI9m.js",
			Name:           "bar",
			Source:         "views/bar.js",
			IsEntry:        true,
			Imports:        []string{"_shared-B7PI925R.js"},
			DynamicImports: []string{"baz.js"},
		}, bar)

		foo, ok := manifest.raw["views/foo.js"]
		require.True(t, ok, "Expected manifest to contain 'views/foo.js' entry")
		assert.Equal(t, &ManifestEntry{
			File:    "assets/foo-BRBmoGS9.js",
			Name:    "foo",
			Source:  "views/foo.js",
			IsEntry: true,
			Imports: []string{"_shared-B7PI925R.js"},
			CSS:     []string{"assets/foo-5UjPuW-k.css"},
		}, foo)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		invalidJSON := []byte(`{invalid json}`)
		_, err := ParseManifest(invalidJSON)
		assert.Error(t, err, "Expected error when parsing invalid JSON")
	})
}

func TestParseManifestFromFS(t *testing.T) {
	t.Run("successful parsing", func(t *testing.T) {
		dir := os.DirFS("testdata")
		manifest, err := ParseManifestFromFS(dir, "manifest.json")
		require.NoError(t, err, "ParseManifestFromFS failed")

		sharedJS, ok := manifest.raw["_shared-B7PI925R.js"]
		require.True(t, ok, "Expected manifest to contain '_shared-B7PI925R.js' entry")
		assert.Equal(t, &ManifestEntry{
			File: "assets/shared-B7PI925R.js",
			Name: "shared",
			CSS:  []string{"assets/shared-ChJ_j-JJ.css"},
		}, sharedJS)

		foo, ok := manifest.raw["views/foo.js"]
		require.True(t, ok, "Expected manifest to contain 'views/foo.js' entry")
		assert.Equal(t, &ManifestEntry{
			File:    "assets/foo-BRBmoGS9.js",
			Name:    "foo",
			Source:  "views/foo.js",
			IsEntry: true,
			Imports: []string{"_shared-B7PI925R.js"},
			CSS:     []string{"assets/foo-5UjPuW-k.css"},
		}, foo)
	})

	t.Run("file not found", func(t *testing.T) {
		dir := os.DirFS("testdata")
		_, err := ParseManifestFromFS(dir, "nonexistent.json")
		assert.Error(t, err, "Expected error when reading non-existent file")
	})

	t.Run("invalid file path", func(t *testing.T) {
		dir := os.DirFS(filepath.Join("testdata", "nonexistent"))
		_, err := ParseManifestFromFS(dir, "manifest.json")
		assert.Error(t, err, "Expected error when using invalid directory")
	})
}
