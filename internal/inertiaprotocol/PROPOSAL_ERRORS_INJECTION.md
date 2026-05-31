# Proposal: `errors` Prop Injection in `inertiaprotocol`

> **Status:** Proposed — awaiting decision before implementation.
> **Scope:** `internal/inertiaprotocol/render.go`, `internal/inertiaprotocol/protocol.go`

---

## 1. Problem Statement

The Inertia v3 protocol requires every page object to contain `props.errors`:

> `props` — Contains all of the page data along with an `errors` object (defaults to `{}` if there are no errors).

**Current state:**

- `inertiaprotocol.Render` does **not** guarantee `props["errors"]` exists.
- The higher-level `inertia` renderer manually constructs `inertiaalways.New("errors", m)` and appends it to `Context.Props`.
- Low-level callers of `inertiaprotocol.Render` must remember to inject this themselves.
- The protocol package is therefore **not self-contained** for protocol compliance.

---

## 2. Design Goals

1. `inertiaprotocol` must guarantee `props["errors"]` exists in the output `Page`.
2. The `errors` prop must always bypass partial-reload filters (per protocol: `errors` is **ALWAYS** included).
3. Custom error bags should remain a framework-level concern.
4. Minimal API surface changes.
5. No special-case logic in the concurrent resolution path.

---

## 3. Proposal A: Auto-Injection (Recommended)

### Overview

After all props are resolved in `makeProps` and `resolvePartialComponentRequest`, check whether `m["errors"]` exists. If it does not, inject `map[string]any{}`.

### Rationale

- The protocol only mandates the **default** `errors` key. Custom bags are application-level.
- The caller is responsible for passing an `errors` prop if they have validation errors.
- The protocol package only guarantees the fallback default.
- `BypassPartialFilters()` on the caller's errors prop ensures it survives partial reloads.

### Implementation

```go
func makeProps(
    ctx context.Context,
    req Request,
    componentName string,
    props []inertiaprop.Prop,
    concurrency int,
) (map[string]any, []string, error) {
    // ... existing resolution logic ...

    // Guarantee protocol compliance: errors prop always present.
    if _, ok := m["errors"]; !ok {
        m["errors"] = map[string]any{}
    }

    return m, rescuedProps, nil
}
```

Same addition in `resolvePartialComponentRequest`.

### Advantages

- **Zero API changes** to `Context`, `Request`, or `Prop`.
- Works for all callers, low-level and high-level.
- Partial-reload safe when the caller uses a prop with `BypassPartialFilters() = true`.
- Does not leak framework concepts (error bags, validation) into the protocol layer.

### Disadvantages / Risks

- If a caller passes an `errors` prop that does **not** bypass partial filters, it gets filtered out during partial reloads and the auto-injected `{}` replaces it. This is arguably correct per the protocol (errors should always be present), but it means the caller's actual errors are lost. The fix is on the caller: use `BypassPartialFilters() = true`.
- The protocol package cannot warn the caller about this misuse.

---

## 4. Proposal B: Context-Level `Errors`

### Overview

Add an `Errors` field to `Context` so the protocol package creates the prop internally:

```go
type Context struct {
    Component        string
    Version          string
    Props            []inertiaprop.Prop
    SharedProps      []inertiaprop.Prop
    PreserveFragment bool
    ClearHistory     bool
    EncryptHistory   bool
    Concurrency      int

    // NEW: If non-nil, the protocol creates an always-included "errors" prop.
    // If nil, injects errors: {}.
    Errors map[string]string
}
```

`makeProps` would create an internal always-included prop for `errors`, ensuring it bypasses partial filters and is never deferred/optional.

### Custom Bag Support (optional extension)

```go
    // NEW: Empty string = default "errors" bag.
    // Non-empty = top-level prop key (e.g. "custom_bag").
    ErrorBag string
```

When `ErrorBag` is non-empty and `Errors` is non-empty, the output becomes:

```json
{
  "props": {
    "custom_bag": {"errors": {...}}
  }
}
```

### Changes Required

- Add `Errors` (and optionally `ErrorBag`) to `Context`.
- Update `makeProps` to create an internal always-included prop for errors.
- Update the `inertia` renderer to populate `Context.Errors` instead of appending `inertiaalways.New("errors", ...)`.

### Advantages

- Protocol package fully owns the `errors` contract.
- Caller cannot forget to include errors.
- Custom bags are supported natively (if `ErrorBag` is added).
- The `errors` prop is guaranteed to bypass partial filters.

### Disadvantages

- `ErrorBag` is a **framework concept** leaking into the protocol layer.
- The `Renderer` would need to populate `Context.Errors` instead of passing a `Prop`, creating a split in how props vs errors are handled.
- More API surface.
- Breaks the current clean separation where the protocol package only knows about generic `Prop`s.

---

## 5. Recommendation

**Adopt Proposal A (Auto-Injection)** because:

1. It satisfies the protocol requirement with **zero API changes**.
2. It keeps the protocol package focused on protocol compliance, not framework features.
3. The existing `inertia` renderer already handles custom bags and validation correctly by passing a `BypassPartialFilters()` prop. Auto-injection is merely a safety net.
4. It does not bifurcate the `Context` API between "props" and "errors".
5. It aligns with the protocol spec's wording: the `errors` object is a **default**, not a framework feature.

The `inertia` renderer can continue to pass the errors prop as it does today. The auto-injection only fills the gap for direct callers of `inertiaprotocol.Render`.

---

## 6. Out of Scope

- **Custom error bags**: These are framework/application concepts. The protocol package only guarantees the default `errors` key.
- **`X-Inertia-Error-Bag` header parsing**: This belongs in `protocol.go` / `parseRequest` at the framework level, not in `inertiaprotocol`.
- **Validation error structs**: These are defined in the top-level `inertia` package and should stay there.

---

*End of proposal.*
