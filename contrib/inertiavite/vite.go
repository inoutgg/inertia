package inertiavite

import "html/template"

var ViteClientTemplateFragment = template.Must(template.New("inertia/contrib/vite/client").Parse(`
<script type="module" src="{{.T.Vite}}/@vite/client"></script>
`))

var ViteReactRefreshTemplateFragment = template.Must(template.New("inertia/contrib/vite/react-refresh").Parse(`
<script type="module">
  import RefreshRuntime from "{{.T.Vite}}/@react-refresh";
  RefreshRuntime.injectIntoGlobalHook(window);
  window.$RefreshReg$ = () => {};
  window.$RefreshSig$ = () => (type) => type;
  window.__vite_plugin_react_preamble_installed__ = true;
</script>
`))
