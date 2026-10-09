package w3ware

import (
	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type csrfTokenExtractor = func(ctx *libw3.Context) (token string)

// CsrfProtection validates Cross-site request forgery (CSRF) tokens on POST/PATCH/PUT/DELETE requests.
// The token should be extracted from user's current session, so put this middleware after the authenticator/session manager.
type CsrfProtection struct {
	TokenExtractor csrfTokenExtractor
}

func (middleware *CsrfProtection) Handle(ctx *libw3.Context) {
}

func (middleware *CsrfProtection) ShouldInject(method, path string) bool {
	return w3utils.IsStateModifyingMethod(method)
}

func (middleware *CsrfProtection) Priority() int8 {
	return libw3.PrioValidate
}
