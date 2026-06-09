package inertiaprop

var (
	_ Merge = (*AppendRootMergeOpts)(nil)
	_ Merge = (*PrependRootMergeOpts)(nil)
	_ Merge = (*PathMergeOpts)(nil)
)

// Merge is the interface accepted by ToMergeable. Each concrete type implements
// toMergeable() for conversion into the wire-format Mergeable.
type Merge interface {
	toMergeable() *Mergeable
}

// ToMergeable converts a Merge value to its Mergeable representation.
// Returns nil when the receiver is nil (handles both nil interface and nil typed pointer).
func ToMergeable(m Merge) *Mergeable {
	if m == nil {
		return nil
	}

	return m.toMergeable()
}

// MergeAt identifies a nested path and an optional match key for client-side merging.
// The path is relative to the prop root (e.g. "messages" for a prop named "conversation").
// When matchOn is set, the client uses it as a key for deduplication when merging.
type MergeAt struct {
	path    string
	matchOn string
}

func NewMergeAt(path string) MergeAt {
	//nolint:exhaustruct
	return MergeAt{path: path}
}

// On sets the match key for this MergeAt and returns the updated value.
func (m MergeAt) On(key string) MergeAt {
	m.matchOn = key
	return m
}

// AppendRootMergeOpts opts a prop into root-level append merging.
// Produces a Mergeable with Append=true and no per-path keys.
type AppendRootMergeOpts struct{}

func NewAppendRootMergeOpts() *AppendRootMergeOpts {
	return &AppendRootMergeOpts{}
}

// toMergeable returns a Mergeable with Append=true. Nil receiver returns nil.
func (o *AppendRootMergeOpts) toMergeable() *Mergeable {
	if o == nil {
		return nil
	}

	//nolint:exhaustruct
	return &Mergeable{
		Append: true,
	}
}

// PrependRootMergeOpts opts a prop into root-level prepend merging.
// Produces a Mergeable with Append=false and no per-path keys.
type PrependRootMergeOpts struct{}

func NewPrependRootMergeOpts() *PrependRootMergeOpts {
	return &PrependRootMergeOpts{}
}

// toMergeable returns a Mergeable with Append=false. Nil receiver returns nil.
func (o *PrependRootMergeOpts) toMergeable() *Mergeable {
	if o == nil {
		return nil
	}

	//nolint:exhaustruct
	return &Mergeable{
		Append: false,
	}
}

// PathMergeOpts configures per-path merge behavior for nested keys within a prop.
// Each path registered via Append or Prepend produces a fully-qualified entry
// in Mergeable.AppendKeys or Mergeable.PrependKeys. If a MergeAt has a matchOn
// value, it is added to Mergeable.MatchOn as "path.matchOn".
//
// PathMergeOpts does not support root-level merging; if no paths are registered,
// toMergeable returns nil so the prop is treated as non-mergeable.
type PathMergeOpts struct {
	appendKeys  []string
	prependKeys []string
	matchOn     []string
}

func NewPathMergeOpts() *PathMergeOpts {
	return &PathMergeOpts{} //nolint:exhaustruct
}

// Append registers nested paths to append during client-side merging.
// Entries with an empty path are skipped. For each non-empty path, the path is
// added to appendKeys and "path.matchOn" is added to matchOn if matchOn is set.
func (o *PathMergeOpts) Append(keys ...MergeAt) *PathMergeOpts {
	for _, k := range keys {
		if k.path == "" {
			continue
		}

		o.appendKeys = append(o.appendKeys, k.path)
		o.matchOn = append(o.matchOn, k.path+"."+k.matchOn)
	}

	return o
}

// Prepend registers nested paths to prepend during client-side merging.
// Entries with an empty path are skipped. For each non-empty path, the path is
// added to prependKeys and "path.matchOn" is added to matchOn if matchOn is set.
func (o *PathMergeOpts) Prepend(keys ...MergeAt) *PathMergeOpts {
	for _, k := range keys {
		if k.path == "" {
			continue
		}

		o.prependKeys = append(o.prependKeys, k.path)
		o.matchOn = append(o.matchOn, k.path+"."+k.matchOn)
	}

	return o
}

// toMergeable returns the Mergeable representation. If no paths or matchOn
// entries are registered, returns nil so the prop is treated as non-mergeable.
func (o *PathMergeOpts) toMergeable() *Mergeable {
	if o == nil {
		return nil
	}

	if len(o.appendKeys) == 0 && len(o.prependKeys) == 0 && len(o.matchOn) == 0 {
		return nil
	}

	//nolint:exhaustruct
	return &Mergeable{
		AppendKeys:  o.appendKeys,
		PrependKeys: o.prependKeys,
		MatchOn:     o.matchOn,
	}
}
