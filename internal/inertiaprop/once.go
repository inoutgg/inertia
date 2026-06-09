package inertiaprop

// OnceOpts configures once-per-session prop behavior. A prop is only sent to
// the client on the first full-page load; subsequent Inertia visits skip it.
// The client tracks loaded once-props via the X-Inertia-Except-Once-Props header.
type OnceOpts struct {
	expiresAt *int64
	key       string
	fresh     bool
}

// NewOnceOpts returns a new OnceOpts with default values.
func NewOnceOpts() *OnceOpts {
	return &OnceOpts{} //nolint:exhaustruct
}

// Key sets the identifier used by the client to track whether this prop has
// already been loaded. If not set, the prop key is used as the default.
func (o *OnceOpts) Key(key string) *OnceOpts {
	o.key = key
	return o
}

// ExpiresAt sets a Unix timestamp after which the client should re-fetch the prop.
func (o *OnceOpts) ExpiresAt(expiresAt *int64) *OnceOpts {
	o.expiresAt = expiresAt
	return o
}

// Fresh marks the prop as always being sent on the current request, regardless
// of whether the client has already loaded it. This is useful for props that
// must be refreshed after a form submission or other action.
func (o *OnceOpts) Fresh(fresh bool) *OnceOpts {
	o.fresh = fresh
	return o
}

// ToOnceable converts OnceOpts to Onceable. Returns nil for nil input.
func ToOnceable(opts *OnceOpts) *Onceable {
	if opts == nil {
		return nil
	}

	return &Onceable{
		ExpiresAt: opts.expiresAt,
		Key:       opts.key,
		Fresh:     opts.fresh,
	}
}
