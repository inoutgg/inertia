package inertiaprop

var (
	_ Merge = (*AppendRootMergeOpts)(nil)
	_ Merge = (*PrependRootMergeOpts)(nil)
	_ Merge = (*PathMergeOpts)(nil)
)

type Merge interface {
	toMergeable() *Mergeable
}

// ToMergeable converts a Merge value to its Mergeable representation.
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

// AppendRootMergeOpts opts a prop into root-level append merging.
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

// PrependRootMergeOpts opts a prop into root-level prepend merging.
type PrependRootMergeOpts struct{}

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
type PathMergeOpts struct {
	appendKeys  []string
	prependKeys []string
	matchOn     []string
}

func NewPathMergeOpts() *PathMergeOpts {
	return &PathMergeOpts{} //nolint:exhaustruct
}

// Append registers nested paths to append during client-side merging.
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
