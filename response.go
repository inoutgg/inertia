package inertia

import (
	"bytes"
	"net/http"
	"sync"
)

var (
	_ http.ResponseWriter                       = (*responseWriter)(nil)
	_ interface{ Unwrap() http.ResponseWriter } = (*responseWriter)(nil)
)

var bufPool = sync.Pool{New: func() interface{} { return bytes.NewBuffer(nil) }}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		buf:            bufPool.Get().(*bytes.Buffer),
	}
}

// responseWriter is a wrapper around http.ResponseWriter that defer
// response writing until the flush method is called.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	buf        *bytes.Buffer
	size       int

	flushed bool
	dirty   bool
}

func (w *responseWriter) WriteHeader(code int) {
	w.dirty = true
	w.statusCode = code
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.dirty = true

	n, err := w.buf.Write(b)
	w.size += n

	if err != nil {
		return n, err
	}

	return n, nil
}

func (w *responseWriter) Empty() bool {
	if w.dirty {
		return false
	}

	return w.size == 0
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// flush writes the buffered response to the underlying http.ResponseWriter.
func (w *responseWriter) flush() error {
	if w.flushed {
		return nil
	}

	w.flushed = true

	w.ResponseWriter.WriteHeader(w.statusCode)
	w.ResponseWriter.Write(w.buf.Bytes())

	w.buf.Reset()
	bufPool.Put(w.buf)

	return nil
}
