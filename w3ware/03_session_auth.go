package w3ware

import (
	"net/http"
	"strings"

	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3store"
)

var (
	ContextSessionError       = w3store.AllocateContextStoreHolder[w3errs.HttpError]()
	contextSessionPlaceholder = w3store.AllocateContextStoreHolder[any]()
	errUnauthorized           = w3errs.NewSimple(http.StatusUnauthorized)
	errInvalidSession         = w3errs.NewPublic(
		http.StatusUnauthorized,
		"Invalid session, authentication denied by the server",
	)
)

type authenticator[T any] func(sessionId string) (session T, passed bool)

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
	//	userSession, ok := ctx.Get(ContextExample)
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
	var err w3errs.HttpError
	var credentials string

	sessionCookie, exists := ctx.Cookie(middleware.Cookie)
	if exists {
		credentials = sessionCookie
		goto authenticate
	} else {
		value := ctx.Request.Header.Get("Authorization")
		if value == "" {
			err = errUnauthorized
			goto report_error
		}

		parts := strings.SplitN(value, " ", 2)
		if len(parts) != 2 {
			err = errUnauthorized
			goto report_error
		}

		authType, authCredentials := parts[0], parts[1]
		if authType != "Bearer" {
			err = w3errs.NewPublic(
				http.StatusUnauthorized,
				"Authorization header must be a Bearer token, got: "+authType,
			)
			goto report_error
		}
		credentials = authCredentials
		goto authenticate
	}

report_error:
	if middleware.Required {
		ctx.AbortWithError(err)
	} else {
		ctx.Set(ContextSessionError, err)
	}
	return

authenticate:
	session, ok := middleware.Authenticator(credentials)
	if !ok {
		ctx.ClearCookie(middleware.Cookie)
		err = errInvalidSession
		goto report_error
	}
	ctx.Set(*middleware.ContextSession, session)
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
