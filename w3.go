package libw3

import (
	"os"

	libclog "codeberg.org/bbfh/lib-clog"
)

var Logger = libclog.NewLogger("W3", os.Stderr)
