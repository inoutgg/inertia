package inertiaprop

var _ Merge = (*MergeOpts)(nil)

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

// MergeOpts configures merge prop behavior.
type MergeOpts struct {
	appendKeys  []string
	prependKeys []string
	matchOn     []string

	isAppend bool
}

// NewMergeOpts creates a default MergeOpts instance with append behavior enabled.
func NewMergeOpts() *MergeOpts {
	return &MergeOpts{isAppend: true} //nolint:exhaustruct
}

// NewMergeAppend creates a MergeOpts for root-level append merging.
func NewMergeAppend() *MergeOpts {
	return NewMergeOpts()
}

// NewMergePrepend creates a MergeOpts for root-level prepend merging.
func NewMergePrepend() *MergeOpts {
	return &MergeOpts{isAppend: false} //nolint:exhaustruct
}

// Append configures the merge prop to append keys to the existing prop value.
func (o *MergeOpts) Append(keys ...MergeAt) *MergeOpts {
	if len(keys) > 0 {
		for _, k := range keys {
			if k.path == "" {
				if k.matchOn != "" {
					o.matchOn = append(o.matchOn, k.matchOn)
				}

				continue
			}

			o.appendKeys = append(o.appendKeys, k.path)
			o.matchOn = append(o.matchOn, k.path+"."+k.matchOn)
		}
	} else {
		o.isAppend = true
	}

	return o
}

// Prepend configures the merge prop to prepend keys to the existing prop value.
func (o *MergeOpts) Prepend(keys ...MergeAt) *MergeOpts {
	if len(keys) > 0 {
		for _, k := range keys {
			if k.path == "" {
				if k.matchOn != "" {
					o.matchOn = append(o.matchOn, k.matchOn)
				}

				continue
			}

			o.prependKeys = append(o.prependKeys, k.path)
			o.matchOn = append(o.matchOn, k.path+"."+k.matchOn)
		}
	} else {
		o.isAppend = false
	}

	return o
}

// ToMergeable converts MergeOpts to Mergeable.
func (o *MergeOpts) toMergeable() *Mergeable {
	if o == nil {
		return nil
	}

	return &Mergeable{
		DeepMerge:   false,
		Append:      o.isAppend,
		AppendKeys:  o.appendKeys,
		PrependKeys: o.prependKeys,
		MatchOn:     o.matchOn,
	}
}
