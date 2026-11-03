package w3html

func (parser *DocumentParser) process(tag Tag) error {
	parser.appendTag(tag)
	return nil
}
