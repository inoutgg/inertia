package inertiaheader

const (
	HeaderXInertia                 = "X-Inertia"                              // client/server
	HeaderXInertiaVersion          = "X-Inertia-Version"                      // client
	HeaderXInertiaLocation         = "X-Inertia-Location"                     // client/server, redirect URL
	HeaderXInertiaRedirect         = "X-Inertia-Redirect"                     // server, preserve fragment
	HeaderXInertiaPartialData      = "X-Inertia-Partial-Data"                 // client, whitelist
	HeaderXInertiaPartialExcept    = "X-Inertia-Partial-Except"               // client, blacklist
	HeaderXInertiaPartialComponent = "X-Inertia-Partial-Component"            // client
	HeaderXInertiaReset            = "X-Inertia-Reset"                        // client, force reload
	HeaderXInertiaErrorBag         = "X-Inertia-Error-Bag"                    // client
	HeaderXInertiaScrollMerge      = "X-Inertia-Infinite-Scroll-Merge-Intent" // client, append/prepend
	HeaderXInertiaExceptOnceProps  = "X-Inertia-Except-Once-Props"            // client, once props already loaded

	HeaderVary        = "Vary"
	HeaderContentType = "Content-Type"
	HeaderReferer     = "Referer"
)

const (
	ContentTypeHTML = "text/html"
	ContentTypeJSON = "application/json"
)

const HeaderValueTrue = "true"
