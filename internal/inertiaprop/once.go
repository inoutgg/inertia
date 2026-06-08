package inertiaprop

// OnceOpts configures once prop behavior.
type OnceOpts struct {
	expiresAt *int64
	key       string
	fresh     bool
}

func NewOnceOpts() *OnceOpts {
	return &OnceOpts{} //nolint:exhaustruct
}

func (o *OnceOpts) Key(key string) *OnceOpts {
	o.key = key
	return o
}

func (o *OnceOpts) ExpiresAt(expiresAt *int64) *OnceOpts {
	o.expiresAt = expiresAt
	return o
}

func (o *OnceOpts) Fresh(fresh bool) *OnceOpts {
	o.fresh = fresh
	return o
}

// ToOnceable converts OnceOpts to Onceable.
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
