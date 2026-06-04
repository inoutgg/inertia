package inertiaprotocol

// Page contains the page state for the given Inertia.js request.
//
// It is rendered by the Render function.
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
	MatchPropsOn     []string              `json:"matchPropsOn,omitempty"`
	SharedProps      []string              `json:"sharedProps,omitempty"`
	RescuedProps     []string              `json:"rescuedProps,omitempty"`
	PreserveFragment bool                  `json:"preserveFragment,omitempty"`
	EncryptHistory   bool                  `json:"encryptHistory,omitempty"`
	ClearHistory     bool                  `json:"clearHistory,omitempty"`
}

type ScrollProp struct {
	PreviousPage any    `json:"previousPage,omitempty"`
	NextPage     any    `json:"nextPage,omitempty"`
	CurrentPage  any    `json:"currentPage"`
	PageName     string `json:"pageName"`

	// Reset instructs the client to clear the scroll container before applying
	// this page's data. It is set when the prop key appears in the
	// X-Inertia-Reset request header.
	Reset bool `json:"reset,omitempty"`
}

type OnceProp struct {
	ExpiresAt *int64 `json:"expiresAt,omitempty"`
	Prop      string `json:"prop"`
}
