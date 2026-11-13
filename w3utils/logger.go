package w3utils

import (
	"os"

	libclog "codeberg.org/bbfh/lib-clog"
)

// Feel free to modify [libclog.Logger.SetLevel], [libclog.Logger.UseSyslog] & [libclog.Logger.SetOutputMode].
var Logger = libclog.NewLogger("W3", os.Stderr)
