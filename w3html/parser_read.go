package w3html

import (
	"errors"
	"fmt"
	"strconv"
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

	args, err := parser.readTagArgs()
	if err != nil {
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
	var args = []Argument{}
	var key string
	var mark uint

	for {
		mark = parser.reader.Here()
		firstChar, err := parser.reader.Read()
		if err != nil {
			parser.reader.HighlightSince(mark)
			return args, err
		}

		if key == "" && isArgKeyCharset(firstChar) {
			key, err = parser.readTagArgKey(firstChar)
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading argument key: %w", err)
			}
			continue
		}

		if isDigit(firstChar) || firstChar == '-' {
			arg, err := parser.readNumberArg(firstChar)
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading number: %w", err)
			}
			arg.Key = key
			args = append(args, arg)
			key = ""
			continue
		}

		switch firstChar {

		case '}':
			char, err := parser.reader.Read()
			if err != nil {
				return args, fmt.Errorf("reading tag closing brace: %w", err)
			}
			if char != '}' {
				return args, fmt.Errorf("expected '}', but got %q", char)
			}
			return args, nil

		case '\'', '"':
			contents, err := parser.readQuoted(firstChar)
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading quoted string: %q", err)
			}
			args = append(args, Argument{
				Kind: ArgString,
				Key:  key,
				Data: contents,
			})
			key = ""

		case 't':
			_, err := parser.readKeyword('t', 'r', 'u', 'e')
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading keyword: %q", err)
			}
			args = append(args, Argument{
				Kind: ArgBool,
				Key:  key,
				Data: true,
			})

		case 'f':
			_, err := parser.readKeyword('f', 'a', 'l', 's', 'e')
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading keyword: %q", err)
			}
			args = append(args, Argument{
				Kind: ArgBool,
				Key:  key,
				Data: false,
			})
		}
	}
}

func (parser *Parser) readTagArgKey(firstChar rune) (string, error) {
	var buffer = []rune{firstChar}

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return "", err
		}

		if char == '=' {
			return string(buffer), nil
		}

		if isArgKeyCharset(char) {
			buffer = append(buffer, char)
			continue
		}

		return "", fmt.Errorf("unexpected character %q", char)
	}
}

func (parser *Parser) readQuoted(quote rune) (string, error) {
	var buffer = []rune{}

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return "", err
		}

		switch char {
		case quote:
			return string(buffer), nil
		case '\n':
			return string(buffer), fmt.Errorf("unexpected new line inside of a quoted string")
		default:
			buffer = append(buffer, char)
		}
	}
}

func (parser *Parser) readNumberArg(firstChar rune) (Argument, error) {
	var buffer = []rune{firstChar}
	var isFloat bool

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return Argument{}, err
		}

		if isDigit(char) {
			buffer = append(buffer, char)
			continue
		}

		if char == '.' && !isFloat {
			isFloat = true
			continue
		}

		if char == ' ' || char == '}' {
			parser.reader.Unread()
			if isFloat {
				value, err := strconv.ParseFloat(string(buffer), 64)
				if err != nil {
					return Argument{}, fmt.Errorf("internal error: %w", err)
				}
				return Argument{Kind: ArgFloat, Data: value}, nil
			}

			value, err := strconv.Atoi(string(buffer))
			if err != nil {
				return Argument{}, fmt.Errorf("internal error: %w", err)
			}
			return Argument{Kind: ArgInt, Data: value}, nil
		}

		return Argument{}, fmt.Errorf("unexpected character: %q", char)
	}
}

func (parser *Parser) readKeyword(firstChar rune, chars ...rune) (string, error) {
	var buffer = []rune{firstChar}
	for _, expected := range chars {
		char, err := parser.reader.Read()
		if err != nil {
			return "", err
		}
		if char != expected {
			return "", fmt.Errorf(
				"expected %q, but got '%s...'",
				string(append([]rune{firstChar}, chars...)),
				string(buffer),
			)
		}
		buffer = append(buffer, char)
	}
	return string(buffer), nil
}

func isArgKeyCharset(char rune) bool {
	return (char >= 'a' && char <= 'z') || char == '_'
}

func isDigit(char rune) bool {
	return (char >= '0' && char <= '9')
}
