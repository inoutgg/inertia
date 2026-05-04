package inertiabase

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageJSONOmitsFalseHistoryFlags(t *testing.T) {
	t.Parallel()

	// arrange
	page := Page{
		Props:     map[string]any{"errors": map[string]any{}},
		Component: "Users/Index",
		URL:       "/users",
		Version:   "1",
	}

	// act
	b, err := json.Marshal(page)

	// assert
	require.NoError(t, err)
	assert.NotContains(t, string(b), "clearHistory")
	assert.NotContains(t, string(b), "encryptHistory")
}

func TestPageJSONIncludesV3OptionalFields(t *testing.T) {
	t.Parallel()

	// arrange
	page := Page{
		Props:            map[string]any{"errors": map[string]any{}},
		Component:        "Users/Index",
		URL:              "/users",
		Version:          "1",
		MergeProps:       []string{"posts"},
		PrependProps:     []string{"notifications"},
		DeepMergeProps:   []string{"conversation"},
		MatchPropsOn:     []string{"posts.id"},
		SharedProps:      []string{"auth"},
		PreserveFragment: true,
		EncryptHistory:   true,
		ClearHistory:     true,
		ScrollProps: map[string]ScrollProp{
			"users": {
				PageName:     "page",
				PreviousPage: nil,
				NextPage:     2,
				CurrentPage:  1,
			},
		},
		OnceProps: map[string]OnceProp{
			"plans": {
				Prop:      "plans",
				ExpiresAt: nil,
			},
		},
	}

	// act
	b, err := json.Marshal(page)

	// assert
	require.NoError(t, err)
	assert.Contains(t, string(b), `"prependProps":["notifications"]`)
	assert.Contains(t, string(b), `"deepMergeProps":["conversation"]`)
	assert.Contains(t, string(b), `"matchPropsOn":["posts.id"]`)
	assert.Contains(t, string(b), `"sharedProps":["auth"]`)
	assert.Contains(t, string(b), `"preserveFragment":true`)
	assert.Contains(t, string(b), `"encryptHistory":true`)
	assert.Contains(t, string(b), `"clearHistory":true`)
	assert.Contains(t, string(b), `"scrollProps":{"users"`)
	assert.Contains(t, string(b), `"onceProps":{"plans"`)
}
