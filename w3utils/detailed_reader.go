package w3utils

import (
	"bufio"
	"os"
	"slices"
	"strings"

	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3errs"
)

type DetailedReader struct {
	parent   *DetailedReader
	filename string
	internal *bufio.Reader
	document *internal.RingBuffer[rune]
	Line     uint
	mark     uint
	head     uint
	rewinded uint
	cutLast  uint
}

func NewDetailedReader(parent *DetailedReader, file *os.File) *DetailedReader {
	return &DetailedReader{
		parent:   parent,
		filename: file.Name(),
		internal: bufio.NewReader(file),
		document: internal.NewRingBuffer[rune](),
		Line:     1,
		mark:     0,
		head:     0,
		rewinded: 0,
		cutLast:  0,
	}
}

func (reader *DetailedReader) Origin() *DetailedReader {
	if reader.parent != nil {
		return reader.parent.Origin()
	}
	return reader
}

func (reader *DetailedReader) FileName() string {
	return reader.filename
}

// Read the next rune.
func (reader *DetailedReader) Read() (rune, error) {
	char, _, err := reader.internal.ReadRune()
	reader.head++
	if char == '\n' {
		reader.Line++
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
func (reader *DetailedReader) MarkHere() uint {
	reader.mark = reader.head
	if reader.head != 0 {
		reader.mark--
	}
	return reader.mark
}

// MoveMarkTo moves the [DetailedReader.mark] to specified absolute byte offset.
func (reader *DetailedReader) MoveMarkTo(offset uint) {
	reader.cutLast = reader.mark - offset
	reader.mark = offset
}

func (reader *DetailedReader) callStack() w3errs.DocumentCallStack {
	location := w3errs.DocumentLocation{
		Filename: reader.filename,
		Line:     reader.Line,
	}
	if reader.parent == nil {
		return w3errs.DocumentCallStack{location}
	}
	return append(reader.parent.callStack(), location)
}

// WrapError creates a [DetailedError] with current state context.
// Use [DetailedError.Write] to pretty-print the error to file/console.
func (reader *DetailedReader) WrapError(filename string, err error) w3errs.DocumentError {
	context := reader.document.Dump()
	length := uint(len(context))
	highlightStart := length - (reader.head - reader.mark)
	highlight := string(context[highlightStart:])
	context = context[:length-reader.cutLast-2]
	contextTail := string(lastLines(context, 4))
	contextTail = strings.TrimSuffix(contextTail, highlight)
	contextLines := strings.Split(contextTail, "\n")
	return w3errs.DocumentError{
		CallStack:    reader.callStack(),
		ContextLines: contextLines,
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
