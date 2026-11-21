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
	// [fs.FS] that will be used as the root for all static files.
	// You might find [fs.Sub] and [os.DirFS] helpful.
	FileSystem fs.FS
	// Middleware that will be applied to the static handler only.
	Middleware MiddlewareChain
	// Whether clients should be able to list static directories.
	// This is usually fine, either way they could simply be scanned for.
	AllowListingDirs bool
	// 'max-age' parameter of Cache-Control header.
	// You might find [MaxCacheAge] helpful.
	CacheMaxAge time.Duration
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
