package inertiaprop

var _ Merge = (*DeepMergeOpts)(nil)

// DeepMergeOpts opts a prop into deep merging. The matchOn values are sent to
// the client as keys for deduplication when merging nested objects.
type DeepMergeOpts struct {
	matchOn []string
}

// NewDeepMergeOpts creates a DeepMergeOpts with the given matchOn keys.
func NewDeepMergeOpts(matchOn ...string) *DeepMergeOpts {
	return &DeepMergeOpts{matchOn}
}

// toMergeable returns a Mergeable with DeepMerge=true and the configured matchOn.
// Nil receiver returns nil.
func (o *DeepMergeOpts) toMergeable() *Mergeable {
	if o == nil {
		return nil
	}

	//nolint:exhaustruct
	return &Mergeable{
		DeepMerge: true,
		MatchOn:   o.matchOn,
	}
}
