//go:build production

package vite

import "html/template"

var (
	noopTemplate = template.Must(template.New("noop").Parse(""))
)

func newTemplate(c *Config) *template.Template {
	t := template.New(c.TemplateName)
	t.Funcs(template.FuncMap{
		"viteResource": func(path string) template.HTML {
			return template.HTML("")
		}
	})

	t.AddParseTree("viteClient", noopTemplate.Tree)
	t.AddParseTree("viteReactRefresh", noopTemplate.Tree)

	return t
}
