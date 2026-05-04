package inertiabase

type Page struct {
	Props            map[string]any        `json:"props"`
	DeferredProps    map[string][]string   `json:"deferredProps,omitempty"`
	ScrollProps      map[string]ScrollProp `json:"scrollProps,omitempty"`
	OnceProps        map[string]OnceProp   `json:"onceProps,omitempty"`
	Component        string                `json:"component"`
	URL              string                `json:"url"`
	Version          string                `json:"version"`
	MergeProps       []string              `json:"mergeProps,omitempty"`
	PrependProps     []string              `json:"prependProps,omitempty"`
	DeepMergeProps   []string              `json:"deepMergeProps,omitempty"`
	MatchPropsOn     []string              `json:"matchPropsOn,omitempty"`
	SharedProps      []string              `json:"sharedProps,omitempty"`
	PreserveFragment bool                  `json:"preserveFragment,omitempty"`
	EncryptHistory   bool                  `json:"encryptHistory,omitempty"`
	ClearHistory     bool                  `json:"clearHistory,omitempty"`
}

type ScrollProp struct {
	PreviousPage any    `json:"previousPage"`
	NextPage     any    `json:"nextPage"`
	CurrentPage  any    `json:"currentPage"`
	PageName     string `json:"pageName"`
}

type OnceProp struct {
	ExpiresAt *int64 `json:"expiresAt"`
	Prop      string `json:"prop"`
}
