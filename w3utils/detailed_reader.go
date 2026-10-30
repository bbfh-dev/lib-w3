package w3utils

import (
	"bufio"
	"slices"
	"strings"

	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3errs"
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
func (reader *DetailedReader) WrapError(filename string, err error) w3errs.DetailedError {
	context := reader.document.Dump()
	highlightStart := uint(len(context)) - (reader.head - reader.mark)
	highlight := string(context[highlightStart:])
	contextTail := string(lastLines(context, 4))
	contextTail = strings.TrimSuffix(contextTail, highlight)
	contextLines := strings.Split(contextTail, "\n")

	return w3errs.DetailedError{
		Filename:     filename,
		ContextLines: contextLines,
		FirstLine:    SafeUint(int(reader.line) - len(contextLines)),
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
