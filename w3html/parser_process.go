package w3html

import (
	"errors"
	"fmt"
	"strings"
)

func (parser *DocumentParser) process(tag Tag) error {
	if tag.IsPath() {
		path := tag.Args[0].Value[1:]
		if isCompileTimeVariable(path) {
			value, ok := parser.Params[path]
			if !ok {
				return fmt.Errorf(
					"unknown compile-time variable %q. Make sure you didn't mess up capitalization & the variable is set in the 'include' or 'extend' tag calling this file",
					path,
				)
			}
			parser.appendRaw([]byte(value))
		} else {
			parser.appendVariable(path)
		}
		return nil
	}

	switch tag.Name {

	case "place":
		name := tag.FindNamedArg("name", 0)
		if name == "" {
			return errors.New(
				"'place' tag requires a 'name' argument. Please, explicitly specify 'default' if that was your intent. Example: {{ place 'default' }}",
			)
		}
		parser.appendPlaceholder()
		pivot := parser.page.Pivot()
		parser.page.Slots[name] = SlotLocation{
			Parent: pivot,
			Index:  uint(len(pivot.Children) - 1),
		}
		return nil

	case "include":
		nestedPage, err := parser.parseNestedDocument(tag)
		if err != nil {
			return err
		}
		parser.page.Append(nestedPage.Root)
		return nil

	case "":
		if len(tag.Args) == 0 {
			return nil
		}
		if len(tag.Args) == 1 {
			return fmt.Errorf(
				"unnamed tag must have a single path argument, but %q is not a valid path",
				tag.Args[0].Value,
			)
		}
		return fmt.Errorf("unnamed tag must have a single path argument, but got %d", len(tag.Args))
	}

	parser.appendTag(tag)
	return nil
}

func isCompileTimeVariable(path string) bool {
	return path == strings.ToLower(path)
}
