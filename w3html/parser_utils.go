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
	parser.appendHunk(parser.hunkBuffer)
}

func (parser *Parser) appendHunk(buffer []byte) {
	parser.ctx.Append(TreeNode{
		Kind:           NodeHunk,
		Data:           buffer,
		SourceCodeLine: parser.line,
		Children:       nil,
	})
}

func (parser *Parser) appendPlaceholder() {
	parser.ctx.Append(TreeNode{
		Kind:           NodePlaceholder,
		Data:           nil,
		SourceCodeLine: parser.line,
		Children:       nil,
	})
}

func (parser *Parser) appendTag(tag Tag) {
	parser.ctx.Append(TreeNode{
		Kind:           NodeTag,
		Data:           tag,
		SourceCodeLine: parser.line,
		Children:       nil,
	})
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

	nestedParser := newParser(parser.reader, file, tag.ToParams("path"))
	result, err := nestedParser.Parse()
	if err != nil {
		return ParsedPage{}, err
	}
	parser.ctx.out.AccessPaths = append(parser.ctx.out.AccessPaths, result.AccessPaths...)
	return result, nil
}
