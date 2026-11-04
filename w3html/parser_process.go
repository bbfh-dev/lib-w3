package w3html

import (
	"errors"
	"fmt"
	"strings"
)

func (parser *DocumentParser) processOpening(tag Tag, mark uint) error {
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
		parser.page.Slots[name] = HunkLocation{
			Parent:     parser.page.Pivot(),
			ChildIndex: uint(parser.page.lastIndex),
		}
		return nil

	case "include":
		nestedPage, err := parser.parseNestedDocument(tag)
		if err != nil {
			return err
		}
		for _, child := range nestedPage.Root.Children {
			parser.page.Append(child)
		}
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
	if tag.IsPaired() {
		parser.openingTags = append(parser.openingTags, HunkLocation{
			Parent:     parser.page.Pivot(),
			ChildIndex: parser.page.lastIndex,
			Line:       parser.line,
			ByteOffset: mark,
		})
	}
	return nil
}

func (parser *DocumentParser) processClosing(opening HunkLocation) error {
	hunk := opening.Hunk()
	tag := hunk.AsTag()

	switch tag.Name {
	case "extend":
		nestedPage, err := parser.parseNestedDocument(tag)
		if err != nil {
			return err
		}

		injectSlots, err := hunk.InjectSlots()
		if err != nil {
			return err
		}
		for slotId, injectHunks := range injectSlots {
			location, ok := nestedPage.Slots[slotId]
			if !ok {
				return fmt.Errorf(
					"trying to inject into slot %q, but it was never defined in the extended document",
					slotId,
				)
			}
			location.Parent.Children[location.ChildIndex] = Hunk{
				Kind:     HunkRoot,
				Line:     location.Parent.Children[location.ChildIndex].Line,
				Data:     nil,
				Children: injectHunks,
			}
		}
		opening.Parent.Children[opening.ChildIndex] = nestedPage.Root
	}

	return nil
}

func isCompileTimeVariable(path string) bool {
	return path == strings.ToLower(path)
}
