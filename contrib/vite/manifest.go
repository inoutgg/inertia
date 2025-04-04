package vite

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
)

type rawManifest = map[string]*ManifestEntry

type Manifest struct {
	raw rawManifest
}

type ManifestEntry struct {
	Source         string   `json:"src"`
	File           string   `json:"file"`
	Name           string   `json:"name"`
	CSS            []string `json:"css"`
	Assets         []string `json:"assets"`
	Imports        []string `json:"imports"`
	DynamicImports []string `json:"dynamicImports"`
	IsEntry        bool     `json:"isEntry"`
	IsDynamicEntry bool     `json:"isDynamicEntry"`
}

func (m *Manifest) HTML(name string) ([]template.HTML, []template.HTML, error) {
	seen := make(map[string]bool)

	entry, ok := m.raw[name]
	if !ok {
		return nil, nil, fmt.Errorf("inertia: entry %s not found in manifest", name)
	}

	var css []template.HTML

	var js []template.HTML

	var walk func(*ManifestEntry)
	walk = func(e *ManifestEntry) {
		if seen[e.Name] {
			return
		}

		seen[e.Name] = true

		for _, link := range e.CSS {
			//nolint:gosec
			css = append(css, template.HTML(fmt.Sprintf(
				`<link rel="stylesheet" href="%s" />`, link)))
		}

		for _, link := range e.Assets {
			//nolint:gosec
			js = append(js, template.HTML(fmt.Sprintf(
				`<script type="module" src="%s"></script>`, link)))
		}

		for _, i := range e.Imports {
			walk(m.raw[i])
		}
	}

	walk(entry)

	return css, js, nil
}

// ParseManifest parses a Vite manifest from a byte slice.
// The manifest is a JSON object where keys are entry names and values are
// manifest entries.
func ParseManifest(b []byte) (*Manifest, error) {
	var raw rawManifest
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("inertia: failed to unmarshal manifest: %w", err)
	}

	return &Manifest{raw: raw}, nil
}

// ParseManifestFromFS reads a Vite manifest from a file system.
func ParseManifestFromFS(fsys fs.FS, path string) (*Manifest, error) {
	b, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to read manifest file: %w", err)
	}

	return ParseManifest(b)
}
