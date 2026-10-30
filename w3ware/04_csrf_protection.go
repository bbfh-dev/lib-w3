package w3ware

import (
	"crypto/subtle"
	"net/http"

	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type csrfTokenExtractor = func(ctx *libw3.Context) (token string)

// CsrfProtection validates Cross-site request forgery (CSRF) tokens on POST/PATCH/PUT/DELETE requests.
// The token should be extracted from user's current session, so put this middleware after the authenticator/session manager.
type CsrfProtection struct {
	TokenExtractor csrfTokenExtractor
}

func (middleware *CsrfProtection) Handle(ctx *libw3.Context) {
	expectedCsrf := middleware.TokenExtractor(ctx)
	if expectedCsrf == "" {
		ctx.AbortWithError(w3errs.NewPublic(
			http.StatusForbidden,
			"Not authorized to access this page",
		))
		return
	}

	if err := ctx.Request.ParseForm(); err != nil {
		ctx.AbortWithError(w3errs.HttpError{
			StatusCode:   http.StatusBadRequest,
			InternalText: err.Error(),
			PublicText:   "Invalid form format during CSRF token validation",
		})
		return
	}

	requestCsrf := ctx.Request.FormValue("csrf_token")
	if requestCsrf == "" {
		ctx.AbortWithError(w3errs.NewPublic(
			http.StatusBadRequest,
			"Missing 'csrf_token' field in the request",
		))
		return
	}

	if subtle.ConstantTimeCompare([]byte(requestCsrf), []byte(expectedCsrf)) != 1 {
		ctx.AbortWithError(w3errs.NewPublic(
			http.StatusBadRequest,
			"Invalid 'csrf_token' field",
		))
		return
	}
}

func (middleware *CsrfProtection) ShouldInject(method, path string) bool {
	return w3utils.IsStateModifyingMethod(method)
}

func (middleware *CsrfProtection) Priority() int8 {
	return libw3.PrioValidate
}
