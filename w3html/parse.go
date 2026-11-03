package w3html

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type HunkEnum uint8

const (
	HunkRaw HunkEnum = iota
	HunkTag
	HunkEmpty
)

func (enum HunkEnum) String() string {
	switch enum {
	case HunkRaw:
		return "HunkRaw"
	case HunkTag:
		return "HunkTag"
	case HunkEmpty:
		return "HunkEmpty"
	default:
		return fmt.Sprintf("HunkEnum(%d)", enum)
	}
}

type Hunk struct {
	Kind     HunkEnum
	Data     any
	Line     uint
	Children []Hunk
}

// MeaningfulChildren returns [Hunk.Children] that aren't purely whitespace.
func (hunk Hunk) MeaningfulChildren() []Hunk {
	out := make([]Hunk, 0, len(hunk.Children))
	for _, child := range hunk.Children {
		switch child.Kind {
		case HunkTag:
			if !child.Tag().IsClosing {
				// NOTE: ensure that this is applicable here.
				// Check whether the whitespace should be kept instead.
				child.Children = child.MeaningfulChildren()
				out = append(out, child)
			}
		case HunkRaw:
			buffer := child.Data.([]byte)
			if !strings.ContainsAny(string(buffer), "\t\n\v\f\r ") {
				out = append(out, child)
			}
		case HunkEmpty:
			// not meaningful
		}
	}
	return out
}

// Tag returns the [Tag] representation of [Hunk.Data].
// Panics if [Hunk.Data] is not a [Tag].
func (hunk Hunk) Tag() Tag {
	return hunk.Data.(Tag)
}

func (hunk Hunk) String() string {
	return hunk.StringIndented(0)
}

func (hunk Hunk) StringIndented(indent uint) string {
	var builder strings.Builder
	indentString := strings.Repeat("    ", int(indent))
	data := hunk.Data
	if data == nil {
		data = "(Empty)"
	}
	fmt.Fprintf(
		&builder,
		"%s— %s —%s\n%[1]s%[4]s%[5]s%[6]s\n",
		indentString,
		hunk.Kind,
		strings.Repeat("—", 40),
		libclog.AnsiColor(libclog.ColorBrightBlack),
		data,
		libclog.AnsiReset,
	)
	for _, child := range hunk.Children {
		builder.WriteString(child.StringIndented(indent + 1))
	}
	return builder.String()
}

type Tag struct {
	Name      string
	Args      []Argument
	IsClosing bool
}

func (tag Tag) IsPaired() bool {
	switch tag.Name {
	case "extend", "if", "slot":
		return true
	default:
		return false
	}
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
	builder.WriteString("{{")
	if tag.IsClosing {
		builder.WriteRune('/')
	}
	if len(tag.Name) != 0 {
		builder.WriteRune(' ')
		builder.WriteString(tag.Name)
	}
	for _, arg := range tag.Args {
		builder.WriteRune(' ')
		if arg.Key != "" {
			builder.WriteString(arg.Key)
			builder.WriteRune('=')
		}
		builder.WriteRune('"')
		quoted := strconv.Quote(arg.Value)
		quoted = quoted[1 : len(quoted)-1]
		builder.WriteString(quoted)
		builder.WriteRune('"')
	}
	builder.WriteString(" }}")
	return builder.String()
}

type ParsedPage struct {
	Filename string
	Slots    map[string][]uint
	Hunks    []Hunk
	// temporary, used during parsing & processing
	breadcrumbs []*Hunk
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

func (page *ParsedPage) appendOrMergeRaw(data []byte, line uint) {
	if len(page.Hunks) != 0 {
		last := &page.Hunks[len(page.Hunks)-1]
		if last.Kind == HunkRaw {
			last.Data = append(last.Data.([]byte), data...)
			return
		}
	}
	page.Hunks = append(page.Hunks, Hunk{
		Kind: HunkRaw,
		Data: data,
		Line: line,
	})
}

func (page *ParsedPage) Replace(path []uint, replacement ...Hunk) {
	page.replace(page.Hunks, path, replacement...)
}

func (page *ParsedPage) replace(children []Hunk, path []uint, replacement ...Hunk) {
	i := path[0]
	if len(path) == 1 {
		switch len(replacement) {
		case 0:
			children[i] = Hunk{
				Kind:     HunkEmpty,
				Data:     nil,
				Line:     children[i].Line,
				Children: nil,
			}
		case 1:
			replacement[0].Line = children[i].Line
			children[i] = replacement[0]
		default:
			children[i] = Hunk{
				Kind:     HunkEmpty,
				Data:     nil,
				Line:     children[i].Line,
				Children: replacement,
			}
		}
		return
	}

	page.replace(children[i].Children, path[1:], replacement...)
}

func (page ParsedPage) String() string {
	var builder strings.Builder
	builder.WriteString(strings.Repeat("—", 80))
	builder.WriteRune('\n')

	for _, hunk := range page.Hunks {
		builder.WriteString(hunk.String())
	}

	builder.WriteString(strings.Repeat("—", 80))
	return builder.String()
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
		Slots: nil,
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
			if tag.IsPaired() {
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
			} else if tag.IsClosing {
				return page, newTagError(
					page.Filename,
					Hunk{Data: tag, Line: line},
					fmt.Sprintf("%q tag is not a closable tag", tag.Name),
				)
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
	char, err := reader.Read()
	if err != nil {
		return Tag{}, err
	}
	reader.Unread()

	var name string
	if char != '.' {
		reader.MarkHere()
		name, err = readTagName(reader)
		if err != nil {
			return Tag{}, fmt.Errorf("reading name: %w", err)
		}
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
