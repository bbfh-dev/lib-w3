package w3html

func (parser *Parser) handleOpeningTag(tag Tag, mark uint) error {
	if tag.Kind.IsPaired {
		parser.appendTag(tag)
		parser.openingTags = append(parser.openingTags, NodeLocation{
			Parent:       parser.ctx.Pivot(),
			ChildIndex:   parser.ctx.lastAddedNodeIndex,
			Line:         parser.line,
			ReaderOffset: mark,
		})
		return nil
	}

	return parser.handleTag(tag)
}

func (parser *Parser) handleClosingTag(opening NodeLocation) error {
	hunk := opening.Pointer()
	tag := hunk.AsTag()
	return parser.handleTag(tag)
}

func (parser *Parser) handleTag(tag Tag) error {
	handler := tag.Kind.Handler
	if handler == nil {
		return nil
	}
	return handler(parser, tag)
}
