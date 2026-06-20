// Package inertiahttp implements the HTTP-level Inertia.js protocol:
// request parsing, response rendering, and the page renderer.
//
// It bridges the transport-neutral inertiaprotocol renderer with
// net/http and html/template, handling both client-side and server-side
// rendering modes.
package inertiahttp

import "go.inout.gg/foundations/debug"

//nolint:gochecknoglobals
var d = debug.Debuglog("inertiahttp")
