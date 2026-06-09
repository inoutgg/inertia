package inertiaprop

var (
	_ Merge = (*AppendRootMergeOpts)(nil)
	_ Merge = (*PrependRootMergeOpts)(nil)
	_ Merge = (*PathMergeOpts)(nil)
)

type Merge interface {
	toMergeable() *Mergeable
}

// ToMergeable is an internal re-export function for private toMergeable interface member
// to prevent user-facing API export of internal details.
func ToMergeable(m Merge) *Mergeable {
	if m == nil {
		return nil
	}

	return m.toMergeable()
}

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

// AppendRootMergeOpts configures root-level append merging.
//
// Root-level merging does not support matchOn; use PathMergeOpts or
// DeepMergeOpts if you need match-on behavior.
type AppendRootMergeOpts struct{}

// NewAppendRootMergeOpts creates a root-level append merge configuration.
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

// PrependRootMergeOpts configures root-level prepend merging.
//
// Root-level merging does not support matchOn; use PathMergeOpts or
// DeepMergeOpts if you need match-on behavior.
type PrependRootMergeOpts struct{}

// NewPrependRootMergeOpts creates a root-level prepend merge configuration.
func NewPrependRootMergeOpts() *PrependRootMergeOpts {
	return &PrependRootMergeOpts{}
}

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
// It does not support root-level merging; use AppendRootMergeOpts or PrependRootMergeOpts
// for that purpose.
type PathMergeOpts struct {
	appendKeys  []string
	prependKeys []string
	matchOn     []string
}

// NewPathMergeOpts creates a default PathMergeOpts instance.
func NewPathMergeOpts() *PathMergeOpts {
	return &PathMergeOpts{} //nolint:exhaustruct
}

// Append configures the merge prop to append keys at the specified nested paths.
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

// Prepend configures the merge prop to prepend keys at the specified nested paths.
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

func (o *PathMergeOpts) toMergeable() *Mergeable {
	if o == nil {
		return nil
	}

	// If no paths are configured, the prop is not meaningfully mergeable.
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
