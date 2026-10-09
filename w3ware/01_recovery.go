package w3ware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"syscall"

	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3errs"
)

type recovery struct{}

var Recovery = &recovery{}

func (middleware *recovery) Handle(ctx *libw3.Context) {
	separator := strings.Repeat("-", 40)
	// Credit: github.com/gin-gonic/gin/recovery.go
	defer func() {
		if recovery := recover(); recovery != nil {
			fmt.Println(separator)
			// Check for a broken connection, as it is not really a
			// condition that warrants a panic stack trace.
			var is_broken_pipe bool
			err, ok := recovery.(error)
			if ok {
				is_broken_pipe = errors.Is(err, syscall.EPIPE) ||
					errors.Is(err, syscall.ECONNRESET) ||
					errors.Is(err, http.ErrAbortHandler)
			} else {
				err = fmt.Errorf("%v", recovery)
			}

			if is_broken_pipe {
				libw3.Logger.Error("[Recovery] %s", recovery)
			} else {
				libw3.Logger.Error(
					"(Recovered) %s\n\n[Request]\n%s[Stack Trace]\n%s",
					recovery,
					internal.DumpRequest(ctx.Request),
					internal.DumpStack(),
				)
			}
			fmt.Println(separator)
			if !is_broken_pipe {
				ctx.AbortWithError(w3errs.NewInternal(http.StatusInternalServerError, err))
			}
		}
	}()

	ctx.ContinueChain()
}

func (middleware *recovery) Priority() int8 {
	return libw3.PrioRecovery
}
