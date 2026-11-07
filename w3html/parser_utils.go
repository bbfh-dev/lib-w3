package w3html

import "unicode/utf8"

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
	if len(parser.openingTags) == 0 {
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

	return visit(parser.openingTags)
}
