package inertiaprotocol

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.segfaultmedaddy.com/inertia/inertiaalways"
	"go.segfaultmedaddy.com/inertia/inertiadeferred"
	"go.segfaultmedaddy.com/inertia/inertiamerge"
	"go.segfaultmedaddy.com/inertia/inertiaonce"
	"go.segfaultmedaddy.com/inertia/inertiaoptional"
	"go.segfaultmedaddy.com/inertia/inertiaprop"
	"go.segfaultmedaddy.com/inertia/inertiascroll"
	intprop "go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

const (
	benchComponent = "Users/Index"
	benchVersion   = "1.0.0"
)

//nolint:gochecknoglobals
var (
	benchPrevPage    = 1
	benchCurrentPage = 2
	benchNextPage    = 3
	benchOnceExpiry  = time.Now().Add(time.Hour).Unix()
	errBenchDefer    = errors.New("bench: deferred error")
)

// benchBuildRequest constructs a Request matching a typical, fully-loaded
// partial reload: PartialComponent set, a whitelist, a blacklist, reset
// props, except-once props and a scroll merge intent.
func benchBuildRequest() Request {
	return Request{
		URL:               "/users?page=2",
		PartialComponent:  benchComponent,
		ScrollMergeIntent: intprop.ScrollMergeIntentAppend,
		PartialData:       []string{"name", "email", "profile"},
		PartialExcept:     []string{"company"},
		ResetProps:        []string{"tags"},
		ExceptOnceProps:   []string{"plans-once"},
	}
}

// benchBuildInitialRequest constructs a Request for an initial (non-partial)
// Inertia visit: no whitelist/blacklist/reset filters, no scroll intent.
func benchBuildInitialRequest() Request {
	return Request{
		URL:               "/users",
		ScrollMergeIntent: intprop.ScrollMergeIntentAppend,
	}
}

// benchBuildProps returns a realistic, large mix of props exercising every
// Inertia prop kind: always, standard (with several merge strategies),
// optional, deferred (grouped, with rescue, with once), once and scroll.
//
// The shared props are returned alongside and are meant to be attached via
// Context.SharedProps so that the sharedProps metadata list is populated in
// the rendered page.
func benchBuildProps() ([]intprop.Prop, []intprop.Prop) {
	prev := benchPrevPage
	cur := benchCurrentPage
	next := benchNextPage
	expiry := benchOnceExpiry

	sharedProps := []intprop.Prop{
		inertiaalways.New("flash", map[string]string{"status": "ok"}),
		inertiaalways.New("csrf", "token-value"),
	}

	user := map[string]any{
		"id":     1,
		"name":   "Roman",
		"email":  "roman@example.com",
		"active": true,
		"roles":  []string{"admin", "user"},
		"profile": map[string]any{
			"bio":     "hello world",
			"locales": []string{"en", "uk"},
		},
	}

	props := []intprop.Prop{
		// Always-on (bypass partial filters).
		inertiaalways.New("auth", user),
		inertiaalways.NewLazy("unread_count", intprop.LazyFunc(func(_ context.Context) (any, error) {
			return 7, nil
		}), inertiaalways.WithConcurrent),
		inertiaalways.New("permissions", []string{"read", "write", "delete"}),

		// Standard props with merge strategies.
		inertiaprop.New("flash", map[string]string{"info": "saved"}),
		inertiaprop.New("tags", []string{"go", "inertia"},
			inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
		),
		inertiaprop.New("activity", []string{"login"},
			inertiaprop.WithMerge(inertiamerge.NewPrependRoot()),
		),
		inertiaprop.New("settings", map[string]any{
			"theme":         "dark",
			"notifications": true,
			"items":         []string{"a", "b"},
		},
			inertiaprop.WithMerge(
				inertiamerge.NewPaths().
					Append(inertiamerge.At("items")).
					Prepend(inertiamerge.At("recent").On("id")),
			),
		),
		inertiaprop.New("profile", map[string]any{
			"id":    1,
			"posts": []map[string]any{{"id": 1}, {"id": 2}},
		},
			inertiaprop.WithMerge(inertiamerge.NewDeepMerge("id")),
		),
		inertiaprop.New("name", "Roman"),
		inertiaprop.New("email", "roman@example.com"),
		inertiaprop.New("company", "Anomaly"),

		// Optional props (only included on partial reloads).
		inertiaoptional.New("expensive_report", intprop.LazyFunc(func(_ context.Context) (any, error) {
			return map[string]any{"rows": 1000, "sum": 42}, nil
		})),
		inertiaoptional.New("audit_log", intprop.LazyFunc(func(_ context.Context) (any, error) {
			return []string{"login", "update"}, nil
		}), inertiaoptional.WithConcurrent),

		// Deferred props in named groups; one with rescue, one with once.
		inertiadeferred.New("stats", intprop.LazyFunc(func(_ context.Context) (any, error) {
			return map[string]int{"visits": 10, "signups": 1}, nil
		}), inertiadeferred.WithGroup("metrics")),
		inertiadeferred.New("chart", intprop.LazyFunc(func(_ context.Context) (any, error) {
			return []int{1, 2, 3, 4, 5}, nil
		}), inertiadeferred.WithGroup("metrics")),
		inertiadeferred.New("recommendations", intprop.LazyFunc(func(_ context.Context) (any, error) {
			return []string{"a", "b", "c"}, nil
		}), inertiadeferred.WithGroup("sidebar")),
		inertiadeferred.New("resilient", intprop.LazyFunc(func(_ context.Context) (any, error) {
			return nil, errBenchDefer
		}), inertiadeferred.WithRescue(true)),
		inertiadeferred.New("locale", intprop.LazyFunc(func(_ context.Context) (any, error) {
			return "en", nil
		}),
			inertiadeferred.WithGroup("ui"),
			inertiadeferred.WithOnce(inertiaonce.NewOnceOpts().Key("locale-once").ExpiresAt(&expiry)),
		),

		// Once prop at the standard layer.
		inertiaprop.New(
			"plans",
			[]string{"free", "pro"},
			inertiaprop.WithOnce(
				inertiaonce.NewOnceOpts().Key("plans-once").ExpiresAt(&expiry).Fresh(true),
			),
		),

		// Scrollable prop with pagination metadata.
		inertiascroll.New("users_scroll", map[string]any{
			"data": []map[string]any{
				{"id": 1, "name": "a"},
				{"id": 2, "name": "b"},
			},
		},
			inertiascroll.WithPagination(&prev, &next, &cur),
			inertiascroll.WithPageName("page"),
		),
	}

	return props, sharedProps
}

// BenchmarkRenderComplex measures the allocations needed to render a
// realistic, fully-loaded partial Inertia reload through inertiaprotocol.Render.
//
// The request sets every relevant request header (PartialComponent,
// PartialData whitelist, PartialExcept blacklist, ResetProps, ExceptOnceProps
// and ScrollMergeIntent) and the prop set includes always, standard (with
// several merge strategies), optional, deferred (grouped, with rescue, with
// once), once and scroll props, plus shared props.
//
// Run with:
//
//	go test -bench=BenchmarkRenderComplex -benchmem -benchtime=1s ./internal/inertiaprotocol/
func BenchmarkRenderComplex(b *testing.B) {
	props, shared := benchBuildProps()
	req := benchBuildRequest()
	ctx := Context{
		Component:   benchComponent,
		Version:     benchVersion,
		Props:       props,
		SharedProps: shared,
	}

	b.ReportAllocs()

	for b.Loop() {
		page, err := Render(b.Context(), req, ctx)
		if err != nil {
			b.Fatalf("Render: %v", err)
		}

		// Defensive: ensure the render path actually populates a non-trivial page
		// to prevent the compiler from eliding the call.
		if page == nil || page.Component != benchComponent {
			b.Fatalf("unexpected page: %+v", page)
		}
	}
}

// BenchmarkRenderInitial is the same complex prop set as
// BenchmarkRenderComplex, but on an initial (non-partial) Inertia visit:
// no whitelist/blacklist/reset filters. Use it to compare the cost of a
// partial reload (filtering, merge metadata for scroll) against a full
// initial render.
func BenchmarkRenderInitial(b *testing.B) {
	props, shared := benchBuildProps()
	req := benchBuildInitialRequest()
	ctx := Context{
		Component:   benchComponent,
		Version:     benchVersion,
		Props:       props,
		SharedProps: shared,
	}

	b.ReportAllocs()

	for b.Loop() {
		page, err := Render(b.Context(), req, ctx)
		if err != nil {
			b.Fatalf("Render: %v", err)
		}

		if page == nil || page.Component != benchComponent {
			b.Fatalf("unexpected page: %+v", page)
		}
	}
}
