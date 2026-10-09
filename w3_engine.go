package libw3

import (
	"net/http"

	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3store"
	"github.com/julienschmidt/httprouter"
)

type Engine struct {
	router       *httprouter.Router
	pool         *internal.Pool[Context]
	middleware   MiddlewareChain
	errorHandler ErrorHandler
}

func (engine *Engine) ListenAndServe(addr string) error {
	Logger.Info("Listening and serving on %s", addr)
	return http.ListenAndServe(addr, engine.router)
}

func (engine *Engine) GET(path string, handler Handler) {
	engine.newEndpoint(http.MethodGet, path, handler, MiddlewareChain{})
}

func (engine *Engine) HEAD(path string, handler Handler) {
	engine.newEndpoint(http.MethodHead, path, handler, MiddlewareChain{})
}

func (engine *Engine) POST(path string, handler Handler) {
	engine.newEndpoint(http.MethodPost, path, handler, MiddlewareChain{})
}

func (engine *Engine) PUT(path string, handler Handler) {
	engine.newEndpoint(http.MethodPut, path, handler, MiddlewareChain{})
}

func (engine *Engine) DELETE(path string, handler Handler) {
	engine.newEndpoint(http.MethodDelete, path, handler, MiddlewareChain{})
}

func (engine *Engine) PATCH(path string, handler Handler) {
	engine.newEndpoint(http.MethodPatch, path, handler, MiddlewareChain{})
}

func (engine *Engine) newContext() Context {
	return Context{
		store:        make([]any, w3store.ContextStoreHolders()),
		errorHandler: engine.errorHandler,
	}
}

func (engine *Engine) wrapErrorHandler(err_getter func(*Context) w3errs.HttpError) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		object_index, ctx := engine.pool.AcquireObject()
		defer engine.pool.Release(object_index)

		ctx.reset(writer, request)

		engine.errorHandler(ctx, err_getter(ctx))
	})
}

func (engine *Engine) newEndpoint(
	method string,
	path string,
	handler Handler,
	local_middleware MiddlewareChain,
) {
	middleware := append(engine.middleware, local_middleware...)
	suitable := middleware.Filtered(method, path)
	suitable.Sort()
	handlers := append(suitable.Handlers(), handler)

	Logger.Debug("New endpoint: %s %q using:\n\t%s → (Handler)", method, path, suitable.String())

	engine.router.Handle(
		method,
		path,
		func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
			object_index, ctx := engine.pool.AcquireObject()
			defer engine.pool.Release(object_index)

			ctx.reset(writer, request)
			ctx.Params = params
			ctx.handlers = handlers

			ctx.executeHandlers()
		},
	)
}
