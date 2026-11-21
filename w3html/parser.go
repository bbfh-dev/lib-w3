package w3html

import (
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3data"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

// FileOpener is called whenever [Parser] tries to open another file.
type FileOpener func(absPath string) (w3utils.FileLike, error)

var defaultFileOpener FileOpener = func(absPath string) (w3utils.FileLike, error) {
	return os.OpenFile(absPath, os.O_RDONLY, os.ModePerm)
}

// ParserApi is used to provide access to specific [Parser] methods for [TagEnum.Handler].
type ParserApi interface {
	ParseNested(tag Tag, slots SlotsMap) (ParsedPage, error)
	Param(name string) (string, error)
	SaveSlot(slotId string, location NodeLocation)
	LastAddedNode() NodeLocation
	AppendHunk(buffer []byte)
	AppendPlaceholder()
	AppendTag(tag Tag)
	Append(node Node)
	OpenFile(filename string) (w3utils.FileLike, error)
	StaticFS() fs.FS
}

type Parser struct {
	Scope        w3data.Path
	Params       map[string]string
	PageDataType reflect.Type
	reader       *internal.DocumentReader
	hunkBuffer   []byte
	line         uint
	ctx          *ParserContext
	fileOpener   FileOpener
	slots        SlotsMap
	placeholders uint
	staticFS     fs.FS
}

// NewParser creates a parser. For parsing nested documents refer to [Parser.NewParser].
func NewParser[T any](file w3utils.FileLike, params map[string]string, staticFS fs.FS) *Parser {
	return newParser(nil, file, params, staticFS, reflect.TypeOf(*new(T)))
}

func newParser(
	parent *internal.DocumentReader,
	file w3utils.FileLike,
	params map[string]string,
	staticFS fs.FS,
	data reflect.Type,
) *Parser {
	reader := internal.NewDocumentReader(parent, file)
	params["_filename"] = file.Name()
	params["_filename.base"] = filepath.Base(file.Name())
	return &Parser{
		Scope:        w3data.Path{},
		Params:       params,
		PageDataType: data,
		reader:       reader,
		hunkBuffer:   []byte{},
		line:         1,
		ctx: &ParserContext{
			Root: Node{
				Kind:           NodeRoot,
				Data:           nil,
				SourceCodeLine: 1,
				Children:       []Node{},
			},
			out: ParsedPage{
				PlacedSlots: map[string]NodeLocation{},
				Nodes:       nil,
			},
			breadcrumbs:        []NodeLocation{},
			lastAddedNode:      nil,
			lastAddedNodeIndex: 0,
		},
		fileOpener: defaultFileOpener,
		slots:      SlotsMap{},
		staticFS:   staticFS,
	}
}

// WithFileOpener replaces the default [FileOpener].
// Refer to [FileOpener] documentation.
func (parser *Parser) WithFileOpener(fileOpener FileOpener) *Parser {
	parser.fileOpener = fileOpener
	return parser
}

func (parser *Parser) Param(path string) (string, error) {
	value, ok := parser.Params[path]
	if !ok {
		return value, fmt.Errorf(
			"unknown compile-time variable %q. Make sure you didn't mess up capitalization & the variable is set in the 'include' or 'extend' tag calling this file. Available parameters are: [%s]",
			path,
			strings.Join(slices.Collect(maps.Keys(parser.Params)), ", "),
		)
	}
	return value, nil
}

func (parser *Parser) SaveSlot(slotId string, location NodeLocation) {
	parser.ctx.out.PlacedSlots[slotId] = location
}

func (parser *Parser) LastAddedNode() NodeLocation {
	return NodeLocation{
		Parent:       parser.ctx.Pivot(),
		ChildIndex:   parser.ctx.lastAddedNodeIndex,
		Line:         parser.ctx.lastAddedNode.SourceCodeLine,
		ReaderOffset: parser.reader.Here(),
	}
}

func (parser *Parser) Parse() (ParsedPage, error) {
	err := parser.parse()
	if err != nil {
		return ParsedPage{}, parser.reader.WrapError(err)
	}
	if len(parser.ctx.breadcrumbs) != 0 {
		return ParsedPage{}, parser.reader.WrapError(fmt.Errorf(
			"paired tag %q was never closed",
			parser.ctx.breadcrumbs[len(parser.ctx.breadcrumbs)-1].Pointer().AsTag().Kind,
		))
	}
	return parser.ToParsedPage(), nil
}

type swallowSpaceEnum uint8

const (
	swallowNone swallowSpaceEnum = iota
	swallowIndent
	swallowWhitespace
)

func (parser *Parser) parse() error {
	swallowMode := swallowWhitespace
	for {
		char, err := parser.reader.Read()
		if err != nil {
			if err == io.EOF {
				parser.flushHunkBuffer()
				return nil
			}
			return err
		}

		switch swallowMode {

		case swallowWhitespace:
			if unicode.IsSpace(char) {
				continue
			}
			swallowMode = swallowNone

		case swallowIndent:
			switch char {
			case '\t', '\v', '\f', ' ', 0x85, 0xA0:
				continue
			default:
				swallowMode = swallowNone
			}
		}

		if char != '{' {
			// not a tag
			parser.writeRune(char)
			if char == '\n' {
				swallowMode = swallowWhitespace
			}
			continue
		}

		// possibly a tag? look for second '{'
		tagBeginning := parser.reader.Here()
		char, err = parser.reader.Read()
		if err != nil {
			if err == io.EOF {
				parser.writeRune('{')
				parser.flushHunkBuffer()
				return nil
			}
			return err
		}

		if char != '{' {
			// it wasn't a tag
			parser.writeRune('{')
			parser.writeRune(char)
			continue
		}

		// at this point it's expected to be a tag
		parser.flushHunkBuffer()
		parser.line = parser.reader.Line

		tag, closing, err := parser.readTag()
		if err != nil {
			return fmt.Errorf("reading tag: %w", err)
		}
		swallowMode = swallowIndent

		if closing {
			if !tag.Kind.IsPaired {
				parser.reader.HighlightSince(tagBeginning)
				parser.reader.HideArrow()
				return fmt.Errorf("%q tag is not a paired tag", tag.Kind)
			}

			openingTag, found := parser.findMatchingOpeningTag(tag.Kind)
			if !found {
				return fmt.Errorf(
					"trying to close tag %q, but couldn't locate the opening pair",
					tag.Kind,
				)
			}

			err := parser.ctx.OnLeaveTagBody(tag)
			if err != nil {
				parser.reader.HighlightSince(tagBeginning)
				parser.reader.HideArrow()
				return err
			}

			restoreLine := parser.reader.Line
			parser.reader.Line = openingTag.Line
			err = parser.handleClosingTag(openingTag)
			if err != nil {
				parser.reader.HighlightSince(tagBeginning)
				parser.reader.HideArrow()
				if _, ok := err.(w3errs.DocumentError); ok {
					return err
				}
				return fmt.Errorf("handling tag closing: %w", err)
			}
			parser.reader.Line = restoreLine
		} else {
			err = parser.handleOpeningTag(tag)
			if err != nil {
				parser.reader.HighlightSince(tagBeginning)
				parser.reader.HideArrow()
				if _, ok := err.(w3errs.DocumentError); ok {
					return err
				}
				return fmt.Errorf("handling opening tag: %w", err)
			}
			if tag.Kind.IsPaired {
				parser.ctx.OnEnterTagBody(tagBeginning)
			}
		}
	}
}

// ————————————————————————————————

type ParsedPage struct {
	PlacedSlots map[string]NodeLocation
	Nodes       []Node
}

func (parser *Parser) ToParsedPage() ParsedPage {
	var visit func(*Node)
	visit = func(node *Node) {
		keepChildren := node.Kind == NodeTag && node.AsTag().Kind.KeepChildren
		if node.Kind == NodePlaceholder {
			id := node.Data.(uint)
			for key, location := range parser.ctx.out.PlacedSlots {
				if location.Pointer().Data.(uint) == id {
					for _, node := range parser.slots[key] {
						keepChildren = node.Kind == NodeTag && node.AsTag().Kind.KeepChildren
						if !keepChildren {
							node.Children = nil
						}
						parser.ctx.out.Nodes = append(parser.ctx.out.Nodes, node)
					}
				}
			}
		}

		if !node.IsPseudo() {
			lastIdx := len(parser.ctx.out.Nodes) - 1
			if lastIdx >= 0 && node.Kind == NodeHunk &&
				parser.ctx.out.Nodes[lastIdx].Kind == NodeHunk {
				parser.ctx.out.Nodes[lastIdx].Data = append(
					parser.ctx.out.Nodes[lastIdx].Data.([]byte),
					node.Data.([]byte)...,
				)
			} else {
				if !keepChildren {
					node.Children = nil
				}
				parser.ctx.out.Nodes = append(parser.ctx.out.Nodes, *node)
			}
		}

		if !keepChildren {
			for _, child := range node.Children {
				visit(&child)
			}
		}
	}

	for _, child := range parser.ctx.Root.Children {
		visit(&child)
	}

	return parser.ctx.out
}
