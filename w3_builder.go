package libw3

import (
	"io/fs"
	"net/http"
	"os"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"codeberg.org/bbfh/lib-w3/w3utils"
	"github.com/julienschmidt/httprouter"
)

// Refer to [NewBuilder].
type Builder struct {
	maxConcurrentRequests uint16
	middleware            MiddlewareChain
	pages                 []w3html.Renderable
	errs                  []error
	staticConfig          *StaticConfig
	errorHandler          ErrorHandler
}

// Used to create a configured [Engine].
func NewBuilder() *Builder {
	return &Builder{
		maxConcurrentRequests: 200,
		middleware:            MiddlewareChain{},
		pages:                 []w3html.Renderable{},
		errs:                  []error{},
		staticConfig:          nil,
		errorHandler:          defaultErrorHandler,
	}
}

// LimitConcurrentRequests sets the pool of concurrent requests.
// Any additional requests will have to wait for another request to finish before being served.
func (builder *Builder) LimitConcurrentRequests(limit uint16) *Builder {
	builder.maxConcurrentRequests = limit
	return builder
}

// Use [Middleware] that will be applied to ALL endpoints.
// Can be called multiple times.
func (builder *Builder) Use(middleware Middleware) *Builder {
	builder.middleware = builder.middleware.ProperAppend(middleware)
	return builder
}

// LoadPage parses, bakes, and stores the provided HTML document.
// The [w3html.Renderable] can be accessed using the provided [w3store.Holder] key in [Context.SendPage].
func (builder *Builder) LoadPage[T any](
	filename string,
	holder *w3utils.PageHolder[T],
) *Builder {
	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		builder.errs = append(builder.errs, err)
		return builder
	}
	defer file.Close()

	var fsys fs.FS
	if builder.staticConfig != nil {
		fsys = builder.staticConfig.FileSystem
	}
	parser := w3html.NewParser[T](file, map[string]string{}, fsys)
	page, err := parser.Parse()
	if err != nil {
		builder.errs = append(builder.errs, err)
		return builder
	}

	*holder = w3utils.PageHolder[T](len(builder.pages))
	builder.pages = append(builder.pages, w3html.Bake(page.Nodes))

	return builder
}

// WithStaticHandler specifies [StaticConfig] to be used by the [Builder].
// Providing this will create a new endpoint `GET /static/*filepath`.
func (builder *Builder) WithStaticHandler(config StaticConfig) *Builder {
	builder.staticConfig = &config
	return builder
}

// WithErrorHandler specifies [ErrorHandler] to be used by the [Engine] & [Context].
// Providing this allows errors to be reported as HTML/JSON instead of plain text.
func (builder *Builder) WithErrorHandler(handler ErrorHandler) *Builder {
	builder.errorHandler = handler
	return builder
}

// Get the configured [Engine].
// Use [Builder.MustBuild] to exit on error instead of handling it manually.
func (builder *Builder) Build() (*Engine, error) {
	if len(builder.errs) != 0 {
		return nil, builder.errs[0]
	}
	engine := &Engine{
		router:       httprouter.New(),
		pool:         nil,
		middleware:   builder.middleware,
		pages:        builder.pages,
		errorHandler: builder.errorHandler,
	}
	engine.endpointBuilder = newEndpointBuilder(engine)
	engine.pool = internal.NewPool(builder.maxConcurrentRequests, engine.newContext)

	engine.router.NotFound = engine.wrapErrorHandler(func(ctx *Context) w3errs.HttpError {
		return w3errs.NewPublic(
			http.StatusNotFound,
			"Page Not Found: "+ctx.Request.URL.String(),
		)
	})
	engine.router.MethodNotAllowed = engine.wrapErrorHandler(func(ctx *Context) w3errs.HttpError {
		return w3errs.NewPublic(
			http.StatusNotFound,
			"Method Not Allowed: "+ctx.Request.Method,
		)
	})

	if builder.staticConfig != nil {
		engine.newEndpoint(
			http.MethodGet,
			"/static/*filepath",
			builder.staticConfig.CreateHandler(),
			builder.staticConfig.Middleware,
		)
	}

	return engine, nil
}

// MustBuild calls [Builder.Build] and calls [os.Exit] on error.
func (builder *Builder) MustBuild() *Engine {
	engine, err := builder.Build()
	if err != nil {
		if derr, ok := err.(w3errs.WritableError); ok {
			derr.Write(os.Stderr, libclog.OutputAuto)
		}
		w3utils.Logger.Fatal("%s", err.Error())
		os.Exit(1)
	}
	return engine
}

func defaultErrorHandler(ctx *Context, err w3errs.HttpError) {
	if err.StatusCode >= 500 {
		w3utils.Logger.Error(
			"%s %q → %s",
			ctx.Request.Method,
			ctx.Request.URL,
			err.InternalText,
		)
	}
	ctx.SendText(err.StatusCode, err.PublicText)
}
