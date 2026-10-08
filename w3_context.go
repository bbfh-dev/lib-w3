package libw3

import (
	"net/http"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"github.com/julienschmidt/httprouter"
)

// ErrorHandler is responsible for handling/reporting the error to the client.
type ErrorHandler func(ctx *Context, err w3errs.HttpError)

type Context struct {
	Writer       ResponseWriter
	Request      *http.Request
	Params       httprouter.Params
	errorHandler ErrorHandler
}
