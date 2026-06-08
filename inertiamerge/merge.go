package inertiamerge

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

type (
	Merge = inertiaprop.Merge

	// MergeAt identifies a prop key and an optional secondary match key used by
	// the client when merging server data into existing client-side state.
	MergeAt = inertiaprop.MergeAt

	// MergeOpts configures the append/prepend direction and per-key merge
	// behavior for a prop built with inertiaprop.WithMerge.
	MergeOpts = inertiaprop.MergeOpts

	DeepMergeOpts = inertiaprop.DeepMergeOpts
)

func At(path string) MergeAt { return inertiaprop.NewMergeAt(path) }

func NewDeepMerge(matchOn ...string) *DeepMergeOpts {
	return inertiaprop.NewDeepMergeOpts(matchOn...)
}

// NewMerge returns a MergeOpts with append behavior enabled by default;
// pass the result to inertiaprop.WithMerge to opt a prop into client-side
// merging.
func NewMerge() *MergeOpts { return inertiaprop.NewMergeOpts() }
