# Analysis: Merge Props Implementation

> **Scope:** `internal/inertiaprotocol/render.go`, `internal/inertiaprop/prop.go`, `inertiascroll/scroll_prop.go`
> **Ignores:** `deepMergeProps` (intentionally omitted by design)

---

## 1. How Merge Works Today

### 1.1 Builder API (`internal/inertiaprop/prop.go`)

```go
type Mergeable struct {
    AppendKeys  []MergeKey   // nested paths to append
    PrependKeys []MergeKey   // nested paths to prepend
    Append      bool         // root-level append
    Prepend     bool         // root-level prepend
}

type MergeKey struct {
    Key     string  // nested path (e.g. "messages", "data")
    MatchOn string  // match field (e.g. "id", "uuid")
}
```

- `NewMergeOpts()` → default `{append: true}` (root-level append)
- `.Append(keys...)` → sets `AppendKeys`, disables root-level append flag
- `.Prepend(keys...)` → sets `PrependKeys`, disables root-level append flag
- Calling `.Append()` with no keys → sets root-level `Append = true`
- Calling `.Prepend()` with no keys → sets root-level `Prepend = true`

### 1.2 Render Pipeline (`render.go`)

`makeMergeProps(props, blacklist, scrollMergeIntent)`:

1. Iterate all props
2. Skip if in `blacklist` (from `X-Inertia-Reset`) or not `Mergeable()`
3. **Scroll prop shortcut:** if prop is `Scrollable()`, its `Path` goes directly to `append` or `prepend` depending on `scrollMergeIntent`
4. **Root-level merge:** `merge.Append` → `mergeProps.append`, `merge.Prepend` → `mergeProps.prepend`
5. **Nested merge keys:** for each `MergeKey` in `AppendKeys`/`PrependKeys`:
   - `path = propKey` (or `QualifyPath(propKey, key.Key)` if key is non-empty)
   - append `path` to the appropriate list
   - if `MatchOn` is set, append `QualifyPath(path, MatchOn)` to `matchOn`

`Render` then sets:
```go
Page{
    MergeProps:   mergeProps.append,
    PrependProps: mergeProps.prepend,
    MatchPropsOn: mergeProps.matchOn,
}
```

### 1.3 Scroll Prop Integration (`inertiascroll/scroll_prop.go`)

```go
prop.scroll = &Scrollable{
    Path: inertiaprotocol.QualifyPath(key, "data"),
    ...
}
prop.merge = &Mergeable{Append: true, Prepend: false, ...}
```

Every scroll prop is inherently mergeable (append by default). Its `Path` (e.g. `"posts.data"`) gets fed into `makeMergeProps`.

---

## 2. Protocol Compliance Check

### 2.1 Page Object Fields

| Protocol Field | Supported? | Notes |
|---|---|---|
| `mergeProps` | Yes | Array of root-level and nested append paths |
| `prependProps` | Yes | Array of root-level and nested prepend paths |
| `matchPropsOn` | Yes | Array of fully-qualified match paths |
| `deepMergeProps` | **N/A** | Intentionally omitted by design |

### 2.2 Merge Examples from Protocol

**Example 1 — Basic merge + prepend + matchOn:**
```json
{
  "mergeProps": ["posts"],
  "prependProps": ["notifications"],
  "matchPropsOn": ["posts.id", "notifications.id", "conversations.data.id"]
}
```

**Our output for equivalent config:**
- `inertiaprop.New("posts", data, WithMerge(NewMergeOpts().Append(MatchOn: "id")))`
  - `merge.Append = false` (keys provided)
  - `AppendKeys = [{Key: "", MatchOn: "id"}]`
  - `path = "posts"` (Key is empty)
  - `matchOn = "posts.id"` ✓
- `inertiaprop.New("notifications", data, WithMerge(NewMergeOpts().Prepend(MatchOn: "uuid")))`
  - `PrependKeys = [{Key: "", MatchOn: "uuid"}]`
  - `prepend = "notifications"` ✓
  - `matchOn = "notifications.uuid"` ✓
- `inertiaprop.New("conversation", data, WithMerge(NewMergeOpts().Append(Key: "messages", MatchOn: "id")))`
  - `path = QualifyPath("conversation", "messages") = "conversation.messages"`
  - `append = "conversation.messages"` ✓
  - `matchOn = "conversation.messages.id"` ✓

**Result:** ✅ Protocol-compliant.

**Example 2 — Scroll props:**
```json
{
  "mergeProps": ["posts.data"],
  "scrollProps": {
    "posts": {
      "pageName": "page",
      "previousPage": null,
      "nextPage": 2,
      "currentPage": 1
    }
  }
}
```

**Our output:**
- Scroll prop key = `"posts"`, path = `QualifyPath("posts", "data") = "posts.data"`
- `makeMergeProps` sees `Scrollable()`, appends `"posts.data"` to `mergeProps` ✓
- `makeScrollProps` creates `scrollProps["posts"] = {...}` ✓

**Result:** ✅ Protocol-compliant.

---

## 3. Potential Issues

### 3.1 Scroll Props Override Root-Level Merge Flags

**Current code:**
```go
if scroll, ok := prop.Scrollable(); ok {
    if scrollMergeIntent == ...Prepend {
        m.prepend = append(m.prepend, scroll.Path)
    } else {
        m.append = append(m.append, scroll.Path)
    }
    continue  // skips regular merge logic entirely
}
```

**Implication:** A scroll prop's `Mergeable()` is never processed. The scroll `Path` is used instead. This is **correct** because scroll props are a special case of merge props — they always merge at their `Path`. But it means a scroll prop cannot simultaneously have custom `AppendKeys`/`PrependKeys`.

**Verdict:** ✅ Acceptable. Scroll props and custom merge keys are mutually exclusive concepts in practice. If you need custom merge keys on a scroll prop, you wouldn't use the scroll prop package — you'd use a standard prop with `MergeOpts`.

### 3.2 Both Append and Prepend Flags Can Be True

**Current code:**
```go
case merge.Prepend:
    m.prepend = append(m.prepend, prop.Key())
case merge.Append:
    m.append = append(m.append, prop.Key())
```

If a `Mergeable` somehow has both `Append = true` and `Prepend = true`, both lists get the key. The `MergeOpts` builder prevents this (`.Append()` sets `prepend = false`, `.Prepend()` sets `append = false`), but if someone constructs `Mergeable` manually, both could be true.

**Verdict:** ⚠️ **Low risk.** The builder prevents this. The protocol doesn't define behavior for both flags being true simultaneously.

### 3.3 Empty `MatchOn` Still Creates a Merge Path

If `MergeKey{Key: "data"}` is used without `MatchOn`, the path is added to `append`/`prepend` but nothing goes to `matchOn`. This is correct.

**Verdict:** ✅ Correct.

### 3.4 `QualifyPath` Behavior with Empty Path

```go
func QualifyPath(propKey, path string) string {
    if path == "" || strings.HasPrefix(path, propKey+".") || path == propKey {
        return path
    }
    return propKey + "." + path
}
```

If `path == ""`, returns `""` (not `propKey`). This means a `MergeKey{Key: ""}` would produce an empty path in `addMergeKeys`.

**Wait — let me re-read:**

```go
func (m *mergeProps) addMergeKeys(propKey string, keys []inertiaprop.MergeKey, props *[]string) {
    for _, key := range keys {
        path := propKey
        if key.Key != "" {
            path = QualifyPath(propKey, key.Key)
        }
        *props = append(*props, path)
        ...
    }
}
```

So when `key.Key == ""`:
- `path = propKey` (not `QualifyPath`)
- This means `MergeKey{Key: "", MatchOn: "id"}` gives `path = "posts"`, `matchOn = "posts.id"`

When `key.Key == "data"`:
- `path = QualifyPath("posts", "data") = "posts.data"`

This is correct. The `QualifyPath` empty-string check is a safety net that doesn't get exercised in the merge path because `addMergeKeys` only calls it when `key.Key != ""`.

**Verdict:** ✅ Correct, but the empty-string branch in `QualifyPath` is technically unreachable from the merge pipeline.

### 3.5 Reset/Blacklist Handling

`makeMergeProps` receives `blacklist` (from `req.ResetProps`). A blacklisted merge prop is skipped entirely.

**Protocol:** `X-Inertia-Reset` resets props before merging new data.

**Our behavior:** The prop is omitted from `mergeProps`/`prependProps`/`matchPropsOn`, so the client won't merge it.

**Verdict:** ✅ Correct.

### 3.6 `matchOn` with Root-Level Match (Empty Key)

Test case: `MergeKey{MatchOn: "id"}` with prop key `"posts"`.

- `path = "posts"` (Key is empty)
- `matchOn = "posts.id"`

**Protocol spec for matching:**
> `matchPropsOn` — Array of prop keys to use for matching when merging props.

So `"posts.id"` means "match items in the `posts` prop by their `id` field". This is correct.

**Verdict:** ✅ Correct.

---

## 4. Test Coverage

From `renderer_test.go`:

```go
assert.Contains(t, page["mergeProps"], "posts")
assert.Contains(t, page["mergeProps"], "conversation.messages")
assert.Contains(t, page["prependProps"], "notifications")
assert.ElementsMatch(t, []any{
    "posts.id",
    "notifications.uuid",
    "conversation.messages.id",
}, page["matchPropsOn"])
```

This tests:
- Root-level append with matchOn (`posts` + `posts.id`)
- Root-level prepend with matchOn (`notifications` + `notifications.uuid`)
- Nested append with matchOn (`conversation.messages` + `conversation.messages.id`)

**Missing coverage:**
- Scroll prop merge metadata (tested separately in "with infinite scroll metadata" test)
- Blacklist/reset interaction with merge props
- Both append and prepend flags on same prop (edge case)

---

## 5. Conclusion

**Status: ✅ Working as expected.**

The merge props implementation correctly produces:
1. `mergeProps` for append targets (root + nested)
2. `prependProps` for prepend targets (root + nested)
3. `matchPropsOn` for matching fields on merge targets
4. Scroll prop paths correctly fed into the merge lists

**No bugs found.** The only minor quirk is that `QualifyPath`'s empty-string branch is unreachable from the merge pipeline, but this is harmless.

**No changes required.**
