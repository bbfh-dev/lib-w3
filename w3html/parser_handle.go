package w3html

func (parser *Parser) handleOpeningTag(tag Tag) error {
	for i, definedArg := range tag.Kind.Arguments {
		if definedArg.IsOptional {
			break
		}
		_, err := tag.FindArgStrict(uint(i))
		if err != nil {
			return err
		}
	}

	if tag.Kind.IsPaired {
		parser.AppendTag(tag)
		return nil
	}

	return parser.handleTag(nil, tag)
}

func (parser *Parser) handleClosingTag(opening NodeLocation) error {
	node := opening.Pointer()
	tag := node.AsTag()
	return parser.handleTag(node, tag)
}

func (parser *Parser) handleTag(node *TreeNode, tag Tag) error {
	handler := tag.Kind.Handler
	if handler == nil {
		return nil
	}
	return handler(parser, node, tag)
}
