package w3html

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"codeberg.org/bbfh/lib-w3/w3data"
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
		var ok bool
		enum, ok = ResolveTag(namespace, id)
		if !ok {
			parser.reader.HighlightSince(mark)
			return Tag{}, false, fmt.Errorf(
				"undefined tag %q. Currently defined tags: %s",
				enum,
				RegisteredTags,
			)
		}
	} else {
		enum = IdVariable
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

		return "", "", fmt.Errorf("unexpected character: %q", char)
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

		case ' ':
			// skip

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
				return args, fmt.Errorf("reading quoted string: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgString,
				Key:  key,
				Data: contents,
			})
			key = ""

		case '`':
			formatString, err := parser.readFormatString()
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading format string: %w", err)
			}
			if len(formatString) == 0 {
				args = append(args, Argument{
					Kind: ArgString,
					Key:  key,
					Data: "",
				})
			} else if len(formatString) == 1 && formatString[0].Kind == FragmentString {
				args = append(args, Argument{
					Kind: ArgString,
					Key:  key,
					Data: formatString[0].Data,
				})
			} else {
				args = append(args, Argument{
					Kind: ArgFormatString,
					Key:  key,
					Data: formatString,
				})
			}
			key = ""

		case 't':
			_, err := parser.readKeyword('t', 'r', 'u', 'e')
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading keyword: %w", err)
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
				return args, fmt.Errorf("reading keyword: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgBool,
				Key:  key,
				Data: false,
			})

		case '&':
			_, err := parser.readKeyword('&', '&')
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading keyword: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgOperation,
				Key:  key,
				Data: OpLogicAnd,
			})

		case '|':
			_, err := parser.readKeyword('|', '|')
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading keyword: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgOperation,
				Key:  key,
				Data: OpLogicOr,
			})

		case '!':
			_, err := parser.readKeyword('!', '=')
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading keyword: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgOperation,
				Key:  key,
				Data: OpNotEquals,
			})

		case '=':
			_, err := parser.readKeyword('=', '=')
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading keyword: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgOperation,
				Key:  key,
				Data: OpEquals,
			})

		case '>':
			nextChar, err := parser.reader.Read()
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading inequality sign: %w", err)
			}
			if nextChar != '=' {
				parser.reader.Unread()
				args = append(args, Argument{
					Kind: ArgOperation,
					Key:  key,
					Data: OpMoreThanStrict,
				})
				continue
			}
			args = append(args, Argument{
				Kind: ArgOperation,
				Key:  key,
				Data: OpMoreThan,
			})

		case '<':
			nextChar, err := parser.reader.Read()
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading inequality sign: %w", err)
			}
			if nextChar != '=' {
				parser.reader.Unread()
				args = append(args, Argument{
					Kind: ArgOperation,
					Key:  key,
					Data: OpLessThanStrict,
				})
				continue
			}
			args = append(args, Argument{
				Kind: ArgOperation,
				Key:  key,
				Data: OpLessThan,
			})

		case '.':
			path, err := parser.readPath()
			if err != nil {
				parser.reader.HighlightSince(mark)
				return args, fmt.Errorf("reading access path: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgPath,
				Key:  key,
				Data: path,
			})
			key = ""
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

func (parser *Parser) readFormatString() (FormatString, error) {
	out := FormatString{}
	var buffer = []rune{}

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return nil, err
		}

		switch char {

		case '`':
			if len(buffer) != 0 {
				if len(out) != 0 && out[0].Kind == FragmentString {
					out[len(out)-1].Data = out[len(out)-1].Data.(string) + string(buffer)
				} else {
					out = append(out, formatFragment{
						Kind: FragmentString,
						Data: string(buffer),
					})
				}
			}
			return out, nil

		case '{':
			path, err := parser.readPath()
			if err != nil {
				return nil, fmt.Errorf("reading path: %w", err)
			}
			nextChar, err := parser.reader.Read()
			if err != nil {
				return nil, err
			}
			if nextChar != '}' {
				return nil, fmt.Errorf("expected '}' after path, but got %q", nextChar)
			}
			if path.IsCompileTime() {
				value, err := parser.Param(path.Flatten())
				if err != nil {
					return nil, err
				}
				if len(out) != 0 && out[0].Kind == FragmentString {
					out[len(out)-1].Data = out[len(out)-1].Data.(string) + value
				} else {
					out = append(out, formatFragment{
						Kind: FragmentString,
						Data: value,
					})
				}
			} else {
				out = append(out, formatFragment{
					Kind: FragmentPath,
					Data: path,
				})
			}

		case '\n':
			return nil, fmt.Errorf("unexpected new line inside of a format string")

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

func (parser *Parser) readPath() (w3data.Path, error) {
	path := w3data.Path{}
	buffer := []rune{}
	segment := w3data.Segment{Call: false, Iterate: false}
	dataType := parser.PageDataType

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return path, err
		}

		if isAccessPathCharset(char) {
			if segment.Call || segment.Iterate {
				return path, fmt.Errorf(
					"path segment already ended with '()' or '[]', but found %q right after",
					char,
				)
			}
			buffer = append(buffer, char)
			continue
		}

		switch char {

		case ' ', '}':
			parser.reader.Unread()
			if len(buffer) != 0 {
				segment.Word = string(buffer)
				segments, err := w3data.GetPath(dataType, segment)
				if err != nil {
					return path, err
				}
				return append(path, segments...), nil
			}
			return path, nil

		case '(':
			if segment.Iterate {
				return path, fmt.Errorf("unexpected %q after '[]'", char)
			}
			_, err := parser.readKeyword('(', ')')
			if err != nil {
				return path, fmt.Errorf("reading keyword: %w", err)
			}
			segment.Call = true

		case '[':
			_, err := parser.readKeyword('[', ']')
			if err != nil {
				return path, fmt.Errorf("reading keyword: %w", err)
			}
			segment.Iterate = true

		case '.':
			if len(buffer) == 0 {
				if len(path) == 0 {
					path = append(path, parser.Scope...)
					path = append(path, parser.ctx.Scope()...)
					dataType = path[len(path)-1].DataType()
					continue
				}
				return path, fmt.Errorf("unexpected '.' in the middle of the path")
			}
			segment.Word = string(buffer)
			segments, err := w3data.GetPath(dataType, segment)
			if err != nil {
				return path, err
			}
			dataType = segments[len(segments)-1].DataType()
			path = append(path, segments...)
			buffer = []rune{}
			segment.Call = false
			segment.Iterate = false

		default:
			return path, fmt.Errorf("unexpected character %q", char)
		}
	}
}

func isArgKeyCharset(char rune) bool {
	return (char >= 'a' && char <= 'z') || char == '_'
}

func isAccessPathCharset(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

func isDigit(char rune) bool {
	return (char >= '0' && char <= '9')
}
