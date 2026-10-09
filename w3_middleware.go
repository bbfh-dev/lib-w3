package libw3

import "math"

// Handler is responsible for handling the request. It can be transit (the [Middleware]) or close the chain (the endpoint).
type Handler func(ctx *Context)

// Middleware is a handler that executes before the request reaches the endpoint.
// This interface has extensions: [Optional], [Ordered], [RequiresInit].
type Middleware interface {
	Handle(ctx *Context)
}

// Optional middleware that only gets injected for specific methods or/and paths.
type Optional interface {
	// Whether the middleware should be injected in the specific [MiddlewareChain].
	ShouldInject(method, path string) bool
}

const (
	PrioRecovery int8 = math.MaxInt8
	PrioTiming   int8 = PrioRecovery - 1
	PrioAuth     int8 = 1
	PrioValidate int8 = -1
	PrioFinal    int8 = math.MinInt8
)

// Ordered middleware that forces itself to be before/after other middleware based on priority.
type Ordered interface {
	// How the middleware should be sorted in the [MiddlewareChain].

	// 0 — Keep order unchanged.
	// Higher numbers put the middleware closer to the beginning of the chain (execute earlier than others).
	Priority() int8
}

// RequiresInit middleware that must be initialized upon being added to the [MiddlewareChain].
type RequiresInit interface {
	// Init initializes the middleware, returns whether the middleware should be kept in the [MiddlewareChain] or be discarded.
	Init() (keep bool)
}
