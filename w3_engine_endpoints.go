package libw3

import "net/http"

type endpointBuilder struct {
	engine     *Engine
	middleware MiddlewareChain
}

func newEndpointBuilder(engine *Engine) endpointBuilder {
	return endpointBuilder{
		engine:     engine,
		middleware: MiddlewareChain{},
	}
}

// Adds middleware that will be applied to THIS endpoint.
// Can be called multiple times.
func (builder endpointBuilder) Using(middleware Middleware) endpointBuilder {
	builder.middleware = builder.middleware.ProperAppend(middleware)
	return builder
}

// Sugar code for [endpointBuilder.Using] wrapping handler using [FromHandler]
func (builder endpointBuilder) UsingHandler(handler Handler) endpointBuilder {
	return builder.Using(FromHandler(handler))
}

func (builder endpointBuilder) GET(path string, handler Handler) {
	builder.engine.newEndpoint(http.MethodGet, path, handler, builder.middleware)
}

func (builder endpointBuilder) HEAD(path string, handler Handler) {
	builder.engine.newEndpoint(http.MethodHead, path, handler, builder.middleware)
}

func (builder endpointBuilder) POST(path string, handler Handler) {
	builder.engine.newEndpoint(http.MethodPost, path, handler, builder.middleware)
}

func (builder endpointBuilder) PUT(path string, handler Handler) {
	builder.engine.newEndpoint(http.MethodPut, path, handler, builder.middleware)
}

func (builder endpointBuilder) DELETE(path string, handler Handler) {
	builder.engine.newEndpoint(http.MethodDelete, path, handler, builder.middleware)
}

func (builder endpointBuilder) PATCH(path string, handler Handler) {
	builder.engine.newEndpoint(http.MethodPatch, path, handler, builder.middleware)
}
