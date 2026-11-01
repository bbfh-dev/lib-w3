package w3errs

import (
	"fmt"
	"os"
	"strings"

	libclog "codeberg.org/bbfh/lib-clog"
)

type DocumentError struct {
	Filename     string
	ContextLines []string
	FirstLine    uint
	Highlight    string
	Message      string
}

func (err DocumentError) Error() string {
	return fmt.Sprintf(
		"%s:%d: %s",
		err.Filename,
		err.FirstLine+uint(len(err.ContextLines)),
		err.Message,
	)
}

func (err DocumentError) Write(file *os.File) error {
	printer := libclog.NewPrinter(file, libclog.OutputAuto)
	printer.Coloredf(libclog.ColorBrightBlack, "─────┬%s\n", strings.Repeat("─", 30))
	printer.Colored(libclog.ColorBrightBlack, "     │ [!] Error: ")
	printer.Coloredln(libclog.ColorBrightRed, err.Message)
	printer.Colored(libclog.ColorBrightBlack, "     │ in ")
	printer.Coloredf(
		libclog.ColorBrightWhite,
		"%s:%d\n",
		err.Filename,
		err.FirstLine+uint(len(err.ContextLines))-1,
	)
	printer.Coloredf(libclog.ColorBrightBlack, "─────┼%s", strings.Repeat("─", 30))
	for i, line := range err.ContextLines {
		printer.Coloredf(libclog.ColorBrightBlack, "\n%4d │ %s", err.FirstLine+uint(i), line)
	}

	highlightedLines := strings.Split(err.Highlight, "\n")
	printer.Colored(libclog.ColorBrightRed, highlightedLines[0])
	firstLine := err.FirstLine + uint(len(err.ContextLines))
	for i, line := range highlightedLines[1:] {
		printer.Coloredf(libclog.ColorBrightBlack, "\n%4d │ ", firstLine+uint(i))
		printer.Colored(libclog.ColorBrightRed, line)
	}
	var length int
	if len(highlightedLines) == 1 {
		length = len(highlightedLines[0]) + len(err.ContextLines[len(err.ContextLines)-1])
	} else {
		length = len(highlightedLines[len(highlightedLines)-1])
	}
	printer.Coloredf(
		libclog.ColorBrightRed,
		"\n%s↑\n",
		strings.Repeat(" ", 7+length-1),
	)

	return printer.ErrorsJoined()
}
