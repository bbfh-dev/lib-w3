package w3html

import (
	"errors"
	"os"
	"path/filepath"
	"unicode/utf8"
)

func (parser *Parser) flushHunkBuffer() {
	if len(parser.hunkBuffer) != 0 {
		parser.appendHunkBuffer()
	}
	parser.hunkBuffer = []byte{}
}

func (parser *Parser) appendHunkBuffer() {
	parser.AppendHunk(parser.hunkBuffer)
}

func (parser *Parser) AppendHunk(buffer []byte) {
	parser.Append(TreeNode{
		Kind:           NodeHunk,
		Data:           buffer,
		SourceCodeLine: parser.line,
		Children:       nil,
	})
}

func (parser *Parser) AppendPlaceholder() {
	parser.Append(TreeNode{
		Kind:           NodePlaceholder,
		Data:           nil,
		SourceCodeLine: parser.line,
		Children:       nil,
	})
}

func (parser *Parser) AppendTag(tag Tag) {
	parser.Append(TreeNode{
		Kind:           NodeTag,
		Data:           tag,
		SourceCodeLine: parser.line,
		Children:       nil,
	})
}

func (parser *Parser) Append(node TreeNode) {
	parser.ctx.Append(node)
}

func (parser *Parser) writeRune(char rune) {
	encoded := make([]byte, 4)
	n := utf8.EncodeRune(encoded, char)
	parser.hunkBuffer = append(parser.hunkBuffer, encoded[:n]...)
}

func (parser *Parser) findMatchingOpeningTag(closing TagEnum) (NodeLocation, bool) {
	if len(parser.ctx.breadcrumbs) == 0 {
		return NodeLocation{}, false
	}

	var visit func([]NodeLocation) (NodeLocation, bool)
	visit = func(openingTags []NodeLocation) (NodeLocation, bool) {
		last := len(openingTags) - 1
		if openingTags[last].Pointer().AsTag().Kind.Equal(closing) {
			return openingTags[last], true
		}

		if last == 0 {
			return NodeLocation{}, false
		}

		return visit(openingTags[:last])
	}

	return visit(parser.ctx.breadcrumbs)
}

func (parser *Parser) ParseNested(tag Tag) (ParsedPage, error) {
	pathArg, err := tag.FindArgNamed(ArgNamePath)
	if err != nil {
		return ParsedPage{}, err
	}
	path := pathArg.AsString()

	path = filepath.Join(filepath.Dir(parser.reader.Filename()), path)
	if path == parser.reader.Filename() {
		return ParsedPage{}, errors.New("trying to parse the same document recursively")
	}

	file, err := parser.fileOpener(path)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = parser.fileOpener(path + ".html")
			if err != nil {
				return ParsedPage{}, err
			}
		} else {
			return ParsedPage{}, err
		}
	}
	defer file.Close()

	nestedParser := newParser(parser.reader, file, tag.ToParams("path"), parser.PageDataType)
	nestedParser.fileOpener = parser.fileOpener

	scopeArg, err := tag.FindArgNamed(ArgNameScope)
	if err == nil {
		nestedParser.Scope = append(nestedParser.Scope, scopeArg.AsPath()...)
	}

	return nestedParser.Parse()
}
