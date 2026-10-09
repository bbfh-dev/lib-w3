package libw3

import "codeberg.org/bbfh/lib-w3/w3utils"

type basicMiddleware struct {
	handler func(ctx *Context)
	name    string
}

// Create [Middleware] from a [Handler].
func FromHandler(handler Handler) Middleware {
	return &basicMiddleware{
		handler: handler,
		name:    w3utils.GetFuncName(handler),
	}
}

func (middleware *basicMiddleware) Handle(ctx *Context) {
	middleware.handler(ctx)
}

func (middleware *basicMiddleware) Name() string {
	return middleware.name
}
