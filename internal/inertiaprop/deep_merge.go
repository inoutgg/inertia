package inertiaprop

var _ Merge = (*DeepMergeOpts)(nil)

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
