package w3html

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type HunkEnum uint8

const (
	HunkRaw HunkEnum = iota
	HunkTag
)

type Hunk struct {
	Kind     HunkEnum
	Data     any
	Line     uint
	Children []Hunk
}

type Tag struct {
	Name      string
	Args      []Argument
	IsClosing bool
}

func (tag Tag) FindNamedArg(name string, i int) string {
	if len(tag.Args) == 0 {
		return ""
	}

	for _, arg := range tag.Args {
		if arg.Key == name {
			return arg.Value
		}
	}

	iLast := len(tag.Args) - 1
	return tag.Args[min(i, iLast)].Value
}

func (tag Tag) ArgsExcluding(names ...string) map[string]string {
	out := map[string]string{}
	for _, arg := range tag.Args {
		if !slices.Contains(names, arg.Key) {
			out[arg.Key] = arg.Value
		}
	}
	return out
}

func (tag Tag) String() string {
	var builder strings.Builder
	builder.WriteString("{{ ")
	if tag.IsClosing {
		builder.WriteRune('/')
	}
	builder.WriteString(tag.Name)
	for _, arg := range tag.Args {
		if arg.Key != "" {
			builder.WriteString(arg.Key)
			builder.WriteRune('=')
		}
		quoted := strconv.Quote(arg.Value)
		quoted = quoted[1 : len(quoted)-1]
		if quoted == arg.Value {
			builder.WriteString(arg.Value)
		} else {
			builder.WriteRune('"')
			builder.WriteString(quoted)
			builder.WriteRune('"')
		}
	}
	builder.WriteString(" }}")
	return builder.String()
}

type ParsedPage struct {
	Filename string
	Slots    map[string][]uint
	Hunks    []Hunk
	// temporary, used during parsing
	breadcrumbs []*Hunk
}

func (page ParsedPage) FindSlot(name string) []uint {
	if len(page.Slots) == 0 {
		return nil
	}
	path, ok := page.Slots[name]
	if !ok {
		return nil
	}
	return path
}

func (page *ParsedPage) walk(hunk *Hunk) *Hunk {
	if len(hunk.Children) == 0 {
		return hunk
	}
	return page.walk(&hunk.Children[len(hunk.Children)-1])
}

func (page *ParsedPage) enterTagBody() {
	page.breadcrumbs = append(
		page.breadcrumbs,
		page.walk(&page.Hunks[len(page.Hunks)-1]),
	)
}

func (page *ParsedPage) leaveTagBody(tag Tag) error {
	pivot := page.breadcrumbs[len(page.breadcrumbs)-1]
	pivotTag := pivot.Data.(Tag)
	if pivotTag.Name != tag.Name {
		return fmt.Errorf(
			"found closing tag for %q, but %q tag should be closed first!",
			tag.Name,
			pivotTag.Name,
		)
	}
	page.breadcrumbs = page.breadcrumbs[:len(page.breadcrumbs)-1]
	return nil
}

func (page *ParsedPage) add(hunk Hunk) {
	if len(page.breadcrumbs) == 0 {
		page.Hunks = append(page.Hunks, hunk)
		return
	}

	pivot := page.breadcrumbs[len(page.breadcrumbs)-1]
	pivot.Children = append(pivot.Children, hunk)
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
		if err, ok := err.(w3errs.TagError); ok {
			err.Filename = file.Name()
			return page, err
		}
		return page, reader.WrapError(file.Name(), err)
	}
	page.Filename = file.Name()
	return page, nil
}

func parse(reader *w3utils.DetailedReader) (ParsedPage, error) {
	page := ParsedPage{
		Hunks: []Hunk{},
		Slots: map[string][]uint{},
	}
	buffer := []byte{}
	line := reader.Line()

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
			reader.MarkHere()
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
				page.add(Hunk{
					Kind:     HunkRaw,
					Data:     buffer,
					Line:     line,
					Children: nil,
				})
				buffer = []byte{}
				line = reader.Line()
			}
			// at this point it has consumed both '{{'
			tag, err := readTag(reader)
			if err != nil {
				return page, fmt.Errorf("reading tag: %w", err)
			}
			page.add(Hunk{
				Kind:     HunkTag,
				Data:     tag,
				Line:     line,
				Children: nil,
			})
			switch tag.Name {

			case "place":
				if tag.IsClosing {
					return page, newTagError(
						page.Filename,
						Hunk{Data: tag, Line: line},
						"'place' tag is not a closable tag",
					)
				}
				// We do this during parsing simply because it's convenient.
				// Realitically that would be a part of processing.
				name := tag.FindNamedArg("name", 0)
				if name == "" {
					name = "default"
					return page, errors.New(
						"'place' tag requires a 'name' argument. Please, explicitly specify 'default' if that was your intent.",
					)
				}
				location := make([]uint, 0, len(page.Hunks)+len(page.breadcrumbs))
				location = append(location, uint(len(page.Hunks)-1))
				for _, tag := range page.breadcrumbs {
					location = append(location, uint(len(tag.Children)-1))
				}
				page.Slots[name] = location

			case "extend", "if", "slot":
				if tag.IsClosing {
					err := page.leaveTagBody(tag)
					if err != nil {
						return page, newTagError(
							page.Filename,
							Hunk{Data: tag, Line: line},
							err.Error(),
						)
					}
				} else {
					page.enterTagBody()
				}

			default:
				if tag.IsClosing {
					return page, newTagError(
						page.Filename,
						Hunk{Data: tag, Line: line},
						fmt.Sprintf("%q tag is not a closable tag", tag.Name),
					)
				}
			}

		default:
			buffer = append(buffer, byte(char))
		}
	}

finish:
	if len(buffer) != 0 {
		page.add(Hunk{
			Kind:     HunkRaw,
			Data:     buffer,
			Line:     line,
			Children: nil,
		})
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
			reader.MarkHere()
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
			if len(buffer) == 0 {
				nextChar, err := reader.Read()
				if err != nil {
					return args, fmt.Errorf("reading equality sign: %w", err)
				}
				if nextChar == '=' {
					buffer = []rune{}
					args = append(args, Argument{Key: "", Value: "=="})
					continue
				}
				reader.Unread()
			}
			keyBuffer = buffer
			buffer = []rune{}

		case '&', '|':
			reader.MarkHere()
			err := readSpecific(reader, char)
			if err != nil {
				return args, fmt.Errorf("reading specific token: %w", err)
			}
			args = append(args, Argument{Key: "", Value: string(char) + string(char)})

		case '<', '>', '!':
			reader.MarkHere()
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

		case '.':
			reader.MarkHere()
			path, err := readPath(reader)
			if err != nil {
				return args, fmt.Errorf("reading path: %w", err)
			}
			args = append(args, Argument{
				Key:   string(keyBuffer),
				Value: path,
			})

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

func readPath(reader *w3utils.DetailedReader) (string, error) {
	var buffer = []rune{}

	for {
		char, err := reader.Read()
		if err != nil {
			return "", err
		}

		if isPathCharset(char) {
			buffer = append(buffer, char)
			continue
		}

		switch char {
		case ' ':
			return "." + string(buffer), nil

		case '(':
			reader.MarkHere()
			if err := readSpecific(reader, ')'); err != nil {
				return "", fmt.Errorf("reading function parentheses: %w", err)
			}
			buffer = append(buffer, '(', ')')

		default:
			return "", fmt.Errorf("unexpected character %q", char)
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
