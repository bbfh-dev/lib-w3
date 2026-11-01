package w3errs

import (
	"os"
)

type WritableError interface {
	Write(out *os.File) error
}

func Write(err error, out *os.File) error {
	if derr, ok := err.(WritableError); ok {
		return derr.Write(out)
	}
	_, e := out.WriteString(err.Error())
	return e
}
