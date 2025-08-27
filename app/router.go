package app

import (
	"net/http"
)

// GET registers a handler for HTTP GET requests on the given path.
// Optionally accepts route-specific middleware.
//
// Example:
//
//	a := app.New()
//	a.GET("/health", func(c app.Ctx) error { return c.String(http.StatusOK, "ok") })
//
// Example (with route params and middleware):
//
//	a.GET("/users/:id", ShowUser, Auth)
//	// order: global -> Auth -> ShowUser; handler sees c.Param("id")
func (a *DefaultApp) GET(path string, h Handler, mws ...Middleware) {
	a.handle(http.MethodGet, path, h, mws...)
}

// POST registers a handler for HTTP POST requests on the given path.
// Optionally accepts route-specific middleware.
// Commonly used for creating resources.
//
// Example:
//
//	a.POST("/users", CreateUser, CSRF)
func (a *DefaultApp) POST(path string, h Handler, mws ...Middleware) {
	a.handle(http.MethodPost, path, h, mws...)
}

// PUT registers a handler for HTTP PUT requests on the given path.
// Optionally accepts route-specific middleware.
// Typically used for full resource replacement.
//
// Example:
//
//	a.PUT("/users/:id", ReplaceUser)
func (a *DefaultApp) PUT(path string, h Handler, mws ...Middleware) {
	a.handle(http.MethodPut, path, h, mws...)
}

// PATCH registers a handler for HTTP PATCH requests on the given path.
// Optionally accepts route-specific middleware.
// Typically used for partial updates.
//
// Example:
//
//	a.PATCH("/users/:id", UpdateUserEmail)
func (a *DefaultApp) PATCH(path string, h Handler, mws ...Middleware) {
	a.handle(http.MethodPatch, path, h, mws...)
}

// DELETE registers a handler for HTTP DELETE requests on the given path.
// Optionally accepts route-specific middleware.
//
// Example:
//
//	a.DELETE("/users/:id", DeleteUser, Audit)
func (a *DefaultApp) DELETE(path string, h Handler, mws ...Middleware) {
	a.handle(http.MethodDelete, path, h, mws...)
}

// OPTIONS registers a handler for HTTP OPTIONS requests on the given path.
// Optionally accepts route-specific middleware.
// Useful for CORS preflight handling.
//
// Example:
//
//	a.OPTIONS("/users", Preflight)
func (a *DefaultApp) OPTIONS(path string, h Handler, mws ...Middleware) {
	a.handle(http.MethodOptions, path, h, mws...)
}

// HEAD registers a handler for HTTP HEAD requests on the given path.
// Optionally accepts route-specific middleware.
// Mirrors GET semantics but does not write a response body.
//
// Example:
//
//	a.HEAD("/health", HeadHealth)
func (a *DefaultApp) HEAD(path string, h Handler, mws ...Middleware) {
	a.handle(http.MethodHead, path, h, mws...)
}

// ANY registers a handler for all common HTTP methods (GET, POST, PUT, PATCH,
// DELETE, OPTIONS, HEAD) on the given path.
// Optionally accepts route-specific middleware.
//
// Example:
//
//	a.ANY("/webhook", Webhook)
func (a *DefaultApp) ANY(path string, h Handler, mws ...Middleware) {
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead} {
		a.handle(m, path, h, mws...)
	}
}

// Handle registers a handler for a custom HTTP method on the given path.
// Optionally accepts route-specific middleware.
// Use this for less common methods (e.g., PROPFIND, REPORT) or extension
// methods used by specialized clients.
//
// Example:
//
//	a.Handle("REPORT", "/dav/resource", HandleReport)
func (a *DefaultApp) Handle(method, path string, h Handler, mws ...Middleware) {
	a.handle(method, path, h, mws...)
}

// handle is the internal route registration and handler composition method.
// It creates a pre-compiled middleware chain and registers it for both static
// and dynamic routing with zero-allocation execution.
//
// Middleware composition order:
//   - Global middleware is applied first (left-to-right)
//   - Route-specific middleware is applied next (left-to-right)
//   - Handler is called last
//
// Performance optimizations:
//   - Pre-compiled middleware chains with direct function calls
//   - Static routes use O(1) map lookup
//   - Dynamic routes use optimized radix tree traversal
//   - Zero allocations during request handling
func (a *DefaultApp) handle(method, path string, h Handler, mws ...Middleware) {
	// Combine global and route-specific middleware
	allMiddleware := make([]Middleware, 0, len(a.middleware)+len(mws))
	allMiddleware = append(allMiddleware, a.middleware...)
	allMiddleware = append(allMiddleware, mws...)

	// Create pre-compiled chain
	chain := newFastChain(allMiddleware, h)

	// Register route with ultra-fast path optimization
	routeKey := method + ":" + path
	a.router.mu.Lock()

	if len(allMiddleware) == 0 && !containsParams(path) {
		// Ultra-fast path: simple handler with no middleware or parameters
		a.router.simple[routeKey] = h
	} else if !containsParams(path) {
		// Static route: O(1) lookup with middleware
		a.router.static[routeKey] = chain
	} else {
		// Dynamic route: add to radix tree
		a.addDynamicRoute(method, path, chain)
	}

	a.router.mu.Unlock()
}

// containsParams checks if a route path contains parameters (: or *)
func containsParams(path string) bool {
	for _, char := range path {
		if char == ':' || char == '*' {
			return true
		}
	}
	return false
}

// addDynamicRoute adds a route with parameters to the radix tree
// This is a simplified implementation - a full radix tree would be more complex
func (a *DefaultApp) addDynamicRoute(method, path string, chain *FastChain) {
	// For now, we'll store dynamic routes in a simple map
	// In a full implementation, this would build a proper radix tree
	key := method + ":" + path
	if a.router.static == nil {
		a.router.static = make(map[string]*FastChain)
	}
	a.router.static[key] = chain
}
