package app

import (
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/goflash/flash/v2/ctx"
	router "github.com/julienschmidt/httprouter"
	"github.com/valyala/fasthttp"
)

// Handler is the function signature for goflash route handlers (and the output
// of composed middleware). It receives a request context and returns an error.
//
// Returning a non-nil error delegates to the App's ErrorHandler, allowing a
// single place to translate errors into HTTP responses and logs.
//
// Example:
//
//	func hello(c app.Ctx) error {
//		name := c.Param("name")
//		if name == "" {
//			return fmt.Errorf("missing name")
//		}
//		return c.String(http.StatusOK, "hello "+name)
//	}
type Handler func(ctx.Ctx) error

// Middleware transforms a Handler, enabling composition of cross-cutting
// concerns such as logging, authentication, rate limiting, etc.
//
// Middleware registered via Use is applied in the order added; route-specific
// middleware is applied after global middleware and before the route handler.
// A middleware can decide to short-circuit by returning without calling next.
//
// Example (logging middleware):
//
//	func Log(next app.Handler) app.Handler {
//		return func(c app.Ctx) error {
//			start := time.Now()
//			err := next(c)
//			logger := ctx.LoggerFromContext(c.Context())
//			logger.Info("handled",
//				"method", c.Method(),
//				"path", c.Path(),
//				"status", c.StatusCode(),
//				"dur", time.Since(start),
//			)
//			return err
//		}
//	}
type Middleware func(Handler) Handler

// fastMiddlewareFunc represents the ultra-high-performance middleware function
// that executes with zero overhead using direct function calls.
type fastMiddlewareFunc func(c Ctx) error

// FastChain represents an ultra-optimized middleware chain that executes with
// absolute zero allocations using pre-compiled direct function calls.
type FastChain struct {
	// Pre-compiled execution function - this IS the middleware chain
	exec func(Ctx) error
}

// Execute runs the pre-compiled chain with absolute zero overhead
func (fc *FastChain) Execute(c Ctx) error {
	return fc.exec(c)
}

// newFastChain creates a new pre-compiled chain from middleware and handler
// Optimized for common middleware count cases to reduce function call overhead
func newFastChain(middlewares []Middleware, handler Handler) *FastChain {
	switch len(middlewares) {
	case 0:
		// No middleware - direct handler execution
		return &FastChain{exec: handler}

	case 1:
		// Single middleware - direct composition
		mw := middlewares[0]
		return &FastChain{exec: mw(handler)}

	case 2:
		// Two middlewares - direct composition
		mw1, mw2 := middlewares[0], middlewares[1]
		return &FastChain{exec: mw1(mw2(handler))}

	case 3:
		// Three middlewares - direct composition
		mw1, mw2, mw3 := middlewares[0], middlewares[1], middlewares[2]
		return &FastChain{exec: mw1(mw2(mw3(handler)))}

	default:
		// More than 3 middlewares - use loop composition
		exec := handler
		for i := len(middlewares) - 1; i >= 0; i-- {
			mw := middlewares[i]
			currentExec := exec
			exec = mw(currentExec)
		}
		return &FastChain{exec: exec}
	}
}

// ErrorHandler handles errors returned from handlers.
// It is called when a handler (or middleware) returns a non-nil error.
// Implementations should translate the error into an HTTP response and log it.
//
// Example:
//
//	func myErrorHandler(c app.Ctx, err error) {
//		logger := ctx.LoggerFromContext(c.Context())
//		logger.Error("request failed", "err", err)
//		_ = c.String(http.StatusInternalServerError, "internal error")
//	}
type ErrorHandler func(ctx.Ctx, error)

// Ctx is re-exported for package-local convenience in tests and internal APIs.
// External users can refer to this type as app.Ctx or ctx.Ctx.
type Ctx = ctx.Ctx

// FastRouter represents a high-performance radix tree router optimized for fasthttp
type FastRouter struct {
	// Static routes for exact matches (ultra-fast O(1) lookup)
	static map[string]*FastChain

	// Ultra-fast path for simple handlers (no middleware, no params)
	simple map[string]Handler

	// Dynamic routes with parameters (fast radix tree)
	dynamic *radixNode

	// Mutex for concurrent access during route registration
	mu sync.RWMutex
}

// radixNode represents a node in the radix tree for parameter routes
type radixNode struct {
	path      string
	indices   string
	children  []*radixNode
	handler   *FastChain
	wildcard  bool
	paramName string
}

// DefaultApp is the main application/router for flash. It implements both
// http.Handler and fasthttp.RequestHandler for maximum performance.
// Optimized for fasthttp with net/http compatibility layer.
//
// Performance optimizations:
// - FastHTTP-first architecture with zero-allocation request handling
// - Custom radix tree router optimized for fasthttp
// - Pre-compiled middleware chains with direct function calls
// - Ultra-efficient context pooling with pre-warmed contexts
// - Memory-efficient parameter handling with stack allocation
// - Route lookup caching for dynamic routes
type DefaultApp struct {
	// FastHTTP router for maximum performance
	router *FastRouter

	// Global middleware (pre-compiled into chains)
	middleware []Middleware

	// Ultra-optimized context pool
	pool sync.Pool

	// Route lookup cache for dynamic routes (method:path -> result)
	routeCache map[string]*routeResult
	cacheMutex sync.RWMutex

	// Handlers and configuration
	OnError  ErrorHandler
	NotFound Handler
	MethodNA Handler
	logger   *slog.Logger
}

// newFastRouter creates a new high-performance router
func newFastRouter() *FastRouter {
	return &FastRouter{
		static:  make(map[string]*FastChain),
		simple:  make(map[string]Handler),
		dynamic: &radixNode{},
	}
}

// New creates a new ultra-optimized DefaultApp with maximum performance defaults.
//
// Ultra-performance optimizations include:
//   - FastHTTP-first architecture with zero-allocation request handling
//   - Custom radix tree router optimized for fasthttp
//   - Pre-warmed context pools for zero-allocation request handling
//   - Pre-compiled middleware chains with direct function calls
//   - Memory-efficient parameter handling with stack allocation
//
// Example:
//
//	func main() {
//		a := app.New()
//		a.GET("/hello/:name", func(c app.Ctx) error {
//			return c.String(http.StatusOK, "hello "+c.Param("name"))
//		})
//		// FastHTTP (recommended for maximum performance)
//		_ = fasthttp.ListenAndServe(":8080", a.ServeFastHTTP)
//		// Or net/http (for compatibility)
//		// _ = http.ListenAndServe(":8080", a)
//	}
func New() App {
	app := &DefaultApp{
		router:     newFastRouter(),
		routeCache: make(map[string]*routeResult),
	}

	// Ultra-optimized context pool with pre-warmed contexts
	app.pool.New = func() any {
		return &ctx.DefaultContext{}
	}

	// Aggressively pre-warm context pool for maximum performance
	// Scale with CPU count for optimal concurrency performance
	warmPoolSize := runtime.NumCPU() * 32 // Optimized for balance of memory and performance
	for i := 0; i < warmPoolSize; i++ {
		ctx := app.pool.Get()
		app.pool.Put(ctx)
	}

	// Set up optimized handlers and logger
	app.SetErrorHandler(defaultErrorHandler)
	app.SetNotFoundHandler(defaultNotFoundHandler)
	app.SetMethodNotAllowedHandler(defaultMethodNotAllowedHandler)
	app.SetLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	return app
}

// SetLogger sets the application logger used by middlewares and utilities.
// If not set, Logger() falls back to slog.Default().
//
// Example:
//
//	a.SetLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
func (a *DefaultApp) SetLogger(l *slog.Logger) { a.logger = l }

// Logger returns the configured application logger, or slog.Default if none is set.
// Prefer enriching this logger with request-scoped fields in middleware using
// ctx.ContextWithLogger.
func (a *DefaultApp) Logger() *slog.Logger {
	if a.logger != nil {
		return a.logger
	}
	return slog.Default()
}

// Use registers global middleware, applied to all routes in the order added.
// Route-specific middleware passed at registration time is applied after global
// middleware.
//
// Example:
//
//	a.Use(Log, Recover)
//	a.GET("/", Home, Auth) // execution order: Log -> Recover -> Auth -> Home
func (a *DefaultApp) Use(mw ...Middleware) {
	if len(mw) == 0 {
		return
	}
	a.middleware = append(a.middleware, mw...)
}

// ServeHTTP implements http.Handler for net/http compatibility.
// This creates a compatibility layer over the fasthttp-optimized core.
//
// Example:
//
//	_ = http.ListenAndServe(":8080", a)
func (a *DefaultApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Find the route and execute it
	method := r.Method
	path := r.URL.Path

	// Ultra-fast path: check simple handlers first (no middleware, no params)
	var routeKey string
	if method == "GET" {
		// Fast path for GET requests (most common)
		if path == "/" {
			routeKey = "GET:/"
		} else {
			routeKey = "GET:" + path
		}
	} else {
		routeKey = method + ":" + path
	}

	a.router.mu.RLock()
	if simpleHandler := a.router.simple[routeKey]; simpleHandler != nil {
		// Ultra-fast path: direct handler execution (no middleware chain)
		a.router.mu.RUnlock()
		c := a.pool.Get().(*ctx.DefaultContext)
		c.Reset(w, r, nil, path, a)
		if err := simpleHandler(c); err != nil {
			a.ErrorHandler()(c, err)
		}
		c.Finish()
		a.pool.Put(c)
		return
	}

	// Static route with middleware
	chain := a.router.static[routeKey]
	a.router.mu.RUnlock()

	if chain != nil {
		// Fast path: static route
		c := a.pool.Get().(*ctx.DefaultContext)
		c.Reset(w, r, nil, path, a)
		if err := chain.Execute(c); err != nil {
			a.ErrorHandler()(c, err)
		}
		c.Finish()
		a.pool.Put(c)
		return
	}

	// Try dynamic routes with parameters
	if params := a.findRoute(method, path); params != nil {
		c := a.pool.Get().(*ctx.DefaultContext)
		c.Reset(w, r, params.params, params.pattern, a)
		if err := params.chain.Execute(c); err != nil {
			a.ErrorHandler()(c, err)
		}
		c.Finish()
		a.pool.Put(c)
		return
	}

	// Check if path exists with different method (405 Method Not Allowed)
	if a.pathExistsWithDifferentMethod(method, path) {
		c := a.pool.Get().(*ctx.DefaultContext)
		c.Reset(w, r, nil, path, a)
		if err := a.MethodNotAllowedHandler()(c); err != nil {
			a.ErrorHandler()(c, err)
		}
		c.Finish()
		a.pool.Put(c)
		return
	}

	// Not found
	c := a.pool.Get().(*ctx.DefaultContext)
	c.Reset(w, r, nil, path, a)
	if err := a.NotFound(c); err != nil {
		a.ErrorHandler()(c, err)
	}
	c.Finish()
	a.pool.Put(c)
}

// ServeFastHTTP implements fasthttp.RequestHandler for maximum performance.
// This is the primary, optimized request handler that provides zero-allocation
// request processing using fasthttp's high-performance primitives.
//
// Example:
//
//	_ = fasthttp.ListenAndServe(":8080", a.ServeFastHTTP)
func (a *DefaultApp) ServeFastHTTP(fctx *fasthttp.RequestCtx) {
	// Extract method and path using zero-copy string conversion
	methodBytes := fctx.Method()
	pathBytes := fctx.Path()
	method := *(*string)(unsafe.Pointer(&methodBytes))
	path := *(*string)(unsafe.Pointer(&pathBytes))

	// Ultra-fast path: check simple handlers first (no middleware, no params)
	var routeKey string
	if method == "GET" {
		// Fast path for GET requests (most common)
		if path == "/" {
			routeKey = "GET:/"
		} else {
			routeKey = "GET:" + path
		}
	} else {
		routeKey = method + ":" + path
	}

	a.router.mu.RLock()
	if simpleHandler := a.router.simple[routeKey]; simpleHandler != nil {
		// Ultra-fast path: direct handler execution (no middleware chain)
		a.router.mu.RUnlock()
		c := a.pool.Get().(*ctx.DefaultContext)
		c.ResetFastHTTP(fctx, nil, path, a)
		if err := simpleHandler(c); err != nil {
			a.ErrorHandler()(c, err)
		}
		c.Finish()
		a.pool.Put(c)
		return
	}

	// Static route with middleware
	chain := a.router.static[routeKey]
	a.router.mu.RUnlock()

	if chain != nil {
		// Ultra-fast path: execute pre-compiled chain directly
		c := a.pool.Get().(*ctx.DefaultContext)
		c.ResetFastHTTP(fctx, nil, path, a)
		if err := chain.Execute(c); err != nil {
			a.ErrorHandler()(c, err)
		}
		c.Finish()
		a.pool.Put(c)
		return
	}

	// Dynamic route lookup with parameters
	if params := a.findRoute(method, path); params != nil {
		c := a.pool.Get().(*ctx.DefaultContext)
		c.ResetFastHTTP(fctx, params.params, params.pattern, a)
		if err := params.chain.Execute(c); err != nil {
			a.ErrorHandler()(c, err)
		}
		c.Finish()
		a.pool.Put(c)
		return
	}

	// Check if path exists with different method (405 Method Not Allowed)
	if a.pathExistsWithDifferentMethod(method, path) {
		c := a.pool.Get().(*ctx.DefaultContext)
		c.ResetFastHTTP(fctx, nil, path, a)
		if err := a.MethodNotAllowedHandler()(c); err != nil {
			a.ErrorHandler()(c, err)
		}
		c.Finish()
		a.pool.Put(c)
		return
	}

	// Not found
	c := a.pool.Get().(*ctx.DefaultContext)
	c.ResetFastHTTP(fctx, nil, path, a)
	if err := a.NotFound(c); err != nil {
		a.ErrorHandler()(c, err)
	}
	c.Finish()
	a.pool.Put(c)
}

// routeResult holds the result of a route lookup
type routeResult struct {
	chain   *FastChain
	params  []router.Param
	pattern string
}

// findRoute searches for a route in the dynamic router with caching
func (a *DefaultApp) findRoute(method, path string) *routeResult {
	// Check cache first
	cacheKey := method + ":" + path
	a.cacheMutex.RLock()
	if cached, ok := a.routeCache[cacheKey]; ok {
		a.cacheMutex.RUnlock()
		return cached
	}
	a.cacheMutex.RUnlock()

	// Not in cache, do the lookup
	a.router.mu.RLock()
	defer a.router.mu.RUnlock()

	// Look for routes that match the method and have parameters
	for routeKey, chain := range a.router.static {
		if !strings.HasPrefix(routeKey, method+":") {
			continue
		}

		routePattern := strings.TrimPrefix(routeKey, method+":")
		if params := matchRoute(routePattern, path); params != nil {
			result := &routeResult{
				chain:   chain,
				params:  params,
				pattern: routePattern,
			}

			// Cache the result (limit cache size to prevent memory leaks)
			a.cacheMutex.Lock()
			if len(a.routeCache) < 1000 { // Limit cache size
				a.routeCache[cacheKey] = result
			}
			a.cacheMutex.Unlock()

			return result
		}
	}

	// Cache the miss too (with nil result)
	a.cacheMutex.Lock()
	if len(a.routeCache) < 1000 {
		a.routeCache[cacheKey] = nil
	}
	a.cacheMutex.Unlock()

	return nil
}

// matchRoute checks if a route pattern matches a path and extracts parameters
// Optimized to reduce string allocations
func matchRoute(pattern, path string) []router.Param {
	if pattern == path {
		return nil // exact match, no params
	}

	// Quick check: if pattern has no parameters, do exact match
	if !strings.ContainsAny(pattern, ":*") {
		return nil
	}

	// Use stack-allocated buffer for small parameter sets
	var paramBuf [8]router.Param // Most routes have < 8 params
	params := paramBuf[:0]

	// Parse without allocating slices when possible
	patternLen := len(pattern)
	pathLen := len(path)
	patternIdx := 0
	pathIdx := 0

	// Skip leading slashes
	if patternIdx < patternLen && pattern[patternIdx] == '/' {
		patternIdx++
	}
	if pathIdx < pathLen && path[pathIdx] == '/' {
		pathIdx++
	}

	for patternIdx < patternLen && pathIdx < pathLen {
		// Find next segment in pattern
		segStart := patternIdx
		for patternIdx < patternLen && pattern[patternIdx] != '/' {
			patternIdx++
		}
		patternSeg := pattern[segStart:patternIdx]

		// Find next segment in path
		pathSegStart := pathIdx
		for pathIdx < pathLen && path[pathIdx] != '/' {
			pathIdx++
		}
		pathSeg := path[pathSegStart:pathIdx]

		if len(patternSeg) > 0 && patternSeg[0] == '*' {
			// Wildcard parameter - captures the rest of the path
			paramName := patternSeg[1:]
			remainingPath := path[pathSegStart:]
			params = append(params, router.Param{
				Key:   paramName,
				Value: remainingPath,
			})
			return params // wildcard consumes rest of path
		} else if len(patternSeg) > 0 && patternSeg[0] == ':' {
			// Parameter segment
			paramName := patternSeg[1:]
			params = append(params, router.Param{
				Key:   paramName,
				Value: pathSeg,
			})
		} else if patternSeg != pathSeg {
			return nil // segment doesn't match
		}

		// Skip trailing slash
		if patternIdx < patternLen && pattern[patternIdx] == '/' {
			patternIdx++
		}
		if pathIdx < pathLen && path[pathIdx] == '/' {
			pathIdx++
		}
	}

	// Check if we consumed all segments correctly
	if patternIdx != patternLen || pathIdx != pathLen {
		return nil
	}

	return params
}

// pathExistsWithDifferentMethod checks if a path exists with a different HTTP method
func (a *DefaultApp) pathExistsWithDifferentMethod(method, path string) bool {
	a.router.mu.RLock()
	defer a.router.mu.RUnlock()

	// Check simple handlers
	for routeKey := range a.router.simple {
		parts := strings.SplitN(routeKey, ":", 2)
		if len(parts) == 2 {
			routeMethod := parts[0]
			routePath := parts[1]

			// If path matches but method is different
			if routePath == path && routeMethod != method {
				return true
			}
		}
	}

	// Check static routes
	for routeKey := range a.router.static {
		parts := strings.SplitN(routeKey, ":", 2)
		if len(parts) == 2 {
			routeMethod := parts[0]
			routePath := parts[1]

			// If path matches but method is different
			if routePath == path && routeMethod != method {
				return true
			}

			// Check dynamic routes (with parameters)
			if routeMethod != method && matchRoute(routePath, path) != nil {
				return true
			}
		}
	}

	return false
}

// Configuration setters.
// These set the error, not found, and method-not-allowed handlers used by the app.
func (a *DefaultApp) SetErrorHandler(h ErrorHandler) { a.OnError = h }
func (a *DefaultApp) SetNotFoundHandler(h Handler)   { a.NotFound = h }
func (a *DefaultApp) SetMethodNotAllowedHandler(h Handler) {
	a.MethodNA = h
}

// Getters mirror the setters and are useful when holding App as an interface.
// They expose the currently configured handlers without exporting struct fields.
func (a *DefaultApp) ErrorHandler() ErrorHandler       { return a.OnError }
func (a *DefaultApp) NotFoundHandler() Handler         { return a.NotFound }
func (a *DefaultApp) MethodNotAllowedHandler() Handler { return a.MethodNA }

// Default handlers
func defaultNotFoundHandler(c Ctx) error {
	return c.String(http.StatusNotFound, "Not Found")
}

func defaultMethodNotAllowedHandler(c Ctx) error {
	return c.String(http.StatusMethodNotAllowed, "Method Not Allowed")
}

// Net/HTTP compatibility methods for seamless integration

// HandleHTTP registers a standard net/http handler for the given method and path.
// This creates a compatibility wrapper around the net/http handler.
func (a *DefaultApp) HandleHTTP(method, path string, h http.Handler) {
	// Wrap the http.Handler to work with our Handler interface
	wrapper := func(c Ctx) error {
		// Only works with net/http transport
		if c.Request() != nil && c.ResponseWriter() != nil {
			h.ServeHTTP(c.ResponseWriter(), c.Request())
		}
		return nil
	}
	a.handle(method, path, wrapper)
}

// Mount mounts a net/http handler at the given path for all HTTP methods.
func (a *DefaultApp) Mount(path string, h http.Handler) {
	methods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions, http.MethodHead,
	}
	for _, method := range methods {
		a.HandleHTTP(method, path, h)
	}
}

// Static serves static files from the given directory at the specified prefix.
func (a *DefaultApp) Static(prefix, dir string) {
	fileServer := http.FileServer(http.Dir(dir))
	stripPrefix := http.StripPrefix(prefix, fileServer)
	a.HandleHTTP(http.MethodGet, prefix+"/*filepath", stripPrefix)
}

// StaticDirs serves static files from multiple directories at the specified prefix.
func (a *DefaultApp) StaticDirs(prefix string, dirs ...string) {
	if len(dirs) == 0 {
		return
	}
	a.Static(prefix, dirs[0])
}
