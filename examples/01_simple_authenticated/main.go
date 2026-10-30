package main

import (
	"net/http"
	"os"

	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3store"
	"codeberg.org/bbfh/lib-w3/w3ware"
)

var ContextAuth w3store.Holder[int]

func main() {
	engine := libw3.NewBuilder().
		Use(w3ware.Recovery).
		Use(&w3ware.SessionAuth[int]{
			Cookie:         "session",
			ContextSession: &ContextAuth,
			Authenticator: func(sessionId string) (session int, passed bool) {
				return 123, sessionId == "abc"
			},
			Required:          false,
			IgnoreStaticFiles: true,
		}).
		MustBuild()

	engine.GET(
		"/:value",
		func(ctx *libw3.Context) {
			id, ok := ctx.Get(ContextAuth)
			if !ok {
				ctx.SendTextf(
					http.StatusOK,
					"Hello %s, unauthenticated!",
					ctx.Params.ByName("value"),
				)
			} else {
				ctx.SendTextf(
					http.StatusOK,
					"Hello %s, authenticated with id='%d'!",
					ctx.Params.ByName("value"),
					id,
				)
			}
		},
	)

	err := engine.ListenAndServe(":8081")
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
}
