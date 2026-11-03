package w3errs

import (
	"fmt"
	"os"
	"slices"
	"strings"

	libclog "codeberg.org/bbfh/lib-clog"
)

type DocumentLocation struct {
	Filename string
	Line     uint
}

func (location DocumentLocation) String() string {
	return fmt.Sprintf("%s:%d", location.Filename, location.Line)
}

type DocumentCallStack []DocumentLocation

func (stack DocumentCallStack) String() string {
	switch len(stack) {
	case 0:
		return ""
	case 1:
		return stack[0].String()
	}

	var builder strings.Builder
	last := len(stack) - 1
	builder.WriteString(stack[last].String())
	for _, location := range slices.Backward(stack[:last]) {
		builder.WriteString(": ")
		builder.WriteString(location.String())
	}
	return builder.String()
}

type DocumentError struct {
	CallStack    DocumentCallStack
	ContextLines []string
	Highlight    string
	Message      string
}

func (err DocumentError) Error() string {
	return fmt.Sprintf(
		"%s: %s",
		err.CallStack,
		err.Message,
	)
}

func (err DocumentError) Write(file *os.File, mode libclog.OutputModeEnum) error {
	printer := libclog.NewPrinter(file, mode)
	printer.Coloredf(libclog.ColorBrightBlack, "─────┬%s\n", strings.Repeat("─", 30))
	printer.Colored(libclog.ColorBrightBlack, "     │ [!] Error: ")
	printer.Coloredln(libclog.ColorBrightRed, err.Message)

	var firstLine uint = 0
	if len(err.CallStack) != 0 {
		last := len(err.CallStack) - 1
		firstLine = err.CallStack[last].Line
		printer.Colored(libclog.ColorBrightBlack, "     │ in ")
		printer.Coloredln(libclog.ColorBrightWhite, err.CallStack[last].String())
		for _, location := range slices.Backward(err.CallStack[:last]) {
			printer.Colored(libclog.ColorBrightBlack, "     │ └ from ")
			printer.Coloredln(libclog.ColorBrightWhite, location.String())
		}
	}

	printer.Coloredf(libclog.ColorBrightBlack, "─────┼%s", strings.Repeat("─", 30))
	for i, line := range err.ContextLines {
		printer.Coloredf(libclog.ColorBrightBlack, "\n%4d │ %s", firstLine+uint(i), line)
	}

	highlightedLines := strings.Split(err.Highlight, "\n")
	printer.Colored(libclog.ColorBrightRed, highlightedLines[0])
	firstLine += uint(len(err.ContextLines))
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
	printer.Colored(
		libclog.ColorBrightBlack,
		"\n     │ ",
	)
	printer.Coloredf(
		libclog.ColorBrightRed,
		"%s↑\n",
		strings.Repeat(" ", length-1),
	)
	printer.Coloredf(libclog.ColorBrightBlack, "─────┴%s\n", strings.Repeat("─", 30))

	return printer.ErrorsJoined()
}
