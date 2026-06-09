package inertiamerge

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

type (
	Merge = inertiaprop.Merge

	// MergeAt identifies a prop key and an optional secondary match key used by
	// the client when merging server data into existing client-side state.
	MergeAt = inertiaprop.MergeAt

	// AppendRootMergeOpts opts a prop into root-level append merging.
	AppendRootMergeOpts = inertiaprop.AppendRootMergeOpts

	// PrependRootMergeOpts opts a prop into root-level prepend merging.
	PrependRootMergeOpts = inertiaprop.PrependRootMergeOpts

	// PathMergeOpts configures per-path merge behavior for nested keys within a prop.
	PathMergeOpts = inertiaprop.PathMergeOpts

	// DeepMergeOpts opts a prop into deep merging.
	DeepMergeOpts = inertiaprop.DeepMergeOpts
)

// At returns a MergeAt for the given nested path.
func At(path string) MergeAt { return inertiaprop.NewMergeAt(path) }

// NewAppendRoot returns an AppendRootMergeOpts for root-level append merging.
func NewAppendRoot() *AppendRootMergeOpts {
	return inertiaprop.NewAppendRootMergeOpts()
}

// NewPrependRoot returns a PrependRootMergeOpts for root-level prepend merging.
func NewPrependRoot() *PrependRootMergeOpts {
	return inertiaprop.NewPrependRootMergeOpts()
}

// NewPaths returns a PathMergeOpts for configuring per-path merge behavior.
func NewPaths() *PathMergeOpts { return inertiaprop.NewPathMergeOpts() }

// NewDeepMerge returns a DeepMergeOpts for deep merging.
func NewDeepMerge(matchOn ...string) *DeepMergeOpts {
	return inertiaprop.NewDeepMergeOpts(matchOn...)
}
