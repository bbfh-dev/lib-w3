package w3ware

import (
	"time"

	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type requestLogger struct{}

var RequestLogger = &requestLogger{}

func (middleware *requestLogger) Handle(ctx *libw3.Context) {
	start := time.Now()
	ctx.ContinueChain()
	w3utils.Logger.HttpRequest(ctx.Request, start, ctx.Writer.StatusCode)
}

func (middleware *requestLogger) Priority() int8 {
	return libw3.PrioTiming
}

var printedWarning bool

func (middleware *requestLogger) Init() (keep bool) {
	if !printedWarning {
		w3utils.Logger.Warn(
			"Using %q middleware should not be done in production due to performance concerns",
			w3utils.GetName(middleware),
		)
		printedWarning = true
	}
	return true
}
