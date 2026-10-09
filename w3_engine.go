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

		ctx.reset()
		ctx.Writer = &ResponseWriter{ResponseWriter: writer}
		ctx.Request = request

		engine.errorHandler(ctx, err_getter(ctx))
	})
}
