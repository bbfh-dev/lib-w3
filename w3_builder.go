package libw3

import "os"

// Refer to [NewBuilder].
type Builder struct {
	maxConcurrentRequests uint16
	middleware            MiddlewareChain
	errs                  []error
}

// Used to create a configured [Engine].
func NewBuilder() *Builder {
	return &Builder{
		maxConcurrentRequests: 200,
		middleware:            MiddlewareChain{},
		errs:                  []error{},
	}
}

// LimitConcurrentRequests sets the pool of concurrent requests.
// Any additional requests will have to wait for another request to finish before being served.
func (builder *Builder) LimitConcurrentRequests(limit uint16) *Builder {
	builder.maxConcurrentRequests = limit
	return builder
}

// Use [Middleware] that will be applied to ALL endpoints.
// Can be called multiple times.
func (builder *Builder) Use(middleware Middleware) *Builder {
	builder.middleware = builder.middleware.ProperAppend(middleware)
	return builder
}

func (builder *Builder) Build() (*Engine, error) {
	return nil, nil
}

// MustBuild calls [Builder.Build] and calls [os.Exit] on error.
func (builder *Builder) MustBuild() *Engine {
	engine, err := builder.Build()
	if err != nil {
		Logger.Fatal("%s", err.Error())
		os.Exit(1)
	}
	return engine
}
