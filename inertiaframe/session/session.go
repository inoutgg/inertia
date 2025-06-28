package session

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.inout.gg/foundations/http/httpcookie"

	"go.inout.gg/inertia"
)

var _ Session = (*session)(nil)

const SessionCookieName = "_inertiaframe"

//nolint:gochecknoglobals
var bufPool = sync.Pool{New: func() any { return bytes.NewBuffer(nil) }}

//nolint:gochecknoinits
func init() {
	gob.Register(&session{}) //nolint:exhaustruct
	gob.Register([]inertia.ValidationError(nil))
}

// Session is a users local session for storing informational data for the
// inertiaframe to work correctly.
//
// It is primarily used to store information about validation errors,
// and the last visited path.
type Session interface {
	// Path returns the last visited path by the user.
	//
	// It is used to redirect the user to the last visited page
	// when calling inertiaframe.RedirectBack.
	Path() string

	// ErrorBag returns a bag associated with the processed request
	// for which the validation errors occurred.
	ErrorBag() string

	// ValidationErrors returns a list of validation errors that occurred
	// during the processing of the request.
	ValidationErrors() []inertia.ValidationError

	// Save saves the session to the client, typically via a cookie.
	Save(w http.ResponseWriter, opts ...func(*httpcookie.Option)) error

	// Clear completely removes the session from the client.
	Clear(w http.ResponseWriter, r *http.Request)
}

type session struct {
	ErrorBag_         string                    //nolint:revive
	Path_             string                    //nolint:revive
	ValidationErrors_ []inertia.ValidationError //nolint:revive
}

func New(path string, validationErrors []inertia.ValidationError, errorBag string) Session {
	return &session{
		Path_:             path,
		ValidationErrors_: validationErrors,
		ErrorBag_:         errorBag,
	}
}

// FromRequest retrieves a session from the request, if the session
// does not exist, a new session is created.
func FromRequest(r *http.Request) (Session, error) {
	val := httpcookie.Get(r, SessionCookieName)
	if val == "" {
		//nolint:exhaustruct
		return &session{}, nil
	}

	b, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to decode session cookie: %w", err)
	}

	sess := &session{} //nolint:exhaustruct
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(sess); err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to decode session: %w", err)
	}

	return sess, nil
}

func (s *session) ValidationErrors() []inertia.ValidationError {
	ret := s.ValidationErrors_
	s.ValidationErrors_ = nil

	return ret
}

func (s *session) ErrorBag() string {
	ret := s.ErrorBag_
	s.ErrorBag_ = ""

	return ret
}

func (s *session) Path() string { return s.Path_ }

func (s *session) Clear(w http.ResponseWriter, r *http.Request) {
	httpcookie.Delete(w, r, SessionCookieName)
}

func (s *session) Save(w http.ResponseWriter, opts ...func(*httpcookie.Option)) error {
	buf := bufPool.Get().(*bytes.Buffer) //nolint:forcetypeassert
	defer func() {
		bufPool.Put(buf)
		buf.Reset()
	}()

	if err := gob.NewEncoder(buf).Encode(s); err != nil {
		return fmt.Errorf("inertiaframe: failed to encode session: %w", err)
	}

	opts = append(opts, httpcookie.WithExpiresIn(time.Second*1))
	val := base64.RawURLEncoding.EncodeToString(buf.Bytes())
	httpcookie.Set(w, SessionCookieName, val, opts...)

	return nil
}
