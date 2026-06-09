package inertiamerge

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

type (
	Merge = inertiaprop.Merge

	// MergeAt identifies a prop key and an optional secondary match key used by
	// the client when merging server data into existing client-side state.
	MergeAt = inertiaprop.MergeAt

	// AppendRootMergeOpts configures root-level append merging.
	AppendRootMergeOpts = inertiaprop.AppendRootMergeOpts

	// PrependRootMergeOpts configures root-level prepend merging.
	PrependRootMergeOpts = inertiaprop.PrependRootMergeOpts

	// PathMergeOpts configures per-path merge behavior for nested keys within a prop.
	PathMergeOpts = inertiaprop.PathMergeOpts

	DeepMergeOpts = inertiaprop.DeepMergeOpts
)

func At(path string) MergeAt { return inertiaprop.NewMergeAt(path) }

// NewAppendRoot creates an AppendRootMergeOpts for root-level append merging.
func NewAppendRoot() *AppendRootMergeOpts {
	return inertiaprop.NewAppendRootMergeOpts()
}

// NewPrependRoot creates a PrependRootMergeOpts for root-level prepend merging.
func NewPrependRoot() *PrependRootMergeOpts {
	return inertiaprop.NewPrependRootMergeOpts()
}

// NewPaths creates a PathMergeOpts for per-path merge configuration.
func NewPaths() *PathMergeOpts { return inertiaprop.NewPathMergeOpts() }

// NewDeepMerge creates a DeepMergeOpts for deep merging.
func NewDeepMerge(matchOn ...string) *DeepMergeOpts {
	return inertiaprop.NewDeepMergeOpts(matchOn...)
}
