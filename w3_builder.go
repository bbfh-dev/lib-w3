package libw3

import (
	"errors"
	"net/http"
	"os"

	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"github.com/julienschmidt/httprouter"
)

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

// Get the configured [Engine].
// Use [Builder.MustBuild] to exit on error instead of handling it manually.
func (builder *Builder) Build() (*Engine, error) {
	if len(builder.errs) != 0 {
		return nil, errors.Join(builder.errs...)
	}
	engine := &Engine{
		router:       httprouter.New(),
		pool:         nil,
		middleware:   builder.middleware,
		errorHandler: defaultErrorHandler,
	}
	engine.pool = internal.NewPool(builder.maxConcurrentRequests, engine.newContext)
	engine.router.NotFound = engine.wrapErrorHandler(func(ctx *Context) w3errs.HttpError {
		return w3errs.NewPublic(
			http.StatusNotFound,
			"Page Not Found: "+ctx.Request.URL.String(),
		)
	})
	engine.router.MethodNotAllowed = engine.wrapErrorHandler(func(ctx *Context) w3errs.HttpError {
		return w3errs.NewPublic(
			http.StatusNotFound,
			"Method Not Allowed: "+ctx.Request.Method,
		)
	})
	return engine, nil
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

func defaultErrorHandler(ctx *Context, err w3errs.HttpError) {
	if err.StatusCode >= 500 {
		Logger.Error(
			"%s %q → %s",
			ctx.Request.Method,
			ctx.Request.URL,
			err.InternalText,
		)
	}
	ctx.SendText(err.StatusCode, err.PublicText)
}
