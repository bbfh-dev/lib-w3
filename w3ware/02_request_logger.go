package w3ware

import libw3 "codeberg.org/bbfh/lib-w3"

type requestLogger struct{}

var RequestLogger = &requestLogger{}

func (middleware *requestLogger) Handle(ctx *libw3.Context) {
}

func (middleware *requestLogger) Priority() int8 {
	return libw3.PrioTiming
}
