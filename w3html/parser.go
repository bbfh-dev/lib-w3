package w3html

import (
	"fmt"
	"io"
	"path/filepath"
	"unicode"

	"codeberg.org/bbfh/lib-w3/internal"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type Parser struct {
	Params     map[string]string
	file       w3utils.FileLike
	reader     *internal.DocumentReader
	hunkBuffer []byte
	line       uint
	ctx        *ParserContext
}

// NewParser creates a parser. For parsing nested documents refer to [Parser.NewParser].
func NewParser(file w3utils.FileLike, params map[string]string) *Parser {
	return newParser(nil, file, params)
}

// NewParser creates a new nested parser.
func (parser *Parser) NewParser(file w3utils.FileLike, params map[string]string) *Parser {
	return newParser(parser.reader, file, params)
}

func newParser(
	parent *internal.DocumentReader,
	file w3utils.FileLike,
	params map[string]string,
) *Parser {
	reader := internal.NewDocumentReader(parent, file)
	params["_filename"] = file.Name()
	params["_filename.base"] = filepath.Base(file.Name())
	return &Parser{
		Params:     params,
		file:       file,
		reader:     reader,
		hunkBuffer: []byte{},
		line:       1,
		ctx: &ParserContext{
			AccessPaths: []AccessPath{},
			PlacedSlots: map[string]NodeLocation{},
			Root: TreeNode{
				Node: Node{
					Kind: NodeRoot,
					Data: nil,
				},
				SourceCodeLine: 1,
				Children:       []TreeNode{},
			},
			breadcrumbs:        []NodeLocation{},
			lastAddedNode:      nil,
			lastAddedNodeIndex: 0,
		},
	}
}

func (parser *Parser) Parse() (ParsedPage, error) {
	err := parser.parse()
	if err != nil {
		return ParsedPage{}, parser.reader.WrapError(err)
	}
	return parser.ToParsedPage(), nil
}

func (parser *Parser) parse() error {
	swallowSpace := true
	for {
		char, err := parser.reader.Read()
		if err != nil {
			if err == io.EOF {
				parser.flushHunkBuffer()
				return nil
			}
			return err
		}

		if swallowSpace {
			if unicode.IsSpace(char) {
				continue
			}
			swallowSpace = false
		}

		if char != '{' {
			// not a tag
			parser.writeRune(char)
			if char == '\n' {
				swallowSpace = true
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
		swallowSpace = true

		if closing {
			if !tag.Kind.IsPaired {
				parser.reader.HighlightSince(tagBeginning)
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
				return err
			}

			restoreLine := parser.reader.Line
			parser.reader.Line = openingTag.Line
			err = parser.handleClosingTag(openingTag)
			if err != nil {
				parser.reader.HighlightSince(tagBeginning)
				if _, ok := err.(w3errs.DocumentError); ok {
					return err
				}
				return fmt.Errorf("handling tag closing: %w", err)
			}
			parser.reader.Line = restoreLine
			continue
		}

		err = parser.handleOpeningTag(tag)
		if err != nil {
			parser.reader.HighlightSince(tagBeginning)
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

// ————————————————————————————————

type ParsedPage struct {
	AccessPaths []AccessPath
	PlacedSlots map[string]NodeLocation
	Nodes       []Node
}

func (parser *Parser) ToParsedPage() ParsedPage {
	nodes := []Node{}

	var visit func(TreeNode)
	visit = func(treeNode TreeNode) {
		nodes = append(nodes, treeNode.Node)
		for _, child := range treeNode.Children {
			visit(child)
		}
	}
	for _, child := range parser.ctx.Root.Children {
		visit(child)
	}

	return ParsedPage{
		AccessPaths: parser.ctx.AccessPaths,
		PlacedSlots: parser.ctx.PlacedSlots,
		Nodes:       nodes,
	}
}
