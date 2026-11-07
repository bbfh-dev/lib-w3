package w3html

import (
	"fmt"
	"io"
	"strings"

	libclog "codeberg.org/bbfh/lib-clog"
)

var NoNodeLocation = NodeLocation{Parent: nil}

type NodeLocation struct {
	Parent       *TreeNode
	ChildIndex   uint
	Line         uint
	ReaderOffset uint
}

func (location NodeLocation) Pointer() *TreeNode {
	return &location.Parent.Children[location.ChildIndex]
}

// ————————————————————————————————

// read-only
type NodeEnum uint8

const (
	NodeRoot NodeEnum = iota
	NodePlaceholder
	NodeHunk
	NodeTag
)

func (enum NodeEnum) String() string {
	switch enum {
	case NodeRoot:
		return "NodeRoot"
	case NodePlaceholder:
		return "NodePlaceholder"
	case NodeHunk:
		return "NodeRaw"
	case NodeTag:
		return "NodeTag"
	default:
		return fmt.Sprintf("NodeEnum(%d)", enum)
	}
}

type Node struct {
	Kind NodeEnum
	Data any
}

func (node Node) Write(w io.Writer) {
	data := node.Data
	if data == nil {
		data = "(empty)"
	}
	fmt.Fprintf(
		w,
		"— %s —%s\n%s%s%s\n",
		node.Kind,
		strings.Repeat("—", 30),
		libclog.AnsiColor(libclog.ColorBrightBlack),
		data,
		libclog.AnsiReset,
	)
}

// AsRaw casts the [Node.Data] value that is expected when [Node.Kind] == [NodeHunk].
func (node Node) AsHunk() []byte {
	return node.Data.([]byte)
}

// AsTag casts the [Node.Data] value that is expected when [Node.Kind] == [NodeTag].
func (node Node) AsTag() Tag {
	return node.Data.(Tag)
}

type TreeNode struct {
	Node
	SourceCodeLine uint
	Children       []TreeNode
}

func (node TreeNode) Write(w io.Writer, indent uint) {
	indentString := strings.Repeat("    ", int(indent))
	data := node.Data
	if data == nil {
		data = "(empty)"
	}
	fmt.Fprintf(
		w,
		"%s— %s —%s\n%[1]s%[4]s%[5]s%[6]s\n",
		indentString,
		node.Kind,
		strings.Repeat("—", 30),
		libclog.AnsiColor(libclog.ColorBrightBlack),
		data,
		libclog.AnsiReset,
	)
	for _, child := range node.Children {
		child.Write(w, indent+1)
	}
}

func (hunk TreeNode) ExtractInjectSlots() (map[string][]TreeNode, error) {
	slots := map[string][]TreeNode{}
	for _, child := range hunk.Children {
		switch child.Kind {
		case NodeTag:
			tag := child.AsTag()
			if tag.Kind.Equal(IdSlot) {
				nameArg, err := tag.FindArgStrict(ArgString, "name", 0)
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
		case NodeHunk:
			if isWhitespace(child.AsHunk()) {
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
