package libw3

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"codeberg.org/bbfh/lib-w3/w3errs"
)

var MaxCacheAge = 24 * 365 * time.Hour

type StaticConfig struct {
	FileSystem       fs.FS
	Middleware       MiddlewareChain
	AllowListingDirs bool
	CacheMaxAge      time.Duration
}

func (config StaticConfig) CreateHandler() Handler {
	fileServer := http.FileServer(http.FS(config.FileSystem))
	return func(ctx *Context) {
		path := ctx.Request.URL.Path // recover the actual path for middleware

		ctx.Request.URL.Path = ctx.Params.ByName("filepath")
		if !config.AllowListingDirs && strings.HasSuffix(ctx.Request.URL.Path, "/") {
			ctx.AbortWithError(w3errs.NewSimple(http.StatusForbidden))
			return
		}

		ctx.Writer.Header().Set(
			"Cache-Control",
			fmt.Sprintf("public, max-age=%.f, immutable", config.CacheMaxAge.Seconds()),
		)
		fileServer.ServeHTTP(ctx.Writer, ctx.Request)

		ctx.Request.URL.Path = path
	}
}
