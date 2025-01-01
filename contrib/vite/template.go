//go:build !production

package vite

import (
	"fmt"
	"html/template"
	"net/url"

	"go.inout.gg/foundations/must"
)

// parseTemplate parses a template from a string.
func parseTemplate(name string, content string) *template.Template {
	return template.Must(template.New(name).Parse(content))
}

// newTemplate creates a new template with Vite support.
func newTemplate(c *Config) *template.Template {
	viteClientURL := must.Must(url.JoinPath(c.ViteAddress, "@vite/client"))
	viteReactRefreshURL := must.Must(url.JoinPath(c.ViteAddress, "@react-refresh"))
	viteClientTemplate := fmt.Sprintf(`<script type="module" src="%s"></script>`, viteClientURL)
	viteReactRefreshTemplate := fmt.Sprintf(`<script type="module">
  import RefreshRuntime from "%s";
  RefreshRuntime.injectIntoGlobalHook(window);
  window.$RefreshReg$ = () => {};
  window.$RefreshSig$ = () => (type) => type;
  window.__vite_plugin_react_preamble_installed__ = true;
</script>`, viteReactRefreshURL)

	tpl := template.New(c.TemplateName).Funcs(template.FuncMap{
		"viteResource": func(path string) template.HTML {
			url := must.Must(url.JoinPath(c.ViteAddress, path))
			return template.HTML(fmt.Sprintf(`<script type="module" src="%s"></script>`, url))
		},
	})

	template.Must(tpl.AddParseTree("viteClient", parseTemplate("inertia/viteClient", viteClientTemplate).Tree))
	template.Must(tpl.AddParseTree("viteReactRefresh", parseTemplate("inertia/viteReactRefresh", viteReactRefreshTemplate).Tree))

	return tpl
}
