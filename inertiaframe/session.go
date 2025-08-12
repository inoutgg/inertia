package inertiaframe

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.inout.gg/foundations/http/httpcookie"

	"go.inout.gg/inertia"
)

type sessCtx struct{}

var kSessCtx = sessCtx{} //nolint:gochecknoglobals

const (
	SessionCookieName = "_inertiaframe"
	SessionPath       = "/"
)

//nolint:gochecknoglobals
var bufPool = sync.Pool{New: func() any { return bytes.NewBuffer(nil) }}

//nolint:gochecknoinits
func init() {
	gob.Register(&session{}) //nolint:exhaustruct
	gob.Register([]inertia.ValidationError(nil))
}

// Session is a user's local session for storing informational data for
// inertiaframe to work correctly.
//
// It is primarily used to store information about validation errors
// and the last visited path.
type session struct {
	ErrorBag_         string                    //nolint:revive
	Path_             string                    //nolint:revive
	ValidationErrors_ []inertia.ValidationError //nolint:revive
}

// sessionFromRequest retrieves a session from the request. If the session
// does not exist, a new session is created.
func sessionFromRequest(r *http.Request) (*session, error) {
	sess, ok := r.Context().Value(kSessCtx).(*session)
	if ok && sess != nil {
		return sess, nil
	}

	val := httpcookie.Get(r, SessionCookieName)
	if val == "" {
		//nolint:exhaustruct
		return &session{}, nil
	}

	b, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to decode session cookie: %w", err)
	}

	sess = &session{} //nolint:exhaustruct
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(sess); err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to decode session: %w", err)
	}

	// Save session for future requests.
	*r = *r.WithContext(context.WithValue(r.Context(), kSessCtx, sess))

	return sess, nil
}

// ValidationErrors returns a list of validation errors that occurred
// during the processing of the request.
//
// Once the errors are accessed, they are cleared from the session.
func (s *session) ValidationErrors() []inertia.ValidationError {
	ret := s.ValidationErrors_
	s.ValidationErrors_ = nil

	return ret
}

// ErrorBag returns the error bag associated with the request
// that produced validation errors.
func (s *session) ErrorBag() string {
	ret := s.ErrorBag_
	s.ErrorBag_ = ""

	return ret
}

// Referer returns the last path visited by the user.
//
// It is used to redirect the user back to the previously visited page
// when calling inertiaframe.RedirectBack.
func (s *session) Referer() string { return s.Path_ }

// Clear completely removes the session from the client.
func (s *session) Clear(w http.ResponseWriter, r *http.Request) {
	httpcookie.Delete(w, r, SessionCookieName)
}

// Save saves the session to the client, typically via a cookie.
func (s *session) Save(w http.ResponseWriter) error {
	buf := bufPool.Get().(*bytes.Buffer) //nolint:forcetypeassert

	defer func() {
		bufPool.Put(buf)
		buf.Reset()
	}()

	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return fmt.Errorf("inertiaframe: failed to encode session: %w", err)
	}

	val := base64.RawURLEncoding.EncodeToString(buf.Bytes())
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    val,
		Path:     SessionPath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now(),
	}

	http.SetCookie(w, cookie)

	return nil
}
