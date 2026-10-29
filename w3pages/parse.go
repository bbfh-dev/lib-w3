package w3pages

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"codeberg.org/bbfh/lib-w3/w3errs"
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
	var expected_size uint64
	stat, err := file.Stat()
	if err == nil {
		expected_size = uint64(stat.Size())
	}
	reader := w3errs.NewDetailedReader(bufio.NewReader(file), expected_size)
	page, err := parse(reader)
	if err != nil {
		return page, reader.WrapError(file.Name(), err)
	}
	return page, nil
}

func parse(reader *w3errs.DetailedReader) (ParsedPage, error) {
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
			next_char, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					buffer = append(buffer, byte(next_char)) // put it as a plain '{'
					goto finish
				}
				return page, err
			}
			if next_char != '{' { // not '{{'
				buffer = append(buffer, byte(char), byte(next_char))
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
				return page, fmt.Errorf("reading tag '{{ ... }}-like structure': %w", err)
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

func readTag(reader *w3errs.DetailedReader) (Tag, error) {
	reader.MarkHere()
	if err := readWhiteSpace(reader); err != nil {
		return Tag{}, fmt.Errorf("reading whitespace: %w", err)
	}

	reader.MarkHere()
	is_closing, err := readTagPrefix(reader)
	if err != nil {
		return Tag{}, fmt.Errorf("reading whether tag is closing another tag: %w", err)
	}

	reader.MarkHere()
	name, err := readTagName(reader)
	if err != nil {
		return Tag{}, fmt.Errorf("reading tag name: %w", err)
	}

	reader.MarkHere()
	args, err := readTagArgs(reader)
	if err != nil {
		return Tag{}, fmt.Errorf("reading tag arguments: %w", err)
	}

	return Tag{
		Name:      name,
		Args:      args,
		IsClosing: is_closing,
	}, nil
}

func readWhiteSpace(reader *w3errs.DetailedReader) error {
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

func readTagPrefix(reader *w3errs.DetailedReader) (is_closing bool, err error) {
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

func readTagName(reader *w3errs.DetailedReader) (string, error) {
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

func readTagArgs(reader *w3errs.DetailedReader) ([]Argument, error) {
	var args = []Argument{}
	var buffer = []rune{}
	var key_buffer = []rune{}

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
				Key:   string(key_buffer),
				Value: arg,
			})

		case ' ':
			if len(buffer) != 0 {
				args = append(args, Argument{
					Key:   string(key_buffer),
					Value: string(buffer),
				})
				key_buffer = []rune{}
				buffer = []rune{}
			}

		case '}':
			char, err := reader.Read()
			if err != nil {
				return args, fmt.Errorf("reading tag closing brace: %w", err)
			}
			if char != '}' {
				return args, fmt.Errorf("expected '}', but got %q", char)
			}
			return args, nil

		case '=':
			if len(key_buffer) != 0 {
				return args, fmt.Errorf("unexpected character %q", char)
			}
			key_buffer = buffer
			buffer = []rune{}

		default:
			return args, fmt.Errorf("unexpected character %q", char)
		}
	}
}

func readQuoted(reader *w3errs.DetailedReader, quote rune) (string, error) {
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
