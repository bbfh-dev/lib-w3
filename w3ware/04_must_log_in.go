package w3ware

import (
	"net/http"

	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3errs"
)

type mustLogIn struct{}

var MustLogIn = &mustLogIn{}

func (middleware *mustLogIn) Handle(ctx *libw3.Context) {
	err, ok := ctx.Get(ContextSessionError)
	if ok {
		if err == errUnauthorized {
			ctx.AbortWithError(
				w3errs.NewPublic(http.StatusForbidden, "Must be logged in to view this page"),
			)
		} else {
			ctx.AbortWithError(err)
		}
	}
}

func (middleware *mustLogIn) Priority() int8 {
	return libw3.PrioValidate
}
