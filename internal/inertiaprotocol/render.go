package inertiaprotocol

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alitto/pond/v2"
	"go.inout.gg/foundations/debug"
	"go.opentelemetry.io/otel/metric"

	"go.segfaultmedaddy.com/inertia/inertiaotel"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/sliceutil"
)

var d = debug.Debuglog("inertiaprotocol") //nolint:gochecknoglobals

// Request holds the request for the rendered page.
type Request struct {
	URL               string
	PartialComponent  string
	ScrollMergeIntent string
	PartialData       []string
	PartialExcept     []string
	ResetProps        []string
	ExceptOnceProps   []string
}

// Context holds the context for the rendered page.
//
// The difference between a Context and a Request is that a Context
// is populated by the server and a Request is populated by the client.
type Context struct {
	ResultPool       pond.ResultPool[Result]
	Component        string
	Version          string
	Props            []inertiaprop.Prop
	SharedProps      []inertiaprop.Prop
	PreserveFragment bool
	ClearHistory     bool
	EncryptHistory   bool
}

// Renderer renders an Inertia page from a request and a server-side context.
//
// Renderer is protocol agnostic (meaning it does not contain any HTTP specific logic)
// and all the protocol-specific logic must be handled by the caller.
type Renderer struct {
	rescuedPropsCounter     metric.Int64Counter
	concurrentPropsDuration metric.Float64Histogram
}

// New creates a Renderer configured with the given OpenTelemetry config.
//
// The instruments are created eagerly; if creation fails, no-op instruments
// are used and the error is logged.
func New(telemetry *inertiaotel.Config) *Renderer {
	debug.Assert(telemetry != nil, "telemetry config must be provided")

	rescuedPropsCounter, err := telemetry.Meter().Int64Counter(
		"inertia.props.rescued",
		metric.WithDescription("Number of rescued deferred props"),
	)
	if err != nil {
		d("failed to create inertia.props.rescued counter: %v", err)
	}

	concurrentPropsDuration, err := telemetry.Meter().Float64Histogram(
		"inertia.props.concurrent.resolution.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of concurrent prop resolution"),
	)
	if err != nil {
		d("failed to create inertia.props.concurrent.resolution.duration histogram: %v", err)
	}

	return &Renderer{
		rescuedPropsCounter:     rescuedPropsCounter,
		concurrentPropsDuration: concurrentPropsDuration,
	}
}

// Render returns a rendered page for the given request and context.
func (r *Renderer) Render(ctx context.Context, req Request, renderCtx Context) (*Page, error) {
	debug.Assert(renderCtx.Component != "", "component must be non-empty")
	debug.Assert(renderCtx.ResultPool != nil, "ResultPool must be set")

	d("component=%q partial=%q props=%d shared=%d",
		renderCtx.Component, req.PartialComponent,
		len(renderCtx.Props), len(renderCtx.SharedProps))

	props, rescuedProps, err := r.resolveProps(ctx, req, renderCtx.Component, renderCtx.Props, renderCtx.ResultPool)
	if err != nil {
		return nil, err
	}

	mergeProps, err := makeMergeProps(
		renderCtx.Props,
		req.ResetProps,
		req.ScrollMergeIntent,
	)
	if err != nil {
		return nil, err
	}

	return &Page{
		Component:      renderCtx.Component,
		Props:          props,
		DeferredProps:  makeDeferredProps(req, renderCtx.Component, renderCtx.Props),
		ScrollProps:    makeScrollProps(renderCtx.Props, req.ResetProps),
		OnceProps:      makeOnceProps(renderCtx.Props),
		MergeProps:     mergeProps.append,
		PrependProps:   mergeProps.prepend,
		DeepMergeProps: mergeProps.deepMerge,
		MatchPropsOn:   mergeProps.matchOn,
		SharedProps: sliceutil.Map(
			renderCtx.SharedProps,
			func(p inertiaprop.Prop) string { return p.Key() },
		),
		RescuedProps:     rescuedProps,
		URL:              req.URL,
		Version:          renderCtx.Version,
		PreserveFragment: renderCtx.PreserveFragment,
		ClearHistory:     renderCtx.ClearHistory,
		EncryptHistory:   renderCtx.EncryptHistory,
	}, nil
}

// resolveProps resolves the props for the given request and component.
//
// It handles both full and partial component requests.
func (r *Renderer) resolveProps(
	ctx context.Context,
	req Request,
	componentName string,
	props []inertiaprop.Prop,
	pool pond.ResultPool[Result],
) (map[string]any, []string, error) {
	if req.PartialComponent == componentName {
		d("partial reload for %q whitelist=%v except=%v",
			componentName, req.PartialData, req.PartialExcept)

		return r.resolvePartialComponentRequest(
			ctx,
			props,
			req.PartialData,
			req.PartialExcept,
			req.ExceptOnceProps,
			pool,
		)
	}

	m := make(map[string]any, len(props))

	for _, prop := range props {
		key := prop.Key()

		// Skip once-per-session props that the client has already loaded.
		if shouldSkipOnceProp(prop, req.ExceptOnceProps) {
			continue
		}

		// Skip deferred and optional props on the first render.
		if prop.IsFirstLoadIgnorable() {
			continue
		}

		val, err := prop.Value(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("inertia: failed to resolve prop %s: %w", key, err)
		}

		m[key] = val
	}

	return m, nil, nil
}

func (r *Renderer) resolvePartialComponentRequest(
	ctx context.Context,
	props []inertiaprop.Prop,
	whitelist, blacklist, exceptOnceProps []string,
	pool pond.ResultPool[Result],
) (map[string]any, []string, error) {
	props = sliceutil.Filter(props, func(prop inertiaprop.Prop) bool {
		key := prop.Key()
		if shouldSkipOnceProp(prop, exceptOnceProps) {
			return len(whitelist) > 0 && slices.Contains(whitelist, key)
		}

		if !prop.BypassPartialFilters() {
			if len(whitelist) > 0 && !slices.Contains(whitelist, key) ||
				len(blacklist) > 0 && slices.Contains(blacklist, key) {
				return false
			}
		}

		return true
	})

	m := make(map[string]any, len(props))
	concurrentProps := make([]inertiaprop.Prop, 0, len(props))

	// rescuedProps are deferred props that were failed to resolve,
	// and marked for being rescued (i.e., not terminating the request),
	// but instead partially resolve the response with a report of the failed
	// props and their errors.
	var rescuedProps []string

	for _, prop := range props {
		key := prop.Key()

		if prop.Concurrent() {
			concurrentProps = append(concurrentProps, prop)
		} else {
			val, err := prop.Value(ctx)
			if err != nil {
				// If the error is of type *inertiaprop.RescueError, the prop is rescued
				// and the error is ignored.
				//
				// The rescued properties will be returned to the client via
				// a special field "rescuedProps" in the response.
				if re, ok := errors.AsType[*inertiaprop.RescueError](err); ok {
					d("rescued prop %q: %v", re.Key, re.Err)
					rescuedProps = append(rescuedProps, re.Key)
				} else {
					return nil, nil, fmt.Errorf(
						"inertia: failed to resolve prop %s: %w",
						key, err,
					)
				}
			}

			m[key] = val
		}
	}

	switch len(concurrentProps) {
	case 0:
		// Nothing to do.
	case 1:
		// Single concurrent prop: resolve inline. The pool machinery
		// (linked buffer, task group, composite future) is far more expensive
		// than the Value call itself for a single item.
		prop := concurrentProps[0]
		key := prop.Key()

		val, err := prop.Value(ctx)
		if err != nil {
			if re, ok := errors.AsType[*inertiaprop.RescueError](err); ok {
				d("rescued prop %q: %v", re.Key, re.Err)
				rescuedProps = append(rescuedProps, re.Key)
			} else {
				return nil, nil, fmt.Errorf(
					"inertia: failed to resolve prop %s: %w",
					key, err,
				)
			}
		}

		m[key] = val
	default:
		d("resolving %d props concurrently", len(concurrentProps))

		group := pool.NewGroupContext(ctx)

		// Resolve the rest of concurrent props in pool. Each prop resolution
		// may return an error.
		for _, prop := range concurrentProps {
			group.SubmitErr(func() (Result, error) {
				val, err := prop.Value(ctx)
				return Result{Value: val, Err: err}, nil
			})
		}

		concurrentStart := time.Now()

		results, err := group.Wait()

		r.concurrentPropsDuration.Record(
			ctx,
			float64(time.Since(concurrentStart).Milliseconds()),
		)

		if err != nil {
			return nil, nil, fmt.Errorf("inertia: failed to resolve concurrent props: %w", err)
		}

		for i, result := range results {
			prop := concurrentProps[i]
			key := prop.Key()

			if result.Err != nil {
				if re, ok := errors.AsType[*inertiaprop.RescueError](result.Err); ok {
					d("rescued prop %q: %v", re.Key, re.Err)
					rescuedProps = append(rescuedProps, re.Key)

					continue
				}

				return nil, nil, fmt.Errorf(
					"inertia: failed to resolve prop %s: %w",
					key, result.Err,
				)
			}

			m[key] = result.Value
		}
	}

	if rescuedPropsCount := len(rescuedProps); rescuedPropsCount > 0 {
		r.rescuedPropsCounter.Add(ctx, int64(rescuedPropsCount))
	}

	return m, rescuedProps, nil
}

// shouldSkipOnceProp returns whether the given prop should be skipped
// from being returned to a client.
func shouldSkipOnceProp(prop inertiaprop.Prop, exceptOnceProps []string) bool {
	once, ok := prop.Onceable()
	if !ok || once.Fresh || !slices.Contains(exceptOnceProps, once.Key) {
		return false
	}

	return true
}

// makeDeferredProps creates a map of deferred props that should be resolved
// on the client side.
func makeDeferredProps(req Request, componentName string, props []inertiaprop.Prop) map[string][]string {
	// If the request is partial, then the client already got information
	// about the deferred props in the initial request so we don't need to
	// send them again.
	if req.PartialComponent == componentName {
		return nil
	}

	m := make(map[string][]string)

	for _, prop := range props {
		deferred, ok := prop.Deferrable()
		if !ok {
			continue
		}

		key := prop.Key()
		debug.Assert(deferred != nil, "the property %s must be deferrable", key)

		// If there is no group yet, create a pre-sized slice so the first
		// append doesn't allocate.
		group, ok := m[deferred.Group]
		if !ok {
			group = make([]string, 0, 4)
		}

		group = append(group, key)
		m[deferred.Group] = group
	}

	if len(m) == 0 {
		return nil
	}

	d("deferred props: %d groups", len(m))

	return m
}

// makeOnceProps creates a once piece of the response to the client.
func makeOnceProps(props []inertiaprop.Prop) map[string]OnceProp {
	m := make(map[string]OnceProp)

	for _, prop := range props {
		once, ok := prop.Onceable()
		if !ok {
			continue
		}

		m[once.Key] = OnceProp{
			Prop:      prop.Key(),
			ExpiresAt: once.ExpiresAt,
		}
	}

	if len(m) == 0 {
		return nil
	}

	d("once props: %d", len(m))

	return m
}

// makeMergeProps creates a list of props that should be merged instead of
// being replaced on the client side.
//
// resetKeys contains the prop keys requested to be reset via the
// X-Inertia-Reset header. For props matching a reset key, the merge
// instruction (and any per-path append/prepend/match entries) is suppressed
// while the prop value itself is still returned to the client. Scroll props
// are an exception: their scrollProps metadata (including the reset flag) is
// always emitted by makeScrollProps.
//
// The root-level append/prepend flag is mutually exclusive with path-based
// keys: when a prop has any append or prepend path, the root-level entry is
// suppressed and only the per-path entries are emitted.
func makeMergeProps(props []inertiaprop.Prop, resetKeys []string, scrollMergeIntent string) (mergeProps, error) {
	var m mergeProps

	for _, prop := range props {
		rootKey := prop.Key()
		resetting := len(resetKeys) > 0 && slices.Contains(resetKeys, rootKey)

		// Scrollable is a special case since it is a combination of merge props
		// with a custom handling.
		// The Scrollable prop check must be performed before the Mergeable prop check
		// so that the scroll.Path is used for merge metadata instead of the
		// prop's own Mergeable configuration.
		if scroll, ok := prop.Scrollable(); ok {
			if resetting {
				continue
			}

			switch scrollMergeIntent {
			case inertiaprop.ScrollMergeIntentPrepend:
				m.prepend = append(m.prepend, scroll.Path)
			case inertiaprop.ScrollMergeIntentAppend, "":
				// Default to append when no intent is present (initial load or
				// non-infinite-scroll visits).
				m.append = append(m.append, scroll.Path)
			default:
				return mergeProps{}, fmt.Errorf("invalid scroll merge intent: %s", scrollMergeIntent)
			}

			continue
		}

		if resetting {
			continue
		}

		if merge, ok := prop.Mergeable(); ok {
			hasAppend := false

			for _, key := range merge.AppendKeys {
				if key == "" {
					continue
				}

				m.append = append(m.append, inertiaprop.QualifyPath(rootKey, key))
				hasAppend = true
			}

			hasPrepend := false

			for _, key := range merge.PrependKeys {
				if key == "" {
					continue
				}

				m.prepend = append(m.prepend, inertiaprop.QualifyPath(rootKey, key))
				hasPrepend = true
			}

			switch {
			case merge.DeepMerge:
				m.deepMerge = append(m.deepMerge, rootKey)
			case !hasAppend && !hasPrepend:
				if !merge.Append {
					m.prepend = append(m.prepend, rootKey)
				} else {
					m.append = append(m.append, rootKey)
				}
			}

			for _, key := range merge.MatchOn {
				if key == "" {
					continue
				}

				m.matchOn = append(m.matchOn, inertiaprop.QualifyPath(rootKey, key))
			}
		}
	}

	d("merge props: append=%d prepend=%d deepMerge=%d matchOn=%d",
		len(m.append), len(m.prepend), len(m.deepMerge), len(m.matchOn))

	return m, nil
}

func makeScrollProps(props []inertiaprop.Prop, resetKeys []string) map[string]ScrollProp {
	m := make(map[string]ScrollProp)

	for _, prop := range props {
		scroll, ok := prop.Scrollable()
		if !ok {
			continue
		}

		key := prop.Key()
		m[key] = ScrollProp{
			PageName:     scroll.PageName,
			PreviousPage: scroll.PreviousPage,
			NextPage:     scroll.NextPage,
			CurrentPage:  scroll.CurrentPage,
			Reset:        len(resetKeys) > 0 && slices.Contains(resetKeys, key),
		}
	}

	if len(m) == 0 {
		return nil
	}

	d("scroll props: %d", len(m))

	return m
}

// Result is a generic value-or-error container.
//
// It is used with pond's ResultPool so that every worker's outcome is
// collected as data rather than aborting the group on the first error.
type Result struct {
	Value any
	Err   error
}

// mergeProps holds the properties for the mergeable props.
type mergeProps struct {
	append    []string
	prepend   []string
	deepMerge []string
	matchOn   []string
}
