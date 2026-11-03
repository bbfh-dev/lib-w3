package w3html

import "errors"

func (parser *DocumentParser) process(tag Tag) error {
	switch tag.Name {
	case "place":
		name := tag.FindNamedArg("name", 0)
		if name == "" {
			return errors.New(
				"'place' tag requires a 'name' argument. Please, explicitly specify 'default' if that was your intent. Example: {{ place 'default' }}.",
			)
		}
		parser.appendPlaceholder()
		pivot := parser.page.Pivot()
		parser.page.Slots[name] = SlotLocation{
			Parent: pivot,
			Index:  uint(len(pivot.Children) - 1),
		}
		return nil
	}

	parser.appendTag(tag)
	return nil
}
