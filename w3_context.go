package libw3

import (
	"math"
	"net/http"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"github.com/julienschmidt/httprouter"
)

const abortCursor uint = math.MaxUint

// ErrorHandler is responsible for handling/reporting the error to the client.
type ErrorHandler func(ctx *Context, err w3errs.HttpError)

type Context struct {
	Writer  ResponseWriter
	Request *http.Request
	Params  httprouter.Params

	handlers []Handler
	cursor   uint

	errorHandler ErrorHandler
}

// Abort the request.
// Stops processing the request gracefully — any active handlers will not be killed, but no additional handlers will be called (stops the chain).
func (ctx *Context) Abort() {
	ctx.cursor = abortCursor
}

// HasAborted returns whether the context has been aborted.
func (ctx *Context) HasAborted() bool {
	return ctx.cursor == abortCursor
}
