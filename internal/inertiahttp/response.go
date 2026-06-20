package inertiahttp

// Response is the HTTP-level Inertia response produced by Renderer.Render.
//
// Headers are written to http.ResponseWriter by the caller; Body is the
// raw JSON (Inertia requests) or HTML (full page loads) payload.
type Response struct {
	Headers map[string]string
	Body    []byte
}
