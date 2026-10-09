package w3ware

import (
	"time"

	libw3 "codeberg.org/bbfh/lib-w3"
)

type requestLogger struct{}

var RequestLogger = &requestLogger{}

func (middleware *requestLogger) Handle(ctx *libw3.Context) {
	start := time.Now()
	ctx.ContinueChain()
	libw3.Logger.HttpRequest(ctx.Request, start, ctx.Writer.StatusCode)
}

func (middleware *requestLogger) Priority() int8 {
	return libw3.PrioTiming
}
