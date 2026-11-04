package w3html

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func (parser *DocumentParser) readTag() (Tag, bool, error) {
	parser.reader.MarkHere()
	if err := parser.readWhiteSpace(); err != nil {
		return Tag{}, false, fmt.Errorf("reading whitespace: %w", err)
	}

	parser.reader.MarkHere()
	closing, err := parser.readTagPrefix()
	if err != nil {
		return Tag{}, false, fmt.Errorf("reading whether tag is closing another tag: %w", err)
	}

	parser.reader.MarkHere()
	char, err := parser.reader.Read()
	if err != nil {
		return Tag{}, false, err
	}
	parser.reader.Unread()

	var name string
	if char != '.' {
		parser.reader.MarkHere()
		name, err = parser.readTagName()
		if err != nil {
			return Tag{}, false, fmt.Errorf("reading name: %w", err)
		}
	}

	parser.reader.MarkHere()
	args, err := parser.readTagArgs()
	if err != nil {
		return Tag{}, false, fmt.Errorf("reading arguments: %w", err)
	}

	return Tag{Name: name, Args: args}, closing, nil
}

func (parser *DocumentParser) readWhiteSpace() error {
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

func (parser *DocumentParser) readTagPrefix() (closing bool, err error) {
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

func (parser *DocumentParser) readTagName() (string, error) {
	var builder strings.Builder

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return "", err
		}

		if char == ' ' {
			return builder.String(), nil
		}

		if isNameCharset(char) {
			builder.WriteRune(char)
			continue
		}

		if char == '}' {
			parser.reader.Unread()
			return builder.String(), nil
		}
	}
}

func (parser *DocumentParser) readTagArgs() ([]Argument, error) {
	var args = []Argument{}
	var buffer = []rune{}
	var keyBuffer = []rune{}

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return args, err
		}

		if isNameCharset(char) {
			if len(buffer) == 0 {
				parser.reader.MarkHere()
			}
			buffer = append(buffer, char)
			continue
		}

		switch char {
		case '\'', '"', '`':
			parser.reader.MarkHere()
			arg, err := parser.readQuoted(char)
			if err != nil {
				return args, fmt.Errorf("parsing a quoted string: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgString,
				Key:  string(keyBuffer),
				Data: arg,
			})

		case ' ':
			parser.reader.MarkHere()
			if len(buffer) != 0 {
				asInt, err := strconv.Atoi(string(buffer))
				if err == nil {
					args = append(args, Argument{
						Kind: ArgInt,
						Key:  string(keyBuffer),
						Data: asInt,
					})
				} else {
					asFloat, err := strconv.ParseFloat(string(buffer), 64)
					if err != nil {
						return args, fmt.Errorf(
							"%q is not a number. Put quotes around if it's meant to be a string",
							string(buffer),
						)
					}
					args = append(args, Argument{
						Kind: ArgFloat,
						Key:  string(keyBuffer),
						Data: asFloat,
					})
				}
				keyBuffer = []rune{}
				buffer = []rune{}
			}

		case '}':
			parser.reader.MarkHere()
			char, err := parser.reader.Read()
			if err != nil {
				return args, fmt.Errorf("reading tag closing brace: %w", err)
			}
			if char != '}' {
				return args, fmt.Errorf("expected '}', but got %q", char)
			}
			return args, nil

		case '=':
			if len(keyBuffer) != 0 {
				return args, fmt.Errorf("unexpected character %q", char)
			}
			if len(buffer) == 0 {
				nextChar, err := parser.reader.Read()
				if err != nil {
					return args, fmt.Errorf("reading equality sign: %w", err)
				}
				if nextChar == '=' {
					buffer = []rune{}
					args = append(args, Argument{Kind: ArgOperation, Data: OpEquals})
					continue
				}
				parser.reader.Unread()
			}
			keyBuffer = buffer
			buffer = []rune{}

		case '&', '|':
			parser.reader.MarkHere()
			err := parser.readExactly(char)
			if err != nil {
				return args, fmt.Errorf("reading %q token: %w", char, err)
			}
			args = append(
				args,
				Argument{Kind: ArgOperation, Data: Operation(string(char) + string(char))},
			)

		case '!':
			parser.reader.MarkHere()
			err := parser.readExactly('=')
			if err != nil {
				return args, fmt.Errorf("reading '=' token: %w", err)
			}
			args = append(args, Argument{Kind: ArgOperation, Data: OpNotEquals})

		case '<', '>':
			parser.reader.MarkHere()
			nextChar, err := parser.reader.Read()
			if err != nil {
				return args, fmt.Errorf("reading inequality sign: %w", err)
			}
			if len(keyBuffer)+len(buffer) != 0 {
				return args, fmt.Errorf("unexpected character %q", char)
			}
			if nextChar != '=' {
				args = append(args, Argument{Kind: ArgOperation, Data: Operation(string(char))})
				parser.reader.Unread()
				continue
			}
			args = append(args, Argument{Kind: ArgOperation, Data: Operation(string(char) + "=")})

		case '.':
			if len(buffer) != 0 {
				buffer = append(buffer, '.')
				continue
			}
			parser.reader.MarkHere()
			path, err := parser.readPath()
			if err != nil {
				return args, fmt.Errorf("reading path: %w", err)
			}
			args = append(args, Argument{
				Kind: ArgPath,
				Key:  string(keyBuffer),
				Data: path,
			})

		default:
			return args, fmt.Errorf("unexpected character %q", char)
		}
	}
}

func (parser *DocumentParser) readExactly(expectedChar rune) error {
	parser.reader.MarkHere()
	actualChar, err := parser.reader.Read()
	if err != nil {
		return err
	}
	if actualChar != expectedChar {
		return fmt.Errorf("expected %q, but got %q", expectedChar, actualChar)
	}
	return nil
}

func (parser *DocumentParser) readQuoted(quote rune) (string, error) {
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

func (parser *DocumentParser) readPath() (StringPath, error) {
	var buffer = []rune{}

	for {
		char, err := parser.reader.Read()
		if err != nil {
			return StringPath{}, err
		}

		if isPathCharset(char) {
			buffer = append(buffer, char)
			continue
		}

		switch char {
		case ' ':
			return StringPath(strings.Split(string(buffer), ".")), nil

		case '(':
			parser.reader.MarkHere()
			if err := parser.readExactly(')'); err != nil {
				return StringPath{}, fmt.Errorf("reading function parentheses: %w", err)
			}
			buffer = append(buffer, '(', ')')

		default:
			return StringPath{}, fmt.Errorf("unexpected character %q", char)
		}
	}
}

func isNameCharset(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= '0' && char <= '9')
}

func isPathCharset(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_' ||
		char == '.'
}
