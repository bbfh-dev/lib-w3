package internal

import (
	"bufio"
	"os"
	"slices"
	"strings"

	"codeberg.org/bbfh/lib-w3/w3errs"
)

type DocumentReader struct {
	Line           uint
	parent         *DocumentReader
	filename       string
	internal       *bufio.Reader
	document       *RingBuffer1024[rune]
	head           uint
	rewinded       uint
	highlightStart uint
	showArrow      bool
}

func NewDocumentReader(parent *DocumentReader, file *os.File) *DocumentReader {
	return &DocumentReader{
		Line:           1,
		parent:         parent,
		filename:       file.Name(),
		internal:       bufio.NewReader(file),
		document:       NewRingBuffer1024[rune](),
		head:           0,
		rewinded:       0,
		highlightStart: 0,
		showArrow:      true,
	}
}

func (reader *DocumentReader) Origin() *DocumentReader {
	if reader.parent == nil {
		return reader
	}
	return reader.parent.Origin()
}

func (reader *DocumentReader) callStack() w3errs.DocumentCallStack {
	location := w3errs.DocumentLocation{
		Filename: reader.filename,
		Line:     reader.Line,
	}
	if reader.parent == nil {
		return w3errs.DocumentCallStack{location}
	}
	return append(reader.parent.callStack(), location)
}

func (reader *DocumentReader) Filename() string {
	return reader.filename
}

func (reader *DocumentReader) Read() (rune, error) {
	char, _, err := reader.internal.ReadRune()
	if err != nil {
		return '\x00', err
	}
	reader.head++
	if char == '\n' {
		reader.Line++
	}
	if reader.rewinded == 0 {
		reader.document.Push(char)
	} else {
		reader.rewinded--
	}
	return char, nil
}

func (reader *DocumentReader) Unread() {
	err := reader.internal.UnreadRune()
	if err != nil {
		panic(err)
	}
	reader.head--
	reader.rewinded++
}

func (reader *DocumentReader) Here() uint {
	return reader.head
}

func (reader *DocumentReader) HighlightSince(head uint) {
	reader.highlightStart = head
}

func (reader *DocumentReader) HideArrow() {
	reader.showArrow = false
}

func (reader *DocumentReader) WrapError(err error) w3errs.DocumentError {
	context := reader.document.Dump()
	length := uint(len(context))

	highlightStart := length - (reader.head - reader.highlightStart)
	highlight := string(context[highlightStart:])

	context = context[:length]
	contextTail := string(lastLines(context, 4))
	contextTail = strings.TrimSuffix(contextTail, highlight)
	contextLines := strings.Split(contextTail, "\n")

	return w3errs.DocumentError{
		CallStack:    reader.callStack(),
		ContextLines: contextLines,
		Highlight:    highlight,
		Message:      err.Error(),
		DrawArrow:    reader.showArrow,
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
