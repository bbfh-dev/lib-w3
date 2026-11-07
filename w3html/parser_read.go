package w3html

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

func (parser *Parser) readTag() (tag Tag, closing bool, err error) {
	mark := parser.reader.Here()
	if err := parser.readWhitespace(); err != nil {
		parser.reader.HighlightSince(mark)
		return Tag{}, false, fmt.Errorf("reading leading whitespace: %w", err)
	}

	mark = parser.reader.Here()
	closing, err = parser.readTagPrefix()
	if err != nil {
		parser.reader.HighlightSince(mark)
		return Tag{}, false, fmt.Errorf("reading tag prefix: %w", err)
	}

	// peek at what the first char is
	mark = parser.reader.Here()
	firstChar, err := parser.reader.Read()
	if err != nil {
		parser.reader.HighlightSince(mark)
		return Tag{}, false, err
	}
	parser.reader.Unread()

	var enum TagEnum
	var namespace, id string
	if firstChar != '.' {
		mark = parser.reader.Here()
		namespace, id, err = parser.readTagName()
		if err != nil {
			parser.reader.HighlightSince(mark)
			return Tag{}, false, fmt.Errorf("reading name: %w", err)
		}
		if id == "" {
			parser.reader.HighlightSince(mark)
			return Tag{}, false, errors.New("name cannot be empty")
		}
		var ok bool
		enum, ok = ResolveTag(namespace, id)
		if !ok {
			parser.reader.HighlightSince(mark)
			return Tag{}, false, fmt.Errorf(
				"undefined tag %q. Currently defined tags: [%s]",
				enum,
				RegisteredTags,
			)
		}
	}

	mark = parser.reader.Here()
	args, err := parser.readTagArgs()
	if err != nil {
		parser.reader.HighlightSince(mark)
		return Tag{}, false, fmt.Errorf("reading arguments: %w", err)
	}

	return Tag{Kind: enum, Arguments: args}, closing, nil
}

func (parser *Parser) readWhitespace() error {
	for {
		char, err := parser.reader.Read()
		if err != nil {
			return err
		}

		if unicode.IsSpace(char) {
			continue
		}

		parser.reader.Unread()
		return nil
	}
}

func (parser *Parser) readTagPrefix() (closing bool, err error) {
	char, err := parser.reader.Read()
	if err != nil {
		return false, err
	}
	if char == '/' {
		return true, nil
	}
	parser.reader.Unread()
	return false, nil
}

func (parser *Parser) readTagName() (namespace string, id string, err error) {
	var builder strings.Builder

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return "", "", err
		}

		if char == ' ' {
			return namespace, builder.String(), nil
		}

		if char == ':' {
			namespace = builder.String()
			builder.Reset()
			continue
		}

		if isNameCharset(char) {
			builder.WriteRune(char)
			continue
		}

		if char == '}' {
			parser.reader.Unread()
			return namespace, builder.String(), nil
		}
	}
}

func isNameCharset(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

func (parser *Parser) readTagArgs() ([]Argument, error) {
	return nil, nil
}
