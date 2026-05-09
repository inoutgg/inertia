package inertia

// MergeOpts configures merge prop behavior.
//
// Use NewMergeOpts to create a default MergeOpts instance.
type MergeOpts struct {
	append  bool // default
	prepend bool

	prependKeys []MergeKey
	appendKeys  []MergeKey
}

// NewMergeOpts creates a default MergeOpts instance with append behavior enabled.
func NewMergeOpts() *MergeOpts {
	return &MergeOpts{append: true}
}

// MergeKey specifies a field to merge and optionally a match key for existing items.
type MergeKey struct {
	// Key specifies the field name to merge.
	Key string

	// MatchOn specifies what existing items should be matched on and replaced
	// instead of being appended or prepended.
	//
	// MatchOn is optional.
	MatchOn string // optional
}

// Append configures the merge prop to append keys to the existing prop value.
//
// If no keys are provided, the merge prop will be configured to append all keys.
func (o *MergeOpts) Append(keys ...MergeKey) *MergeOpts {
	if len(keys) > 0 {
		o.appendKeys = append(o.appendKeys, keys...)
	} else {
		o.append = true
		o.prepend = false
	}

	return o
}

// Prepend configures the merge prop to prepend keys to the existing prop value.
//
// If no keys are provided, the merge prop will be configured to prepend all keys.
func (o *MergeOpts) Prepend(keys ...MergeKey) *MergeOpts {
	if len(keys) > 0 {
		o.prependKeys = append(o.prependKeys, keys...)
	} else {
		o.append = false
		o.prepend = true
	}

	return o
}
