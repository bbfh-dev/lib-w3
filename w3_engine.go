package libw3

import (
	"net/http"

	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3store"
	"github.com/julienschmidt/httprouter"
)

// Refer to [NewBuilder].
type Engine struct {
	endpointBuilder
	router       *httprouter.Router
	pool         *internal.Pool[Context]
	middleware   MiddlewareChain
	errorHandler ErrorHandler
}

func (engine *Engine) ListenAndServe(addr string) error {
	Logger.Info("Listening and serving on %s", addr)
	return http.ListenAndServe(addr, engine.router)
}

func (engine *Engine) newContext() Context {
	return Context{
		store:        make([]any, w3store.ContextStoreHolders()),
		errorHandler: engine.errorHandler,
	}
}

func (engine *Engine) wrapErrorHandler(errGetter func(*Context) w3errs.HttpError) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		objectIndex, ctx := engine.pool.AcquireObject()
		defer engine.pool.Release(objectIndex)

		ctx.reset(writer, request)

		engine.errorHandler(ctx, errGetter(ctx))
	})
}

func (engine *Engine) newEndpoint(
	method string,
	path string,
	handler Handler,
	localMiddleware MiddlewareChain,
) {
	middleware := append(engine.middleware, localMiddleware...)
	suitable := middleware.Filtered(method, path)
	suitable.Sort()
	handlers := append(suitable.Handlers(), handler)

	Logger.Debug("New endpoint: %s %q using:\n\t%s → (Handler)", method, path, suitable.String())

	engine.router.Handle(
		method,
		path,
		func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
			objectIndex, ctx := engine.pool.AcquireObject()
			defer engine.pool.Release(objectIndex)

			ctx.reset(writer, request)
			ctx.Params = params
			ctx.handlers = handlers

			ctx.executeHandlers()
		},
	)
}
