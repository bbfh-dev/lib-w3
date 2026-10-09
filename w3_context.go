package libw3

import (
	"fmt"
	"math"
	"net/http"
	"net/url"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3store"
	"github.com/julienschmidt/httprouter"
)

const abortCursor uint = math.MaxUint

// ErrorHandler is responsible for handling/reporting the error to the client.
// (e.g. render a proper HTML error page rather than plain text error)
type ErrorHandler func(ctx *Context, err w3errs.HttpError)

// Context represents the current request.
// It passes variables between middleware, validates requests, manages the flow and sends the response.
type Context struct {
	Writer  *ResponseWriter // Wrapper around [http.ResponseWriter], can be used directly if needed.
	Request *http.Request
	Params  httprouter.Params // The URL parameters from the router.

	handlers []Handler // Middleware + endpoint
	cursor   uint      // handler that is being currently executed in the chain. May be set to [abortCursor]

	store        []any        // arbitrary data passed between middleware. Refer to [w3store.Holder].
	errorHandler ErrorHandler // refer to [ErrorHandler] documentation.
}

// reset is called when this object is being reused from the pool.
func (ctx *Context) reset(writer http.ResponseWriter, request *http.Request) {
	ctx.cursor = 0
	for i := range ctx.store {
		ctx.store[i] = nil
	}
	ctx.Writer = &ResponseWriter{ResponseWriter: writer}
	ctx.Request = request
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

// AbortWithError calls [Context.Abort] after sending the error to the client.
func (ctx *Context) AbortWithError(err w3errs.HttpError) {
	ctx.errorHandler(ctx, err)
	ctx.Abort()
}

// Executes the rest of the handlers in the chain.
//
// Used to run code after all other handlers (logging, recovery, statistics, etc.)
func (ctx *Context) ContinueChain() {
	if ctx.HasAborted() {
		return
	}
	ctx.cursor++
	ctx.executeHandlers()
}

func (ctx *Context) executeHandlers() {
	length := uint(len(ctx.handlers))
	for ; ctx.cursor < length; ctx.cursor++ {
		if handler := ctx.handlers[ctx.cursor]; handler != nil {
			handler(ctx)
		}
		if ctx.HasAborted() {
			return
		}
	}
}

// Set stores a value in the context for later use.
// Use the holder returned by [w3store.AllocateContextStoreHolder] during creation.
// Any middleware should expose those at the top-level.
func (ctx *Context) Set[T any](holder w3store.Holder[T], value T) {
	ctx.store[holder] = value
}

// Retrieves the stored value from context.
// Use the holder returned by [w3store.AllocateContextStoreHolder] during creation.
// Any middleware should expose those at the top-level.
func (ctx *Context) Get[T any](holder w3store.Holder[T]) (T, bool) {
	if int(holder) >= len(ctx.store) {
		Logger.Error("Context.Get(): Trying to access out-of-bounds store")
		goto return_empty
	}

	if ctx.store[holder] == nil {
		goto return_empty
	}

	if value, ok := ctx.store[holder].(T); ok {
		return value, true
	}

return_empty:
	var empty T
	return empty, false
}

// Cookie retrieves a named cookie from the request.
func (ctx *Context) Cookie(name string) (string, bool) {
	cookie, err := ctx.Request.Cookie(name)
	if err != nil {
		return "", false
	}
	value, _ := url.QueryUnescape(cookie.Value)
	return value, true
}

// ClearCookie will remove the cookie from the client.
func (ctx *Context) ClearCookie(name string) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// SendText writes the header and sends plain text response.
// Use [Context.Writer] directly to write any additional data.
func (ctx *Context) SendText(status_code int, text string) {
	ctx.SendContentType("text/plain; charset=utf-8")
	ctx.Writer.WriteHeader(status_code)
	ctx.Writer.Write([]byte(text))
}

// SendTextf writes the header and sends plain text response using [fmt.Fprintf].
// Use [Context.Writer] directly to write any additional data.
func (ctx *Context) SendTextf(status_code int, format string, args ...any) {
	ctx.SendContentType("text/plain; charset=utf-8")
	ctx.Writer.WriteHeader(status_code)
	fmt.Fprintf(ctx.Writer, format, args...)
}

// SendEmpty writes 204 No Content.
func (ctx *Context) SendEmpty() {
	ctx.Writer.WriteHeader(http.StatusNoContent)
}

// SendContentType is sugar code for setting specific content type.
func (ctx *Context) SendContentType(content_type string) {
	ctx.Writer.Header().Set("Content-Type", content_type)
}
