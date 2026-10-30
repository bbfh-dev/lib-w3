package w3errs

import (
	"os"
)

func Write(err error, out *os.File) error {
	if derr, ok := err.(DocumentError); ok {
		return derr.Write(out)
	}
	_, e := out.WriteString(err.Error())
	return e
}
