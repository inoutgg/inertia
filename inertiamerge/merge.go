package inertiamerge

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

type (
	Merge = inertiaprop.Merge

	// MergeAt identifies a prop key and an optional secondary match key used by
	// the client when merging server data into existing client-side state.
	MergeAt = inertiaprop.MergeAt

	AppendRootMergeOpts  = inertiaprop.AppendRootMergeOpts
	PrependRootMergeOpts = inertiaprop.PrependRootMergeOpts
	PathMergeOpts        = inertiaprop.PathMergeOpts
	DeepMergeOpts        = inertiaprop.DeepMergeOpts
)

func At(path string) MergeAt { return inertiaprop.NewMergeAt(path) }

func NewAppendRoot() *AppendRootMergeOpts {
	return inertiaprop.NewAppendRootMergeOpts()
}

func NewPrependRoot() *PrependRootMergeOpts {
	return inertiaprop.NewPrependRootMergeOpts()
}

func NewPaths() *PathMergeOpts { return inertiaprop.NewPathMergeOpts() }

func NewDeepMerge(matchOn ...string) *DeepMergeOpts {
	return inertiaprop.NewDeepMergeOpts(matchOn...)
}
