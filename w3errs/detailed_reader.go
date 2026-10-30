package w3errs

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type DetailedReader struct {
	internal *bufio.Reader
	document *internal.RingBuffer[rune]
	line     uint
	mark     uint
	head     uint
	rewinded uint
}

func NewDetailedReader(reader *bufio.Reader, expectedSize uint64) *DetailedReader {
	return &DetailedReader{
		internal: reader,
		document: internal.NewRingBuffer[rune](),
		line:     1,
		mark:     0,
		head:     0,
		rewinded: 0,
	}
}

// Read the next rune.
func (reader *DetailedReader) Read() (rune, error) {
	char, _, err := reader.internal.ReadRune()
	reader.head++
	if char == '\n' {
		reader.line++
	}
	if reader.rewinded == 0 {
		reader.document.Push(char)
	} else {
		reader.rewinded--
	}
	return char, err
}

// Unread last read rune.
func (reader *DetailedReader) Unread() {
	reader.internal.UnreadRune()
	reader.head--
	reader.rewinded++
}

// MustUnread is similar to [DetailedReader.Unread], but panics on error.
func (reader *DetailedReader) MustUnread() {
	if err := reader.internal.UnreadRune(); err != nil {
		panic("trying to unread a character, but it was never read to begin with!")
	}
}

// MarkHere marks the current position as the beginning of context.
//
// e.g. the position where parser began reading an object name.
func (reader *DetailedReader) MarkHere() {
	reader.mark = reader.head
	if reader.head != 0 {
		reader.mark--
	}
}

// WrapError creates a [DetailedError] with current state context.
// Use [DetailedError.Write] to pretty-print the error to file/console.
func (reader *DetailedReader) WrapError(filename string, err error) DetailedError {
	context := reader.document.Dump()
	highlightStart := uint(len(context)) - (reader.head - reader.mark)
	highlight := string(context[highlightStart:])
	contextTail := string(lastLines(context, 4))
	contextTail = strings.TrimSuffix(contextTail, highlight)
	contextLines := strings.Split(contextTail, "\n")

	return DetailedError{
		Filename:     filename,
		ContextLines: contextLines,
		FirstLine:    w3utils.SafeUint(int(reader.line) - len(contextLines)),
		Highlight:    highlight,
		Message:      err.Error(),
	}
}

func lastLines(runes []rune, maxLines int) []rune {
	if maxLines <= 0 {
		return nil
	}
	lines := 0
	for i, rune := range slices.Backward(runes) {
		if rune == '\n' {
			lines++
			if lines == maxLines {
				return runes[i+1:]
			}
		}
	}
	return runes
}

type DetailedError struct {
	Filename     string
	ContextLines []string
	FirstLine    uint
	Highlight    string
	Message      string
}

func (err DetailedError) Error() string {
	return fmt.Sprintf(
		"%s:%d: %s",
		err.Filename,
		err.FirstLine+uint(len(err.ContextLines)),
		err.Message,
	)
}

func (err DetailedError) Write(file *os.File) error {
	printer := libclog.NewPrinter(file, libclog.OutputColored)
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

func Write(err error, out *os.File) error {
	if derr, ok := err.(DetailedError); ok {
		return derr.Write(out)
	}
	_, e := out.WriteString(err.Error())
	return e
}
