package w3html

func (parser *DocumentParser) process(tag Tag) error {
	parser.page.AppendAndRemember(Hunk{
		Kind:     HunkTag,
		Line:     parser.line,
		Data:     tag,
		Children: nil,
	})
	return nil
}
