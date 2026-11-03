package w3html

import (
	"fmt"
	"io"
	"strings"

	libclog "codeberg.org/bbfh/lib-clog"
)

// HunkEnum represents the datatype of [Hunk.Data].
type HunkEnum uint8

const (
	HunkRoot HunkEnum = iota
	HunkPlaceholder
	HunkRaw
	HunkTag
)

func (enum HunkEnum) String() string {
	switch enum {
	case HunkRoot:
		return "HunkRoot"
	case HunkPlaceholder:
		return "HunkPlaceholder"
	case HunkRaw:
		return "HunkRaw"
	case HunkTag:
		return "HunkTag"
	default:
		return fmt.Sprintf("HunkEnum(%d)", enum)
	}
}

// IsPseudo returns whether the [HunkEnum] is a pseudo hunk.
//
// A pseudo hunk doesn't increment indentation and doesn't have any contents.
func (enum HunkEnum) IsPseudo() bool {
	switch enum {
	case HunkRoot, HunkPlaceholder:
		return true
	default:
		return false
	}
}

// ————————————————————————————————

// Hunk represents a snippet of the document.
type Hunk struct {
	Kind     HunkEnum
	Line     uint
	Data     any
	Children []Hunk
}

// AsRaw casts the [Hunk.Data] value that is expected when [Hunk.Kind] == [HunkRaw].
func (hunk Hunk) AsRaw() []byte {
	return hunk.Data.([]byte)
}

// AsTag casts the [Hunk.Data] value that is expected when [Hunk.Kind] == [HunkTag].
func (hunk Hunk) AsTag() Tag {
	return hunk.Data.(Tag)
}

func (hunk Hunk) String() string {
	var builder strings.Builder
	hunk.Write(&builder, 0)
	return builder.String()
}

func (hunk Hunk) Write(w io.Writer, indent uint) {
	indentString := strings.Repeat("    ", int(indent))
	data := hunk.Data
	if data == nil {
		data = "(empty)"
	}
	fmt.Fprintf(
		w,
		"%s— %s —%s\n%[1]s%[4]s%[5]s%[6]s\n",
		indentString,
		hunk.Kind,
		strings.Repeat("—", 30),
		libclog.AnsiColor(libclog.ColorBrightBlack),
		data,
		libclog.AnsiReset,
	)
	if !hunk.Kind.IsPseudo() {
		indent++
	}
	for _, child := range hunk.Children {
		child.Write(w, indent)
	}
}
