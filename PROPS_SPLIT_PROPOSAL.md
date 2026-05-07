# Prop Type Split Proposal

## Goal

Split the current catch-all `Prop` type into smaller prop types that match Inertia v3 behavior more directly while preserving the current public constructors and emitted protocol metadata.

The proposed groups are:

- `prop`
- `deferredProp`
- `scrollProp`
- `alwaysProp`
- `optionalProp`

The refactor should preserve behavior, not just names. Inertia's upstream model allows several prop behaviors to intersect, so the Go implementation should keep those intersections available instead of replacing them with mutually exclusive categories.

## Upstream Reference

The Inertia Laravel `3.x` adapter models props as concrete types plus reusable capabilities:

- `AlwaysProp`: always included, bypasses partial reload filtering.
- `OptionalProp`: excluded from the initial response and included only when explicitly requested by partial reloads.
- `DeferProp`: excluded from the initial response and advertised in `deferredProps` by group.
- `ScrollProp`: mergeable paginated prop that emits `scrollProps` metadata.
- `MergeProp`: mergeable prop that emits merge metadata.
- `OnceProp`: remembered client-side and advertised through `onceProps`.

Laravel also composes behavior through interfaces/traits:

- `Deferrable`: `shouldDefer()`, `group()`.
- `IgnoreFirstLoad`: excluded from initial response.
- `Mergeable`: append/prepend/deep-merge/match metadata.
- `Onceable`: once key, expiration, fresh refresh behavior.
- `Rescuable`: deferred error rescue behavior.

This means merge and once behavior are capabilities, not exclusive prop groups. Deferred, optional, scroll, and regular props may need to carry merge or once metadata depending on the constructor/options used.

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

- Existing constructors continue to work: `NewProp`, `NewDeferred`, `NewScroll`, `NewAlways`, `NewOptional`, and `NewOnce`.
- Existing option structs continue to work: `PropOptions`, `DeferredOptions`, `ScrollOptions`, and `OnceOptions`.
- `NewDeferred` keeps the current Go-specific `Concurrent` option.
- `OnceOptions` can also mark a once prop for concurrent resolution.
- `DefaultDeferredGroup` remains `default`.
- `Props` and `Proper` continue to let callers pass one prop or a collection into rendering options.
- Initial responses continue to skip deferred and optional props.
- Partial reloads continue to honor `only` and `except`, except for always props.
- Once props continue to skip already-loaded keys from `X-Inertia-Except-Once-Props` unless explicitly requested or marked fresh.
- Scroll props continue to use `X-Inertia-Infinite-Scroll-Merge-Intent` to choose append or prepend metadata.

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

## Current State

`props.go` stores all behavior flags and metadata on one exported `Prop` struct:

- value resolution: `val`, `valFn`, `concurrent`
- identity: `key`
- deferred: `deferred`, `group`
- optional/deferred initial exclusion: `lazy`
- partial reload filtering: `ignorable`
- merge: `mergeable`, `prepend`, `deepMerge`, `matchOn`
- scroll: `scroll`, `scrollPath`, `scrollMeta`
- once: `once`, `onceKey`, `expiresAt`, `fresh`

The renderer consumes those fields to produce:

- `deferredProps`
- `mergeProps`
- `prependProps`
- `deepMergeProps`
- `matchPropsOn`
- `scrollProps`
- `onceProps`
- filtered `props`

The split should preserve those outputs first, then improve naming and composition.

## Proposed Shape

Keep the public `Proper` and `Props` collection model, and keep `Prop` as a value type. Internally, group related fields into capability structs and expose capability-style methods on `Prop`.

Do not store an internal implementation interface like `impl propImpl` on `Prop`. That design better models polymorphism, but it risks one heap allocation per prop from interface boxing or pointer-backed implementations. The current API stores props as `[]Prop`, so a grouped-field value struct preserves the allocation profile while still making the renderer depend on behavior methods instead of raw fields.

```go
type Prop struct {
	value    value
	scroll   scrollable
	once     onceable
	deferred deferrable
	merge    mergeable
	partial  partial
}
```

The exact field and method names can change during implementation. The important part is that fields are grouped by behavior and renderer decisions ask `Prop` for capabilities through methods.

Example methods:

```go
func (p Prop) key() string
func (p Prop) value(context.Context) (any, error)
func (p Prop) includeOnInitial() bool
func (p Prop) ignorePartialFilters() bool
func (p Prop) deferrable() (deferrable, bool)
func (p Prop) mergeable() (mergeable, bool)
func (p Prop) scrollable() (scrollable, bool)
func (p Prop) onceable() (onceable, bool)
func (p Prop) resolveConcurrently() bool
```

The capability method names should read as predicates while still returning the metadata needed by the renderer. The boolean reports whether the capability is enabled for that prop.

Avoid this shape in renderer code:

```go
switch p.impl.(type) {
case deferredProp:
case scrollProp:
case alwaysProp:
}
```

That makes intersections like deferred + merge + once harder to preserve. Prefer small assertions instead:

```go
if meta, ok := p.mergeable(); ok {
	// collect merge metadata
}

if meta, ok := p.onceable(); ok {
	// collect once metadata
}
```

This is capability-oriented without interface-backed storage.

## Allocation Considerations

The grouped-field value design should not add per-prop allocations compared to the current `Prop` struct.

Avoid this shape:

```go
type Prop struct {
	impl propImpl
}
```

Risks with `impl propImpl`:

- `Prop{impl: &deferredProp{...}}` usually adds one heap allocation per prop.
- `Prop{impl: deferredProp{...}}` can still allocate when boxing a non-trivial struct into an interface.
- Interface method dispatch is harder for the compiler to inline.
- Escape analysis becomes less predictable around `Props []Prop`.

Preferred shape:

```go
type Prop struct {
	value    value
	scroll   scrollable
	once     onceable
	deferred deferrable
	merge    mergeable
	partial  partial
}
```

Benefits:

- Keeps `Prop` copyable and storable in `[]Prop` without per-prop heap allocation.
- Keeps renderer logic readable through capability methods.
- Keeps invalid upstream combinations under constructor control.
- Avoids direct renderer dependency on individual fields like `merge.enabled` or `deferred.enabled`.
- Avoids a separate `kind` field that could become a second source of truth.

Tradeoff:

- `Prop` remains a tagged value with grouped capability fields rather than separate runtime concrete implementations. This is acceptable because third-party custom prop implementations are not a goal for the first pass.

## Shared Building Blocks

Use small embedded structs for capabilities that are reused across prop types.

### Value

Required by every prop group.

Fields:

- `key string`
- `val any`
- `valFn Lazy`
- `concurrent bool`

Behavior:

- Resolve `valFn` when present.
- Return `val` otherwise.
- Mark resolution as concurrent independently of whether the prop is deferred or onceable.

### Merge Metadata

Reusable by `prop`, `deferredProp`, and `scrollProp`.

Fields:

- `mergeable bool`
- `prepend bool`
- `deepMerge bool`
- `matchOn []string`
- future: append/prepend paths if path-specific merging is added.

Behavior:

- Emits `mergeProps`, `prependProps`, `deepMergeProps`, and `matchPropsOn`.
- Skips metadata when the prop is reset via `X-Inertia-Reset`.

### Once Metadata

Reusable by `prop`, `deferredProp`, and `optionalProp`.

Fields:

- `once bool`
- `onceKey string`
- `expiresAt *int64`
- `fresh bool`
- `concurrent bool` in `OnceOptions`, stored on the value capability.

Behavior:

- Emits `onceProps` as `onceKey -> { prop, expiresAt }`.
- Skips resolution when `X-Inertia-Except-Once-Props` contains the key unless explicitly requested or marked fresh.

## Prop Groups

### `prop`

Represents normal page data.

Required behavior:

- Included on initial responses.
- Included on partial reloads only when selected by `only`/`except` filters.
- May be eager or lazy depending on whether the value implements `Lazy` or is wrapped by constructor options.

Capabilities:

- Merge metadata.
- Once metadata.

Constructor mapping:

- `NewProp` returns this group.
- Existing `PropOptions` should continue to configure merge and once behavior.

### `deferredProp`

Represents data loaded after the initial page render.

Required behavior:

- Excluded from initial responses.
- Advertised through `deferredProps[group]`.
- Default group is `default`.
- Resolved only when requested through partial reloads.

Capabilities:

- Merge metadata.
- Once metadata.
- Concurrent resolution through the shared value capability, preserving the current Go-specific option.
- Future rescue metadata.

Constructor mapping:

- `NewDeferred` returns this group.
- Existing `DeferredOptions` should continue to configure group, merge, once, and concurrency.

Upstream parity gap:

- Laravel supports `rescue: true` and emits `rescuedProps` when a deferred prop fails and is rescued. Add this only if the renderer is updated to support rescued resolution.

### `scrollProp`

Represents paginated data for Inertia v3 infinite scroll.

Required behavior:

- Mergeable by default.
- Emits `scrollProps[key]` metadata.
- Uses the configured wrapper path for merge metadata, defaulting to `data`.
- Uses `X-Inertia-Infinite-Scroll-Merge-Intent` to decide append vs prepend.

Required metadata:

- `pageName`
- `previousPage`
- `nextPage`
- `currentPage`

Capabilities:

- Merge metadata.
- Lazy value resolution if the value implements `Lazy`.
- Future deferrable behavior if parity with Laravel `ScrollProp implements Deferrable` is needed.

Constructor mapping:

- `NewScroll` returns this group.
- `NewScrollOptions` should remain the type-safe metadata helper.

Upstream parity gap:

- Laravel `scrollProps` includes `reset` when the prop is listed in `X-Inertia-Reset`. Add this when touching renderer metadata.

### `alwaysProp`

Represents data that must be included in every response.

Required behavior:

- Included on initial responses.
- Included on partial reloads even when omitted by `only` or present in `except`.
- Does not behave as deferred, optional, mergeable, or onceable upstream.

Constructor mapping:

- `NewAlways` returns this group.

Notes:

- This should remain intentionally small.
- It should support value/callable resolution because Laravel `AlwaysProp` resolves callables.

### `optionalProp`

Represents data that is never sent unless explicitly requested.

Required behavior:

- Excluded from initial responses.
- Included only when explicitly requested by partial reload `only`.
- Not included by default partial reloads unless selected.

Capabilities:

- Once metadata.

Constructor mapping:

- `NewOptional` returns this group.

## Public API Compatibility

Keep existing constructors initially:

- `NewProp`
- `NewDeferred`
- `NewScroll`
- `NewAlways`
- `NewOptional`
- `NewOnce`

Options can remain unchanged for the first refactor:

- `PropOptions`
- `DeferredOptions`
- `ScrollOptions`
- `OnceOptions`

The return type decision is to keep returning `Prop` from all constructors. Constructors should populate grouped capability fields according to the prop group they create.

Do not expose concrete split types in the first pass. The names `prop`, `deferredProp`, `scrollProp`, `alwaysProp`, and `optionalProp` are conceptual groups represented by the capability fields each constructor enables.

## Renderer Changes

The renderer should stop checking fields like `prop.lazy`, `prop.ignorable`, `prop.deferred`, and `prop.mergeable` directly.

Replace those checks with capability methods:

- initial response inclusion
- partial filter behavior
- deferred capability and metadata
- merge capability and metadata
- scroll capability and metadata
- once capability and metadata
- concurrent resolution

This keeps the behavior table explicit without adding interface assertions or type switches to hot paths.

## Behavior Matrix

| Group | Initial Response | Partial Reload Default | Partial Reload `only` | Bypasses Partial Filters | Deferred Metadata | Merge Metadata | Scroll Metadata | Once Metadata |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `prop` | yes | yes, unless excluded | yes, when selected | no | no | optional | no | optional |
| `deferredProp` | no | no, unless selected | yes, when selected | no | yes | optional | no | optional |
| `scrollProp` | yes | yes, unless excluded | yes, when selected | no | future optional | yes | yes | no |
| `alwaysProp` | yes | yes | yes | yes | no | no | no | no |
| `optionalProp` | no | no | yes, when selected | no | no | no | no | optional |

## Deliverable Steps

Each step should be independently reviewable and committed before starting the next one.

1. Document the target design and implementation sequence.
   Commit this proposal and the deliverable breakdown before changing runtime code.

2. Introduce grouped capability fields on `Prop`.
   Add `value`, `partial`, `deferrable`, `mergeable`, `scrollable`, and `onceable` structs. Move existing fields into those groups without changing renderer behavior.

3. Add capability-style methods on `Prop`.
   Add methods such as `deferrable()`, `mergeable()`, `scrollable()`, and `onceable()` that return metadata plus enabled status. Keep direct field reads in the renderer until the methods are in place and tested.

4. Update constructors to populate capabilities.
   Make `NewProp`, `NewDeferred`, `NewScroll`, `NewAlways`, `NewOptional`, and `NewOnce` configure the grouped fields according to the preservation matrix.

5. Update renderer helpers to consume capability methods.
   Replace direct checks like `prop.lazy`, `prop.ignorable`, `prop.deferred`, and `prop.mergeable` with behavior methods.

6. Add or update behavior tests.
   Cover initial exclusion, partial filtering, always-prop bypass, deferred metadata, merge metadata, scroll metadata, and once metadata using the new grouped representation.

7. Run the full test suite and clean up naming.
   Remove obsolete field names, simplify comments, and verify the implementation does not add avoidable allocations.

8. Handle parity follow-ups separately.
   Add `rescuedProps`, scroll `reset`, scroll deferring, or richer path-specific merge behavior only in follow-up changes unless a previous step requires them.

## Non-Goals

- Do not add nested prop resolution as part of this refactor.
- Do not expose all split concrete types publicly in the first pass.
- Do not add backward compatibility shims beyond preserving existing constructors and option structs.
- Do not expand path-specific merge behavior unless the renderer is being updated for that feature at the same time.

## Open Questions

- Should `NewOnce` remain a dedicated constructor, or should once be represented only through options on other prop constructors?
- Should `scrollProp` support deferring immediately for Laravel parity, or wait until there is a concrete use case?
- Should deferred `rescue` be added during this split or handled as a separate protocol parity change?
