package inertiahttp

// import (
// 	"testing"

// 	"github.com/go-json-experiment/json"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.segfaultmedaddy.com/inertia/inertiaprop"
// 	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
// 	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
// )

// func TestRenderer_render(t *testing.T) {
// 	t.Parallel()

// 	t.Run("returns transport-neutral JSON response", func(t *testing.T) {
// 		t.Parallel()

// 		// arrange
// 		renderer := New(testTpl, &Config{Version: "1.0.0"})
// 		req := inertiahttp.Request{
// 			URL:       "/users",
// 			IsInertia: true,
// 			Version:   "1.0.0",
// 		}
// 		rCtx := NewRenderContext(WithProps(Props{inertiaprop.New("name", "Roman")}))

// 		// act
// 		resp, err := renderer.render(t.Context(), req, "Users/Index", rCtx)

// 		// assert
// 		require.NoError(t, err)
// 		assert.Equal(t, inertiaheader.ContentTypeJSON,
// 			resp.Headers[inertiaheader.HeaderContentType])
// 		assert.Equal(t, "true", resp.Headers[inertiaheader.HeaderXInertia])

// 		var page map[string]any

// 		err = json.Unmarshal(resp.Body, &page)
// 		require.NoError(t, err)
// 		assert.Equal(t, "Users/Index", page["component"])
// 		assert.Equal(t, "/users", page["url"])
// 	})
// }
