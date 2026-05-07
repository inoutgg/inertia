# Prop Type Split Proposal

## Goal

Replace the current catch-all `Prop` struct with a `Prop` interface, concrete prop types, and capability methods that match Inertia v3 behavior. Compatibility with the current `Prop` struct API is not a constraint.

The target prop types are:

- `standardProp`
- `deferredProp`
- `scrollProp`
- `alwaysProp`
- `optionalProp`
- `onceProp`

Shared behaviors should be modeled as capabilities, not as fields on every prop type.

## Upstream Reference

The Inertia Laravel `3.x` adapter uses concrete prop classes plus reusable capabilities:

- `AlwaysProp`: always included, bypasses partial reload filtering.
- `OptionalProp`: excluded from the initial response and included only when explicitly requested by partial reloads.
- `DeferProp`: excluded from the initial response and advertised in `deferredProps` by group.
- `ScrollProp`: mergeable paginated prop that emits `scrollProps` metadata.
- `MergeProp`: mergeable prop that emits merge metadata.
- `OnceProp`: remembered client-side and advertised through `onceProps`.

Laravel composes behavior through interfaces/traits:

- `Deferrable`: `shouldDefer()`, `group()`.
- `IgnoreFirstLoad`: excluded from initial response.
- `Mergeable`: append/prepend/deep-merge/match metadata.
- `Onceable`: once key, expiration, fresh refresh behavior.
- `Rescuable`: deferred error rescue behavior.

## Behavior To Preserve

Preserve these upstream capability intersections:

| Intersection | Reason |
| --- | --- |
| deferred + merge | Laravel `DeferProp` implements `Mergeable`; docs show deferred merge/deep merge. |
| deferred + once | Laravel `DeferProp` implements `Onceable`; docs show `defer(...)->once()`. |
| deferred + merge + once | Combination follows from `DeferProp` being both mergeable and onceable. |
| deferred + rescue | Laravel `DeferProp` implements `Rescuable`; rescued failures emit `rescuedProps`. |
| optional + once | Laravel `OptionalProp` implements `Onceable`; docs show `optional(...)->once()`. |
| merge + once | Laravel `MergeProp` implements `Onceable`; docs show `merge(...)->once()`. |
| scroll + merge | Laravel `ScrollProp` implements `Mergeable`; infinite scroll relies on merge metadata. |
| scroll + deferred | Laravel `ScrollProp` implements `Deferrable`; scroll is not deferred by default but can be deferred. |
| scroll + deferred + merge | Combination follows from `ScrollProp` being both deferrable and mergeable. |

Preserve these non-intersections unless there is a concrete reason to expand beyond upstream behavior:

| Non-Intersection | Reason |
| --- | --- |
| always + merge | Laravel `AlwaysProp` is not `Mergeable`. |
| always + once | Laravel `AlwaysProp` is not `Onceable`. |
| always + deferred | Laravel `AlwaysProp` is not `Deferrable`. |
| always + optional | `AlwaysProp` bypasses partial filters; `OptionalProp` is only sent when requested. |
| optional + merge | Laravel `OptionalProp` is not `Mergeable`. |
| optional + deferred | Laravel `OptionalProp` is `IgnoreFirstLoad`, not `Deferrable`, so it does not emit `deferredProps`. |
| scroll + once | Laravel `ScrollProp` is not `Onceable`. |

Preserve these current Go behaviors:

- Constructors continue to exist: `NewProp`, `NewDeferred`, `NewScroll`, `NewAlways`, `NewOptional`, and `NewOnce`.
- Option structs continue to exist: `PropOptions`, `DeferredOptions`, `ScrollOptions`, and `OnceOptions`.
- `DefaultDeferredGroup` remains `default`.
- Initial responses skip deferred and optional props.
- Partial reloads honor `only` and `except`, except for always props.
- Once props skip already-loaded keys from `X-Inertia-Except-Once-Props` unless explicitly requested or marked fresh.
- Scroll props use `X-Inertia-Infinite-Scroll-Merge-Intent` to choose append or prepend metadata.
- `DeferredOptions.Concurrent` and `OnceOptions.Concurrent` mark lazy resolution as concurrent.

Preserve these protocol outputs:

- `props`
- `deferredProps`
- `mergeProps`
- `prependProps`
- `deepMergeProps`
- `matchPropsOn`
- `scrollProps`
- `onceProps`
- `sharedProps`

Consider adding these upstream v3 parity outputs during or after the split:

- `rescuedProps` for rescued deferred resolution failures.
- `scrollProps[key].reset` when the prop key is present in `X-Inertia-Reset`.

## Proposed Shape

`Prop` should become an interface:

```go
type Prop interface {
	Key() string
	Value(context.Context) (any, error)
	IgnoreFirstLoad() bool
	BypassPartialFilters() bool
	Deferrable() (deferrable, bool)
	Mergeable() (mergeable, bool)
	Scrollable() (scrollable, bool)
	Onceable() (onceable, bool)
	Concurrent() bool
}
```

Collections continue to use `Proper`, but now hold interface values:

```go
type Proper interface {
	Props() []Prop
	Len() int
}

type Props []Prop
```

Constructors return `Prop` interface values backed by pointers to concrete implementations:

```go
func NewProp(...) Prop
func NewDeferred(...) Prop
func NewScroll(...) Prop
func NewAlways(...) Prop
func NewOptional(...) Prop
func NewOnce(...) Prop
```

This intentionally breaks compatibility with code that expected `Prop` to be a concrete struct.

## Base Value

All concrete prop types should embed a small value resolver:

```go
type baseProp struct {
	key   string
	val   any
	valFn Lazy
}

func (p baseProp) Key() string
func (p baseProp) Value(context.Context) (any, error)
func (p baseProp) IgnoreFirstLoad() bool
func (p baseProp) BypassPartialFilters() bool
func (p baseProp) Deferrable() (deferrable, bool)
func (p baseProp) Mergeable() (mergeable, bool)
func (p baseProp) Scrollable() (scrollable, bool)
func (p baseProp) Onceable() (onceable, bool)
func (p baseProp) Concurrent() bool
```

`baseProp` supplies disabled defaults for all capability methods. Concrete prop types override only the capabilities they enable.

Concurrent resolution is a capability of lazy resolution, not a deferred-only field. Store it on concrete types that expose concurrency, or use a small embedded `concurrent` capability.

## Capability Interfaces

Capability methods live on `Prop` itself instead of separate assertion-only interfaces. Renderer logic should avoid switching on concrete prop type and ask each prop for the relevant capability:

```go
if prop.IgnoreFirstLoad() {
	continue
}

if merge, ok := prop.Mergeable(); ok {
	// collect merge metadata
}
```

## Concrete Types

### `standardProp`

Represents normal page data.

Fields/capabilities:

- `baseProp`
- optional `mergeable`
- optional `onceable`
- optional `concurrent`

Behavior:

- Included on initial responses.
- Included on partial reloads only when selected by `only`/`except` filters.

### `deferredProp`

Represents data loaded after the initial page render.

Fields/capabilities:

- `baseProp`
- `deferrable`
- `IgnoreFirstLoad()` override
- optional `mergeable`
- optional `onceable`
- optional `concurrent`
- future `rescuable`

Behavior:

- Excluded from initial responses.
- Advertised through `deferredProps[group]`.
- Default group is `default`.
- Resolved only when requested through partial reloads.

### `scrollProp`

Represents paginated data for Inertia v3 infinite scroll.

Fields/capabilities:

- `baseProp`
- `scrollable`
- `mergeable`
- optional `deferrable` in a follow-up if scroll deferring is implemented.

Behavior:

- Mergeable by default.
- Emits `scrollProps[key]` metadata.
- Uses the configured wrapper path for merge metadata, defaulting to `data`.
- Uses `X-Inertia-Infinite-Scroll-Merge-Intent` to decide append vs prepend.

### `alwaysProp`

Represents data that must be included in every response.

Fields/capabilities:

- `baseProp`
- `BypassPartialFilters()` override

Behavior:

- Included on initial responses.
- Included on partial reloads even when omitted by `only` or present in `except`.
- Does not implement merge, once, deferred, or optional capabilities.

### `optionalProp`

Represents data that is never sent unless explicitly requested.

Fields/capabilities:

- `baseProp`
- `IgnoreFirstLoad()` override
- optional `onceable`
- optional `concurrent`

Behavior:

- Excluded from initial responses.
- Included only when explicitly requested by partial reload `only`.
- Not included by default partial reloads unless selected.

### `onceProp`

Represents a standalone once prop.

Fields/capabilities:

- `baseProp`
- `onceable`
- optional `concurrent`

Behavior:

- Included on initial responses unless already loaded by the client.
- Advertised through `onceProps`.

## Metadata Types

Keep metadata structs small and unexported unless there is a reason to expose them.

```go
type deferrable struct {
	group string
}

type mergeable struct {
	matchOn   []string
	deepMerge bool
	prepend   bool
	append    bool
}

type scrollable struct {
	PageName     string
	PreviousPage any
	NextPage     any
	CurrentPage  any
	path         string
}

type onceable struct {
	expiresAt *int64
	key       string
	fresh     bool
}
```

## Merge Option Validation

`Merge`, `Prepend`, and `DeepMerge` are mutually exclusive root merge modes.

Rules:

- `Merge` means append and emits `mergeProps`.
- `Prepend` emits `prependProps` and does not require `Merge`.
- `DeepMerge` emits `deepMergeProps` and does not require `Merge`.
- `MatchOn` can be used with any single merge mode.
- Invalid combinations are rejected by `validate()` methods on `PropOptions` and `DeferredOptions`.

## Allocation Considerations

The split design stores concrete props behind interfaces, so it may allocate more than the current single value struct. This is acceptable because API compatibility and zero-allocation struct storage are not the primary goals.

Mitigations:

- Return concrete pointers as `Prop` at constructor boundaries.
- Keep concrete prop structs small.
- Use embedded value/capability structs to avoid repeated fields.
- Benchmark after the split if prop creation becomes hot.

## Renderer Changes

Renderer helpers should consume `[]Prop` interface values and call capability methods directly:

- `IgnoreFirstLoad` for initial response exclusion.
- `BypassPartialFilters` for always props.
- `Deferrable` for `deferredProps`.
- `Mergeable` for merge metadata.
- `Scrollable` for `scrollProps`.
- `Onceable` for `onceProps` and loaded-once skipping.
- `Concurrent` for concurrent lazy resolution.

The renderer should not use concrete type switches for normal protocol behavior.

## Deliverable Steps

Each step should be independently reviewable and committed before starting the next one.

1. Convert `Prop` from a concrete struct to an interface and update `Props` to hold `[]Prop`.
2. Add `baseProp` and concrete prop types with constructor mappings.
3. Add capability methods and metadata structs.
4. Move constructor option handling and validation onto the concrete types.
5. Update renderer helpers to consume capability methods.
6. Update struct parsing and tests for the new constructor return types.
7. Run the full test suite and clean up obsolete value-struct code.
8. Handle parity follow-ups separately: `rescuedProps`, scroll `reset`, scroll deferring, and richer path-specific merge behavior.

## Non-Goals

- Do not add nested prop resolution as part of this refactor.
- Do not preserve compatibility with code that directly accesses fields on `Prop`.
- Do not use concrete type switches for renderer protocol behavior.
- Do not add path-specific merge behavior unless the renderer is updated for that feature at the same time.
