package inertiaprotocol

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/alitto/pond/v2"
	"go.inout.gg/foundations/debug"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/sliceutil"
)

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
	Component        string
	Version          string
	Props            []inertiaprop.Prop
	SharedProps      []inertiaprop.Prop
	PreserveFragment bool
	ClearHistory     bool
	EncryptHistory   bool
	Concurrency      int
}

// Render returns a rendered page for the given request and context.
//
// Render is protocol agnostic (meaning it does not contain any HTTP specific logic)
// and all the protocol-specific logic must be handled by the caller.
func Render(ctx context.Context, req Request, renderCtx Context) (*Page, error) {
	props, rescuedProps, err := resolveProps(ctx, req, renderCtx.Component, renderCtx.Props, renderCtx.Concurrency)
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
		Component:     renderCtx.Component,
		Props:         props,
		DeferredProps: makeDeferredProps(req, renderCtx.Component, renderCtx.Props),
		ScrollProps:   makeScrollProps(renderCtx.Props),
		OnceProps:     makeOnceProps(renderCtx.Props),
		MergeProps:    mergeProps.append,
		PrependProps:  mergeProps.prepend,
		MatchPropsOn:  mergeProps.matchOn,
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
func resolveProps(
	ctx context.Context,
	req Request,
	componentName string,
	props []inertiaprop.Prop,
	concurrency int,
) (map[string]any, []string, error) {
	// If the request is a partial, we need to filter the props.
	if req.PartialComponent == componentName {
		return resolvePartialComponentRequest(
			ctx,
			props,
			req.PartialData,
			req.PartialExcept,
			req.ExceptOnceProps,
			concurrency,
		)
	}

	props = sliceutil.Filter(props, func(p inertiaprop.Prop) bool {
		return !shouldSkipOnceProp(p, req.ExceptOnceProps)
	})

	m := make(map[string]any, len(props))

	for _, prop := range props {
		// Skip deferred and optional props on the first render.
		if prop.IsFirstLoadIgnorable() {
			continue
		}

		val, err := prop.Value(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("inertia: failed to resolve prop %s: %w", prop.Key(), err)
		}

		m[prop.Key()] = val
	}

	return m, nil, nil
}

func resolvePartialComponentRequest(
	ctx context.Context,
	props []inertiaprop.Prop,
	whitelist, blacklist, exceptOnceProps []string,
	concurrency int,
) (map[string]any, []string, error) {
	props = sliceutil.Filter(props, func(p inertiaprop.Prop) bool {
		if shouldSkipOnceProp(p, exceptOnceProps) {
			return len(whitelist) > 0 && slices.Contains(whitelist, p.Key())
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

		if !prop.BypassPartialFilters() {
			// It should be fine to go through slices here, as the number of props is expected to be small.
			if len(whitelist) > 0 && !slices.Contains(whitelist, key) ||
				len(blacklist) > 0 && slices.Contains(blacklist, key) {
				continue
			}
		}

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
					rescuedProps = append(rescuedProps, re.Key)
				} else {
					return nil, nil, fmt.Errorf(
						"inertia: failed to resolve prop %s: %w",
						prop.Key(),
						err,
					)
				}
			}

			m[key] = val
		}
	}

	if len(concurrentProps) > 0 {
		pool := pond.NewResultPool[result[any, error]](concurrency)
		group := pool.NewGroupContext(ctx)

		// Resolve the rest of concurrent props in pool. Each prop resolution
		// may return an error.
		for _, prop := range concurrentProps {
			group.SubmitErr(func() (result[any, error], error) {
				return newResult(prop.Value(ctx)), nil
			})
		}

		results, err := group.Wait()
		if err != nil {
			return nil, nil, fmt.Errorf("inertia: failed to resolve concurrent props: %w", err)
		}

		for i, r := range results {
			prop := concurrentProps[i]

			if r.err != nil {
				if err, ok := errors.AsType[*inertiaprop.RescueError](r.err); ok {
					rescuedProps = append(rescuedProps, err.Key)
					continue
				}

				return nil, nil, fmt.Errorf(
					"inertia: failed to resolve prop %s: %w",
					prop.Key(),
					r.err,
				)
			}

			m[prop.Key()] = r.value
		}
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

	props = sliceutil.Filter(props, func(prop inertiaprop.Prop) bool {
		_, ok := prop.Deferrable()
		return ok
	})
	if len(props) == 0 {
		return nil
	}

	return sliceutil.Reduce(props, func(prop inertiaprop.Prop, m map[string][]string) map[string][]string {
		deferred, _ := prop.Deferrable()

		debug.Assert(deferred != nil, "the property %s must be deferrable", prop.Key())

		// if there is no group, create a new one.
		if _, ok := m[deferred.Group]; !ok {
			m[deferred.Group] = []string{}
		}

		m[deferred.Group] = append(m[deferred.Group], prop.Key())

		return m
	}, make(map[string][]string, len(props)))
}

// makeOnceProps creates a once piece of the response to the client.
func makeOnceProps(props []inertiaprop.Prop) map[string]OnceProp {
	props = sliceutil.Filter(props, func(prop inertiaprop.Prop) bool {
		_, ok := prop.Onceable()
		return ok
	})
	if len(props) == 0 {
		return nil
	}

	return sliceutil.Reduce(props, func(prop inertiaprop.Prop, m map[string]OnceProp) map[string]OnceProp {
		if once, ok := prop.Onceable(); ok {
			m[once.Key] = OnceProp{
				Prop:      prop.Key(),
				ExpiresAt: once.ExpiresAt,
			}
		}

		return m
	}, make(map[string]OnceProp, len(props)))
}

// makeMergeProps creates a list of props that should be merged instead of
// being replaced on the client side.
func makeMergeProps(props []inertiaprop.Prop, blacklist []string, scrollMergeIntent string) (mergeProps, error) {
	var m mergeProps

	for _, prop := range props {
		if len(blacklist) > 0 && slices.Contains(blacklist, prop.Key()) {
			continue
		}

		// Scrollable is a special case since it is a combination of merge props
		// with a custom handling.
		// The Scrollable prop check must be performed before the Mergeable prop check
		// since the inertiascoll.Prop has no Mergeable capability.
		if scroll, ok := prop.Scrollable(); ok {
			switch scrollMergeIntent {
			case inertiaprop.ScrollIntentPrepend:
				m.prepend = append(m.prepend, scroll.Path)
			case inertiaprop.ScrollIntentAppend:
				m.append = append(m.append, scroll.Path)
			default:
				return mergeProps{}, fmt.Errorf("invalid scroll merge intent: %s", scrollMergeIntent)
			}

			continue
		}

		if merge, ok := prop.Mergeable(); ok {
			key := prop.Key()

			if merge.Prepend {
				m.prepend = append(m.prepend, key)
			} else if merge.Append {
				m.append = append(m.append, key)
			}

			for _, k := range merge.AppendKeys {
				m.addMergeKey(key, k, false)
			}

			for _, k := range merge.PrependKeys {
				m.addMergeKey(key, k, true)
			}
		}
	}

	return m, nil
}

func (m *mergeProps) addMergeKey(propKey string, key inertiaprop.MergeKey, prepend bool) {
	path := propKey
	if key.Key != "" {
		path = QualifyPath(propKey, key.Key)
	}

	if key.MatchOn != "" {
		m.matchOn = append(m.matchOn, QualifyPath(path, key.MatchOn))
	}

	if prepend {
		m.prepend = append(m.prepend, path)
	} else {
		m.append = append(m.append, path)
	}
}

func makeScrollProps(props []inertiaprop.Prop) map[string]ScrollProp {
	props = sliceutil.Filter(props, func(prop inertiaprop.Prop) bool {
		_, ok := prop.Scrollable()
		return ok
	})
	if len(props) == 0 {
		return nil
	}

	m := sliceutil.Reduce(props, func(prop inertiaprop.Prop, m map[string]ScrollProp) map[string]ScrollProp {
		scroll, _ := prop.Scrollable()
		m[prop.Key()] = ScrollProp{
			PageName:     scroll.PageName,
			PreviousPage: scroll.PreviousPage,
			NextPage:     scroll.NextPage,
			CurrentPage:  scroll.CurrentPage,
		}

		return m
	}, make(map[string]ScrollProp))

	return m
}

// QualifyPath prefixes path with propKey if path is non-empty, not already equal to propKey,
// and does not already start with propKey+".".
func QualifyPath(propKey, path string) string {
	if path == "" || strings.HasPrefix(path, propKey+".") || path == propKey {
		return path
	}

	return propKey + "." + path
}

// result is a generic value-or-error container.
//
// use newResult to create a result value.
//
// It is used with pond's ResultPool so that every worker's outcome is
// collected as data rather than aborting the group on the first error.
type result[R any, E error] struct {
	value R
	err   E
}

func newResult[R any, E error](value R, err E) result[R, E] {
	return result[R, E]{
		value: value,
		err:   err,
	}
}

// mergeProps holds the properties for the mergeable props.
type mergeProps struct {
	append  []string
	prepend []string
	matchOn []string
}
