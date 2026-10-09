package w3ware

import (
	"strings"

	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3store"
)

var ContextSessionError = w3store.AllocateContextStoreHolder[w3errs.HttpError]()
var contextSessionPlaceholder = w3store.AllocateContextStoreHolder[any]()

type authenticator[T any] func(session_id string) (session T, passed bool)

// Middleware to authenticate requests.
// Uses specified cookie by default, falls back to Bearer HTTP Authorization otherwise.
type SessionAuth[T any] struct {
	// The name of the cookie that contains session token
	Cookie string
	// Pointer to a [w3store.Holder] that will be set on initialization.
	// Example usage:
	//	var ContextExample w3store.Holder[UserSession]
	//	w3ware.SessionAuth{ContextSession: &ContextExample}
	// Then query it from the [libw3.Context] using:
	//	user_session, ok := ctx.Get(ContextExample)
	ContextSession *w3store.Holder[T]
	// Function that validates the token and returns session data to pass down to other handlers.
	Authenticator authenticator[T]
	// Whether the user must be authenticated (aborts on errors).
	//
	// Authentication errors can be retrieved using
	//	err, _ := ctx.Get(w3store.ContextSessionError)
	Required bool
	// Skip authenticating /static/* requests (assuming the files are public)
	IgnoreStaticFiles bool
}

func (middleware *SessionAuth[T]) Handle(ctx *libw3.Context) {
}

func (middleware *SessionAuth[T]) ShouldInject(method, path string) bool {
	return !(middleware.IgnoreStaticFiles && strings.HasPrefix(path, "/static"))
}

func (middleware *SessionAuth[T]) Init() (keep bool) {
	*middleware.ContextSession = w3store.Holder[T](contextSessionPlaceholder)
	return true
}

func (middleware *SessionAuth[T]) Priority() int8 {
	return libw3.PrioAuth
}
