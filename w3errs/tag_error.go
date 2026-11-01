package w3errs

import (
	"fmt"
	"os"
	"strings"

	libclog "codeberg.org/bbfh/lib-clog"
)

type TagError struct {
	Filename   string
	LineNumber uint
	Context    string
	Message    string
}

func (err TagError) Error() string {
	return fmt.Sprintf(
		"%s:%d: %s",
		err.Filename,
		err.LineNumber,
		err.Message,
	)
}

func (err TagError) Write(file *os.File) error {
	printer := libclog.NewPrinter(file, libclog.OutputAuto)
	printer.Coloredf(libclog.ColorBrightBlack, "─────┬%s\n", strings.Repeat("─", 30))
	printer.Colored(libclog.ColorBrightBlack, "     │ [!] Error: ")
	printer.Coloredln(libclog.ColorBrightRed, err.Message)
	printer.Colored(libclog.ColorBrightBlack, "     │ in ")
	printer.Coloredf(libclog.ColorBrightWhite, "%s:%d\n", err.Filename, err.LineNumber)
	printer.Coloredf(libclog.ColorBrightBlack, "─────┼%s", strings.Repeat("─", 30))
	printer.Coloredf(libclog.ColorBrightBlack, "\n     │ %s\n", err.Context)

	return printer.ErrorsJoined()
}
