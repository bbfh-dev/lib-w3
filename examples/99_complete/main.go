package main

import (
	"net/http"
	"os"
	"path/filepath"

	libclog "codeberg.org/bbfh/lib-clog"
	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3store"
	"codeberg.org/bbfh/lib-w3/w3ware"
)

type Session struct {
	Id        int
	CsrfToken string
}

var ContextAuth w3store.Holder[Session]

func init() {
	libw3.Logger.SetLevel(libclog.LogDebug)
}

func main() {
	workdir, _ := os.Getwd()
	engine := libw3.NewBuilder().
		Use(w3ware.Recovery).
		Use(&w3ware.CsrfProtection{
			TokenExtractor: func(ctx *libw3.Context) (token string) {
				session, ok := ctx.Get(ContextAuth)
				if !ok {
					return ""
				}
				return session.CsrfToken
			},
		}).
		Use(w3ware.RequestLogger).
		Use(&w3ware.SessionAuth[Session]{
			Cookie:         "session",
			ContextSession: &ContextAuth,
			Authenticator: func(session_id string) (session Session, passed bool) {
				return Session{
					Id:        123,
					CsrfToken: "hello_world!",
				}, session_id == "abc"
			},
			Required:          false,
			IgnoreStaticFiles: true,
		}).
		WithStaticHandler(libw3.StaticConfig{
			FileSystem:       os.DirFS(filepath.Join(workdir, "www", "static")),
			Middleware:       libw3.MiddlewareChain{},
			AllowListingDirs: true,
			CacheMaxAge:      libw3.MaxCacheAge,
		}).
		MustBuild()

	engine.
		Using(libw3.FromHandler(CustomMiddleware)).
		GET("/benchmark", func(ctx *libw3.Context) {
			ctx.SendText(http.StatusCreated, "Done.")
		})

	engine.PUT("/upload", func(ctx *libw3.Context) {
		value := ctx.Request.FormValue("key")
		ctx.SendTextf(http.StatusOK, "Received %q", value)
	})

	err := engine.ListenAndServe(":8081")
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
}

func CustomMiddleware(ctx *libw3.Context) {
	ctx.ContinueChain()
	ctx.Writer.Write([]byte("\nMessage from middleware! :)"))
}
