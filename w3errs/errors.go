package w3errs

import (
	"os"

	libclog "codeberg.org/bbfh/lib-clog"
)

type WritableError interface {
	Write(out *os.File, mode libclog.OutputModeEnum) error
}

func Write(err error, out *os.File, mode libclog.OutputModeEnum) error {
	if derr, ok := err.(WritableError); ok {
		return derr.Write(out, mode)
	}
	_, e := out.WriteString(err.Error())
	return e
}
