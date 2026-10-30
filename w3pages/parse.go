package w3pages

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"codeberg.org/bbfh/lib-w3/w3utils"
)

type HunkEnum uint8

const (
	HunkRaw HunkEnum = iota
	HunkBlock
)

type Tag struct {
	Name      string
	Args      []Argument
	IsClosing bool
}

type ParsedPage struct {
	Hunks    []HunkEnum
	HunkData []any
}

func Parse(file *os.File) (ParsedPage, error) {
	var expectedSize uint64
	stat, err := file.Stat()
	if err == nil {
		expectedSize = uint64(stat.Size())
	}
	reader := w3utils.NewDetailedReader(bufio.NewReader(file), expectedSize)
	page, err := parse(reader)
	if err != nil {
		return page, reader.WrapError(file.Name(), err)
	}
	return page, nil
}

func parse(reader *w3utils.DetailedReader) (ParsedPage, error) {
	page := ParsedPage{
		Hunks:    []HunkEnum{},
		HunkData: []any{},
	}
	buffer := []byte{}

	for {
		char, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				goto finish
			}
			return page, err
		}

		switch char {
		case '{':
			nextChar, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					buffer = append(buffer, byte(nextChar)) // put it as a plain '{'
					goto finish
				}
				return page, err
			}
			if nextChar != '{' { // not '{{'
				buffer = append(buffer, byte(char), byte(nextChar))
				continue
			}
			if len(buffer) != 0 {
				page.HunkData = append(page.HunkData, buffer)
				page.Hunks = append(page.Hunks, HunkRaw)
				buffer = []byte{}
			}
			// at this point it has consumed both '{{'
			block, err := readTag(reader)
			if err != nil {
				return page, fmt.Errorf("reading tag: %w", err)
			}
			page.HunkData = append(page.HunkData, block)
			page.Hunks = append(page.Hunks, HunkBlock)

		default:
			buffer = append(buffer, byte(char))
		}
	}

finish:
	if len(buffer) != 0 {
		page.HunkData = append(page.HunkData, buffer)
		page.Hunks = append(page.Hunks, HunkRaw)
	}
	return page, nil
}

func readTag(reader *w3utils.DetailedReader) (Tag, error) {
	reader.MarkHere()
	if err := readWhiteSpace(reader); err != nil {
		return Tag{}, fmt.Errorf("reading whitespace: %w", err)
	}

	reader.MarkHere()
	isClosing, err := readTagPrefix(reader)
	if err != nil {
		return Tag{}, fmt.Errorf("reading whether tag is closing another tag: %w", err)
	}

	reader.MarkHere()
	name, err := readTagName(reader)
	if err != nil {
		return Tag{}, fmt.Errorf("reading name: %w", err)
	}

	reader.MarkHere()
	args, err := readTagArgs(reader)
	if err != nil {
		return Tag{}, fmt.Errorf("reading arguments: %w", err)
	}

	return Tag{
		Name:      name,
		Args:      args,
		IsClosing: isClosing,
	}, nil
}

func readWhiteSpace(reader *w3utils.DetailedReader) error {
	for {
		char, err := reader.Read()
		if err != nil {
			return err
		}

		if unicode.IsSpace(char) {
			continue
		}

		reader.Unread()
		return nil
	}
}

func readTagPrefix(reader *w3utils.DetailedReader) (isClosing bool, err error) {
	char, err := reader.Read()
	if err != nil {
		return false, err
	}
	if char == '/' {
		return true, nil
	}
	reader.Unread()
	return false, nil
}

func readTagName(reader *w3utils.DetailedReader) (string, error) {
	var builder strings.Builder

	for {
		char, err := reader.Read()
		if err != nil {
			return "", err
		}

		if char == ' ' {
			goto finish
		}

		if isNameCharset(char) {
			builder.WriteRune(char)
			continue
		}

		if char == '}' {
			reader.Unread()
			goto finish
		}
	}

finish:
	return builder.String(), nil
}

type Argument struct {
	Key, Value string
}

func (arg Argument) HasKey() bool {
	return arg.Key != ""
}

func readTagArgs(reader *w3utils.DetailedReader) ([]Argument, error) {
	var args = []Argument{}
	var buffer = []rune{}
	var keyBuffer = []rune{}

	for {
		char, err := reader.Read()
		if err != nil {
			return args, err
		}

		if isNameCharset(char) {
			if len(buffer) == 0 {
				reader.MarkHere()
			}
			buffer = append(buffer, char)
			continue
		}

		switch char {
		case '\'', '"', '`':
			reader.MarkHere()
			arg, err := readQuoted(reader, char)
			if err != nil {
				return args, fmt.Errorf("parsing a quoted string: %w", err)
			}
			args = append(args, Argument{
				Key:   string(keyBuffer),
				Value: arg,
			})

		case ' ':
			if len(buffer) != 0 {
				args = append(args, Argument{
					Key:   string(keyBuffer),
					Value: string(buffer),
				})
				keyBuffer = []rune{}
				buffer = []rune{}
			}

		case '}':
			reader.MarkHere()
			char, err := reader.Read()
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
			if len(buffer) == 0 || buffer[0] == '=' {
				buffer = []rune{}
				args = append(args, Argument{Key: "", Value: "=="})
				continue
			}
			keyBuffer = buffer
			buffer = []rune{}

		case '&', '|':
			err := readSpecific(reader, char)
			if err != nil {
				return args, fmt.Errorf("reading specific token: %w", err)
			}
			args = append(args, Argument{Key: "", Value: string(char) + string(char)})

		case '<', '>', '!':
			nextChar, err := reader.Read()
			if err != nil {
				return args, fmt.Errorf("reading inequality sign: %w", err)
			}
			if len(keyBuffer)+len(buffer) != 0 {
				return args, fmt.Errorf("unexpected character %q", char)
			}
			if nextChar != '=' {
				args = append(args, Argument{Key: "", Value: string(char)})
				reader.Unread()
				continue
			}
			args = append(args, Argument{Key: "", Value: string(char) + "="})

		default:
			return args, fmt.Errorf("unexpected character %q", char)
		}
	}
}

func readSpecific(reader *w3utils.DetailedReader, expectedChar rune) error {
	reader.MarkHere()
	actualChar, err := reader.Read()
	if err != nil {
		return err
	}
	if actualChar != expectedChar {
		return fmt.Errorf("expected %q, but got %q", expectedChar, actualChar)
	}
	return nil
}

func readQuoted(reader *w3utils.DetailedReader, quote rune) (string, error) {
	var buffer = []rune{}

	for {
		char, err := reader.Read()
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

func isNameCharset(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_' ||
		char == '-' ||
		char == '.'
}
