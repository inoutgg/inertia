# Proposed Changes to `internal/inertiaprotocol/` for Inertia v3 Protocol Compliance

> **Status:** Audit complete — do not apply changes yet.
> **Source:** https://inertiajs.com/docs/v3/core-concepts/the-protocol
> **Target:** `internal/inertiaprotocol/`
>
> **Intentional Omissions:**
> - `deepMergeProps` is explicitly omitted by design.

---

## 1. Page Object — Missing Fields

### 1.1 `rescuedProps` (array)

**Protocol Spec:**

> `rescuedProps` — Array of [deferred prop](https://inertiajs.com/docs/v3/data-props/deferred-props#error-handling) keys that failed to resolve and were rescued server-side. Used by the client to render the `rescue` slot on the `<Deferred>` component.

**Current State:**

The `Page` struct does not include a `rescuedProps` field. There is no logic in `render.go` to track or emit rescued deferred props.

**Proposed Change:**

1. Add `RescuedProps []string` to the `Page` struct with the JSON tag `json:"rescuedProps,omitempty"`.
2. Introduce rescue tracking in the deferred-prop resolution path (likely in `makeProps` or a new helper). When a deferred prop is resolved with `rescue: true` and throws, its key should be added to `rescuedProps` and omitted from `props`.
3. Populate `Page.RescuedProps` in `Render`.

> ⚠️ This may require extending the `inertiaprop.Prop` interface to expose whether a prop is configured for rescue.

---

## 2. Prop Interface — Missing Capabilities

### 2.1 Rescue Support for Deferred Props

**Protocol Spec:**

Deferred props may be configured with `rescue: true`. When they fail to resolve, they should be omitted from `props` and their key added to `rescuedProps`.

**Current State:**

The `Prop` interface exposes:

```go
Deferrable() (*Deferrable, bool)
```

But `Deferrable` only contains `Group string`.

**Proposed Change:**

1. Add `Rescue bool` to the `Deferrable` struct.
2. Add a builder option (e.g., `DeferOpts.Rescue(bool)`) in `inertiaprop`.
3. Update prop resolution logic in `render.go` to catch errors from rescued deferred props, add the key to `rescuedProps`, and omit it from `props`.

---

## 3. Request Struct — Missing Headers

The `Request` struct in `render.go` currently maps these headers:

- `X-Inertia-Partial-Component` → `PartialComponent`
- `X-Inertia-Infinite-Scroll-Merge-Intent` → `ScrollMergeIntent`
- `X-Inertia-Partial-Data` → `PartialData`
- `X-Inertia-Partial-Except` → `PartialExcept`
- `X-Inertia-Reset` → `ResetProps`
- `X-Inertia-Except-Once-Props` → `ExceptOnceProps`

### 3.1 `Purpose` (Prefetch)

**Protocol Spec:**

> `Purpose` — Set to `prefetch` when making [prefetch](https://inertiajs.com/docs/v3/data-props/prefetching) requests.

**Current State:** Missing from `Request`.

**Proposed Change:** Add `Purpose string` to `Request` struct.

---

### 3.2 `X-Inertia-Error-Bag`

**Protocol Spec:**

> `X-Inertia-Error-Bag` — Specifies which error bag to use for [validation errors](https://inertiajs.com/docs/v3/the-basics/validation).

**Current State:** The header constant exists in `inertiaheader`, but `Request` does not expose it.

**Proposed Change:** Add `ErrorBag string` to `Request` struct.

---

### 3.3 `Cache-Control`

**Protocol Spec:**

> `Cache-Control` — Set to `no-cache` for reload requests to prevent serving stale content.

**Current State:** Missing from `Request`.

**Proposed Change:** Add `CacheControl string` to `Request` struct (or `NoCache bool`).

---

### 3.4 Precognition Headers

**Protocol Spec:**

- `Precognition` — Set to `true` to indicate this is a Precognition validation request.
- `Precognition-Validate-Only` — Comma-separated list of field names to validate.

**Current State:** Completely missing from `Request`.

**Proposed Change:**

1. Add `Precognition bool` to `Request`.
2. Add `PrecognitionValidateOnly []string` to `Request`.
3. Add corresponding header constants to `inertiaheader` (if not already present).

---

## 4. Response Headers — Not in Scope for `inertiaprotocol`

The following response headers are defined in the protocol but are typically handled by middleware or the HTTP response writer, not the protocol renderer itself. They are listed here for completeness, but **no changes are proposed** to `inertiaprotocol` for these:

- `X-Inertia`
- `X-Inertia-Location`
- `X-Inertia-Redirect`
- `Vary`
- `Precognition`
- `Precognition-Success`

---

## 5. Other Observations

### 5.1 `errors` Prop Default

**Protocol Spec:**

> `props` — Contains all of the page data along with an `errors` object (defaults to `{}` if there are no errors).

**Current State:**

The `makeProps` function builds `map[string]any` from resolved prop values but does not inject an `errors` key. This may be handled by a higher layer (e.g., validation middleware or framework adapter), but it is worth confirming that the protocol package does not need to guarantee this default.

**Recommendation:** Verify whether the caller is responsible for injecting `errors` into `renderCtx.Props` before `Render` is invoked. If not, `makeProps` should ensure `props["errors"]` is set to `{}` when absent.

---

### 5.2 `version` Type

**Protocol Spec:**

> `version` — `string|number`

**Current State:**

The `Page` struct declares `Version string`. This is acceptable for Go serialization (numbers can be formatted as strings), but if the framework needs to preserve numeric versions, a broader type (e.g., `any`) may be needed.

**Recommendation:** No change required unless numeric version passthrough is a known requirement.

---

## Summary of Files to Modify

| File | Changes |
|------|---------|
| `internal/inertiaprotocol/protocol.go` | Add `RescuedProps` to `Page` |
| `internal/inertiaprotocol/render.go` | Add `makeRescuedProps`, populate new `Page` field, add new `Request` fields |
| `internal/inertiaprop/prop.go` | Add rescue support to `Deferrable` and builder |
| `internal/inertiaheader/header.go` | Add `Precognition`, `Precognition-Validate-Only`, `Purpose`, `Cache-Control` header constants (if not already present) |

---

*End of audit.*
