package inertiaprop

// OnceOpts configures once-per-session prop behavior. The client tracks
// loaded props via the X-Inertia-Except-Once-Props header.
type OnceOpts struct {
	expiresAt *int64
	key       string
	fresh     bool
}

func NewOnceOpts() *OnceOpts {
	return &OnceOpts{} //nolint:exhaustruct
}

// Key sets the identifier the client uses to track loaded props. If unset, the
// prop key is used as the default.
func (o *OnceOpts) Key(key string) *OnceOpts {
	o.key = key
	return o
}

func (o *OnceOpts) ExpiresAt(expiresAt *int64) *OnceOpts {
	o.expiresAt = expiresAt
	return o
}

// Fresh marks the prop as sent on the current request even if already loaded.
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
