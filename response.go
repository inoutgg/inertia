package inertia

import (
	"net/http"
)

const nFrontChunkSize = 4

var _ http.ResponseWriter = (*responseWriter)(nil)

// responseWriter is a wrapper around http.ResponseWriter that captures
// the status code and whether the response has been written to.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	nFront     int
	front      [nFrontChunkSize][]byte
	back       [][]byte
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.nFront < len(w.front) {
		w.front[w.nFront] = b
		w.nFront++

		return len(b), nil
	}

	w.back = append(w.back, b)

	return len(b), nil
}

func (w *responseWriter) Empty() bool {
	return len(w.back) == 0
}

// flush writes the buffered response to the underlying http.ResponseWriter.
func (w *responseWriter) flush() error {
	w.ResponseWriter.WriteHeader(w.statusCode)

	if w.nFront > 0 {
		for i, chunk := range w.front {
			if _, err := w.ResponseWriter.Write(chunk); err != nil {
				return err
			}
			w.front[i] = nil
		}
		w.nFront = 0
	}

	if len(w.back) > 0 {
		for _, chunk := range w.back {
			if _, err := w.ResponseWriter.Write(chunk); err != nil {
				return err
			}
		}
		w.back = nil
	}

	return nil
}
