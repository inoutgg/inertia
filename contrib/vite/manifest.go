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

	// html and js are used to store the HTML and JS assets for a given entry.
	css []template.HTML
	js  []template.HTML
}

type ManifestEntry struct {
	Source         string   `json:"src"`
	File           string   `json:"file"`
	CSS            []string `json:"css"`
	Assets         []string `json:"assets"`
	IsEntry        bool     `json:"isEntry"`
	Name           string   `json:"name"`
	IsDynamicEntry bool     `json:"isDynamicEntry"`
	Imports        []string `json:"imports"`
	DynamicImports []string `json:"dynamicImports"`
}

func (m *Manifest) HTML(name string) ([]template.HTML, []template.HTML, error) {
	seen := make(map[string]bool)
	entry, ok := m.raw[name]
	if !ok {
		return nil, nil, fmt.Errorf("inertia: entry %s not found in manifest", name)
	}

	var html template.HTML
	var walk func(*ManifestEntry)
	walk = func(e *ManifestEntry) {
		if seen[e.Name] {
			return
		}
		seen[e.Name] = true

		for _, css := range e.CSS {
			html += template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="%s" />`, css))
		}

		for _, asset := range e.Assets {
			html += template.HTML(fmt.Sprintf(`<script type="module" src="%s"></script>`, asset))
		}

		for _, i := range e.Imports {
			walk(m.raw[i])
		}
	}

	walk(entry)

	return nil, nil, nil
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
