package inertiaprotocol

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/alitto/pond/v2"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/sliceutil"
)

type Request struct {
	URL               string
	PartialComponent  string
	ScrollMergeIntent string
	PartialData       []string
	PartialExcept     []string
	ResetProps        []string
	ExceptOnceProps   []string
}

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

func Render(ctx context.Context, req Request, renderCtx Context) (*Page, error) {
	props, rescuedProps, err := makeProps(ctx, req, renderCtx.Component, renderCtx.Props, renderCtx.Concurrency)
	if err != nil {
		return nil, err
	}

	mergeProps := makeMergeProps(
		renderCtx.Props,
		req.ResetProps,
		req.ScrollMergeIntent,
	)

	return &Page{
		Component:     renderCtx.Component,
		Props:         props,
		DeferredProps: makeDeferredProps(req, renderCtx.Component, renderCtx.Props),
		ScrollProps:   makeScrollProps(renderCtx.Props),
		OnceProps:     makeOnceProps(renderCtx.Props),
		MergeProps:    mergeProps.append,
		PrependProps:  mergeProps.prepend,
		MatchPropsOn:  mergeProps.matchOn,
		SharedProps: sliceutil.Map(renderCtx.SharedProps,
			func(p inertiaprop.Prop) string { return p.Key() }),
		RescuedProps:     rescuedProps,
		URL:              req.URL,
		Version:          renderCtx.Version,
		PreserveFragment: renderCtx.PreserveFragment,
		ClearHistory:     renderCtx.ClearHistory,
		EncryptHistory:   renderCtx.EncryptHistory,
	}, nil
}

func makeProps(
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
		return !shouldSkipOnceProp(p, req.ExceptOnceProps, nil)
	})

	m := make(map[string]any, len(props))

	var rescuedProps []string

	for _, prop := range props {
		// Skip deferred and optional props on the first render.
		if prop.IsFirstLoadIgnorable() {
			continue
		}

		val, err := prop.Value(ctx)
		if err != nil {
			var re *inertiaprop.RescueError
			if errors.As(err, &re) {
				rescuedProps = append(rescuedProps, re.Key)
				continue
			}

			return nil, nil, fmt.Errorf("inertia: failed to resolve prop %s: %w", prop.Key(), err)
		}

		m[prop.Key()] = val
	}

	return m, rescuedProps, nil
}

func resolvePartialComponentRequest(
	ctx context.Context,
	props []inertiaprop.Prop,
	whitelist, blacklist, exceptOnceProps []string,
	concurrency int,
) (map[string]any, []string, error) {
	props = sliceutil.Filter(props, func(p inertiaprop.Prop) bool {
		return !shouldSkipOnceProp(p, exceptOnceProps, whitelist)
	})

	m := make(map[string]any, len(props))
	concurrentProps := make([]inertiaprop.Prop, 0, len(props))

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
				var re *inertiaprop.RescueError
				if errors.As(err, &re) {
					rescuedProps = append(rescuedProps, re.Key)
					continue
				}

				return nil, nil, fmt.Errorf("inertia: failed to resolve prop %s: %w", prop.Key(), err)
			}

			m[key] = val
		}
	}

	if len(concurrentProps) > 0 {
		pool := pond.NewResultPool[result[any, error]](concurrency)
		group := pool.NewGroupContext(ctx)

		for _, prop := range concurrentProps {
			group.SubmitErr(func() (result[any, error], error) {
				val, err := prop.Value(ctx)

				return result[any, error]{value: val, err: err}, nil
			})
		}

		results, err := group.Wait()
		if err != nil {
			return nil, nil, fmt.Errorf("inertia: failed to resolve concurrent props: %w", err)
		}

		for i, r := range results {
			prop := concurrentProps[i]

			if r.err != nil {
				var re *inertiaprop.RescueError
				if errors.As(r.err, &re) {
					rescuedProps = append(rescuedProps, re.Key)
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

func shouldSkipOnceProp(prop inertiaprop.Prop, exceptOnceProps, whitelist []string) bool {
	once, ok := prop.Onceable()
	if !ok || once.Fresh || !slices.Contains(exceptOnceProps, once.Key) {
		return false
	}

	return !slices.Contains(whitelist, prop.Key())
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

	m := make(map[string][]string, len(props))

	for _, prop := range props {
		deferred, ok := prop.Deferrable()
		if !ok {
			continue
		}

		if _, ok := m[deferred.Group]; !ok {
			m[deferred.Group] = []string{}
		}

		m[deferred.Group] = append(m[deferred.Group], prop.Key())
	}

	return m
}

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

	return m
}

// makeMergeProps creates a list of props that should be merged instead of
// being replaced on the client side.
func makeMergeProps(props []inertiaprop.Prop, blacklist []string, scrollMergeIntent string) mergeProps {
	var m mergeProps

	for _, prop := range props {
		merge, ok := prop.Mergeable()
		if len(blacklist) > 0 && slices.Contains(blacklist, prop.Key()) || !ok {
			continue
		}

		if scroll, ok := prop.Scrollable(); ok {
			if scrollMergeIntent == inertiaheader.HeaderValueScrollMergeIntentPrepend {
				m.prepend = append(m.prepend, scroll.Path)
			} else {
				m.append = append(m.append, scroll.Path)
			}

			continue
		}

		switch {
		case merge.Prepend:
			m.prepend = append(m.prepend, prop.Key())
		case merge.Append:
			m.append = append(m.append, prop.Key())
		}

		m.addMergeKeys(prop.Key(), merge.AppendKeys, &m.append)
		m.addMergeKeys(prop.Key(), merge.PrependKeys, &m.prepend)
	}

	return m
}

func (m *mergeProps) addMergeKeys(propKey string, keys []inertiaprop.MergeKey, props *[]string) {
	for _, key := range keys {
		path := propKey
		if key.Key != "" {
			path = QualifyPath(propKey, key.Key)
		}

		*props = append(*props, path)
		if key.MatchOn != "" {
			m.matchOn = append(m.matchOn, QualifyPath(path, key.MatchOn))
		}
	}
}

func makeScrollProps(props []inertiaprop.Prop) map[string]ScrollProp {
	m := make(map[string]ScrollProp)

	for _, prop := range props {
		if scroll, ok := prop.Scrollable(); ok {
			m[prop.Key()] = ScrollProp{
				PageName:     scroll.PageName,
				PreviousPage: scroll.PreviousPage,
				NextPage:     scroll.NextPage,
				CurrentPage:  scroll.CurrentPage,
			}
		}
	}

	if len(m) == 0 {
		return nil
	}

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

// result is a generic value-or-error container. It is used with pond's
// ResultPool so that every worker's outcome is collected as data rather
// than aborting the group on the first error.
type result[R any, E error] struct {
	value R
	err   E
}

type mergeProps struct {
	append  []string
	prepend []string
	matchOn []string
}
