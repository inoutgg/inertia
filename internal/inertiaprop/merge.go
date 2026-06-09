package inertiaprop

var (
	_ Merge = (*AppendRootMergeOpts)(nil)
	_ Merge = (*PrependRootMergeOpts)(nil)
	_ Merge = (*PathMergeOpts)(nil)
	_ Merge = (*DeepMergeOpts)(nil)
)

type Merge interface {
	toMergeable() *Mergeable
}

// ToMergeable converts a Merge value to its Mergeable representation.
// The nil guard handles both nil interface and nil typed pointer.
func ToMergeable(m Merge) *Mergeable {
	if m == nil {
		return nil
	}

	return m.toMergeable()
}

// MergeAt identifies a nested path relative to the prop root and an optional
// match key for client-side deduplication.
type MergeAt struct {
	path    string
	matchOn string
}

func NewMergeAt(path string) MergeAt {
	//nolint:exhaustruct
	return MergeAt{path: path}
}

func (m MergeAt) On(key string) MergeAt {
	m.matchOn = key
	return m
}

type AppendRootMergeOpts struct{}

func NewAppendRootMergeOpts() *AppendRootMergeOpts {
	return &AppendRootMergeOpts{}
}

func (o *AppendRootMergeOpts) toMergeable() *Mergeable {
	if o == nil {
		return nil
	}

	//nolint:exhaustruct
	return &Mergeable{
		Append: true,
	}
}

type PrependRootMergeOpts struct{}

func NewPrependRootMergeOpts() *PrependRootMergeOpts {
	return &PrependRootMergeOpts{}
}

func (o *PrependRootMergeOpts) toMergeable() *Mergeable {
	//nolint:exhaustruct
	return &Mergeable{
		Append: false,
	}
}

// PathMergeOpts configures per-path merge behavior. When no paths are
// registered, toMergeable returns nil so the prop is treated as non-mergeable.
type PathMergeOpts struct {
	appendKeys  []string
	prependKeys []string
	matchOn     []string
}

func NewPathMergeOpts() *PathMergeOpts {
	return &PathMergeOpts{} //nolint:exhaustruct
}

// Append registers paths to append. Entries with an empty path are skipped.
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

// Prepend registers paths to prepend. Entries with an empty path are skipped.
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

// toMergeable returns nil when no paths are registered, treating the prop as
// non-mergeable.
func (o *PathMergeOpts) toMergeable() *Mergeable {
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

// DeepMergeOpts opts a prop into deep merging. Unlike regular merge, the
// client recursively merges nested objects rather than replacing them.
type DeepMergeOpts struct {
	matchOn []string
}

func NewDeepMergeOpts(matchOn ...string) *DeepMergeOpts {
	return &DeepMergeOpts{matchOn}
}

func (o *DeepMergeOpts) toMergeable() *Mergeable {
	//nolint:exhaustruct
	return &Mergeable{
		DeepMerge: true,
		MatchOn:   o.matchOn,
	}
}
