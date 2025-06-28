package session

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"net/http"
	"sync"

	"go.inout.gg/foundations/http/httpcookie"
	"go.inout.gg/inertia"
)

const SessionCookieName = "_inertiaframe"

//nolint:gochecknoglobals
var bufPool = sync.Pool{New: func() any { return bytes.NewBuffer(nil) }}

//nolint:gochecknoinits
func init() {
	gob.Register(&Session{})
	gob.Register([]inertia.ValidationError(nil))
}

type Session struct {
	ValidationErrors []inertia.ValidationError // flash
	ErrorBag         string                    // flash
}

func Load(r *http.Request) (*Session, error) {
	val := httpcookie.Get(r, SessionCookieName)
	if val == "" {
		return &Session{}, nil
	}

	b, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to decode session cookie: %w", err)
	}

	session := &Session{}
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(session); err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to decode session: %w", err)
	}

	return session, nil
}

func (s *Session) Save(w http.ResponseWriter, opts ...func(*httpcookie.Option)) error {
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		bufPool.Put(buf)
		buf.Reset()
	}()

	if err := gob.NewEncoder(buf).Encode(s); err != nil {
		return fmt.Errorf("inertiaframe: failed to encode session: %w", err)
	}

	val := base64.RawURLEncoding.EncodeToString(buf.Bytes())

	httpcookie.Set(w, SessionCookieName, val, opts...)

	return nil
}
