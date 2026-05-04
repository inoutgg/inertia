# Inertia Protocol v3 Upgrade Plan

- [x] Step 1: Update protocol constants.
  Add `X-Inertia-Redirect`, `X-Inertia-Infinite-Scroll-Merge-Intent`, and `X-Inertia-Except-Once-Props` constants and tests.

- [x] Step 2: Update initial HTML hydration format.
  Emit the v3 `<script data-page="..." type="application/json">...</script><div id="..."></div>` shape, preserving root view attributes on the root element.

- [x] Step 3: Update page object serialization.
  Omit false history flags and add optional v3 page object fields.

- [x] Step 4: Fix asset version mismatch behavior.
  Return `409 Conflict` only for `GET` Inertia version mismatches and let non-GET requests continue normally.

- [x] Step 5: Add fragment redirect support.
  Support redirects that return `409 Conflict` with `X-Inertia-Redirect` when preserving fragments.

- [x] Step 6: Expand merge prop support.
  Support append, prepend, deep merge, and match metadata without nested prop resolution.

- [x] Step 7: Add once prop support.
  Emit `onceProps`, read `X-Inertia-Except-Once-Props`, skip already-loaded once props, and support fresh/expiration metadata.

- [x] Step 8: Add infinite scroll support.
  Emit `scrollProps`, configure merge metadata, and use `X-Inertia-Infinite-Scroll-Merge-Intent` for append/prepend behavior.

- [x] Step 9: Track shared props metadata.
  Emit `sharedProps` for keys registered as shared props while preserving response prop precedence.

- [x] Step 10: Adjust SSR behavior.
  Keep v3 initial page JSON available with SSR and fall back to client-side rendering on SSR failure.

- [x] Step 11: Run full test suite.
  Ensure all implementation and tests pass, then mark this plan complete.

- [x] Step 12: Remove v1/v2 deprecated leftovers.
  Remove deprecated compatibility wording and v2-style terminology that remains after the v3 protocol upgrade.

## Explicitly Omitted

Nested prop and dot-notation resolution for server-side prop evaluation/filtering is intentionally out of scope. Dot-path strings may be emitted as metadata, but the server will not resolve nested props at arbitrary depth.
