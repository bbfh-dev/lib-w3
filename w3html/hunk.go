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
	HunkVariable
	HunkLoop
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
	case HunkVariable:
		return "HunkVariable"
	case HunkLoop:
		return "HunkLoop"
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

type InlineHunk struct {
	Kind HunkEnum
	Data any
}

// AsRaw casts the [Hunk.Data] value that is expected when [Hunk.Kind] == [HunkRaw].
func (hunk InlineHunk) AsRaw() []byte {
	return hunk.Data.([]byte)
}

// AsTag casts the [Hunk.Data] value that is expected when [Hunk.Kind] == [HunkTag].
func (hunk InlineHunk) AsTag() Tag {
	return hunk.Data.(Tag)
}

// AsVariable casts the [Hunk.Data] value that is expected when [Hunk.Kind] == [HunkVariable].
func (hunk InlineHunk) AsVariable() string {
	return hunk.Data.(string)
}

// AsLoop casts the [Hunk.Data] value that is expected when [Hunk.Kind] == [HunkLoop].
func (hunk InlineHunk) AsLoop() StringPath {
	return hunk.Data.(StringPath)
}

type Hunk struct {
	InlineHunk
	Line     uint
	Children []Hunk
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
	for _, child := range hunk.Children {
		child.Write(w, indent+1)
	}
}

func (hunk Hunk) InjectSlots() (map[string][]Hunk, error) {
	slots := map[string][]Hunk{}

	for _, child := range hunk.Children {
		switch child.Kind {
		case HunkTag:
			tag := child.AsTag()
			if tag.Name == "slot" {
				nameArg, err := tag.RequireNamedArg(ArgString, "name", 0)
				if err != nil {
					return slots, fmt.Errorf(
						"%w. Example: {{ slot 'body' }}. If your intent was to place into 'default' slot, simply remove '{{ slot }}' tag alltogether",
						err,
					)
				}
				name := nameArg.AsString()
				slots[name] = append(slots[name], child.Children...)
				continue
			}
			fallthrough
		case HunkRaw:
			if isWhitespace(child.AsRaw()) {
				continue
			}
			fallthrough
		default:
			slots["default"] = append(slots["default"], child)
		}
	}

	return slots, nil
}

// isWhitespace reports whether the buffer is purely whitespace.
func isWhitespace(buffer []byte) bool {
	for _, char := range buffer {
		switch char {
		case '\t', '\n', '\v', '\f', '\r', ' ':
		default:
			return false
		}
	}
	return true
}
