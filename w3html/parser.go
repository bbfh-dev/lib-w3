package w3html

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type DocumentParser struct {
	StripWhitespace bool
	Params          map[string]string
	file            *os.File
	reader          *w3utils.DetailedReader
	buffer          []byte
	line            uint
	page            ParsedPage
	openingTags     []OpeningTag
}

type OpeningTag struct {
	Hunk *Hunk
	Mark uint
	Line uint
}

func Parse(
	parent *w3utils.DetailedReader,
	file *os.File,
	stripWhitespace bool,
	params map[string]string,
) (ParsedPage, error) {
	reader := w3utils.NewDetailedReader(parent, file)
	filename := reader.Origin().FileName()
	params["_filename"] = filename
	params["_filename.base"] = filepath.Base(filename)
	parser := DocumentParser{
		StripWhitespace: stripWhitespace,
		Params:          params,
		file:            file,
		reader:          reader,
		buffer:          []byte{},
		line:            1,
		openingTags:     []OpeningTag{},
	}
	err := parser.Parse()
	if err != nil {
		if _, ok := err.(w3errs.DocumentError); ok {
			return ParsedPage{}, err
		}
		return ParsedPage{}, parser.reader.WrapError(parser.file.Name(), err)
	}
	return parser.Result(), nil
}

func (parser *DocumentParser) Parse() error {
	parser.page = ParsedPage{
		Slots: map[string]HunkLocation{},
		Root: Hunk{
			Kind:     HunkRoot,
			Line:     1,
			Data:     nil,
			Children: []Hunk{},
		},
		breadcrumbs: []*Hunk{},
	}

	for {
		char, err := parser.reader.Read()
		if err != nil {
			if err == io.EOF {
				parser.flush()
				return nil
			}
			return err
		}

		if char != '{' {
			// not a tag
			parser.writeRune(char)
			continue
		}

		// possibly a tag? look for second '{'
		tagBeginning := parser.reader.MarkHere()
		char, err = parser.reader.Read()
		if err != nil {
			if err == io.EOF {
				parser.writeRune('{')
				parser.flush()
				return nil
			}
			return err
		}

		if char != '{' {
			// it wasn't a tag
			parser.writeRune('{')
			parser.writeRune(char)
			continue
		}
		parser.flush()
		parser.line = parser.reader.Line

		// at this point it's expected to be a tag
		tag, closing, err := parser.readTag()
		if err != nil {
			return fmt.Errorf("reading tag: %w", err)
		}
		if closing {
			openingTag := parser.findMatchingOpeningTag(tag.Name)
			if openingTag == nil {
				return fmt.Errorf("tag %q was never opened", tag.Name)
			}
			line := parser.reader.Line
			parser.reader.Line = openingTag.Line
			// TODO: replace opening tag with new contents
			if err := parser.processClosing(openingTag.Hunk.AsTag()); err != nil {
				parser.reader.MoveMarkTo(openingTag.Mark)
				if _, ok := err.(w3errs.DocumentError); ok {
					return err
				}
				return fmt.Errorf("processing tag closing: %w", err)
			}
			parser.reader.Line = line
		} else {
			if err := parser.processOpening(tag, tagBeginning); err != nil {
				parser.reader.MoveMarkTo(tagBeginning)
				if _, ok := err.(w3errs.DocumentError); ok {
					return err
				}
				return fmt.Errorf("processing opening tag: %w", err)
			}
		}

		if tag.IsPaired() {
			if closing {
				err := parser.page.OnLeaveTagBody(tag)
				if err != nil {
					parser.reader.MoveMarkTo(tagBeginning)
					return err
				}
			} else {
				parser.page.OnEnterTagBody()
			}
		} else if closing {
			parser.reader.MoveMarkTo(tagBeginning)
			return fmt.Errorf("%q tag is not a closable tag", tag.Name)
		}
	}
}

func (parser *DocumentParser) Result() ParsedPage {
	return parser.page
}

func (parser *DocumentParser) flush() {
	if len(parser.buffer) != 0 {
		parser.appendBuffer()
	}
	parser.buffer = []byte{}
}

func (parser *DocumentParser) writeRune(char rune) {
	encoded := make([]byte, 4)
	n := utf8.EncodeRune(encoded, char)
	parser.buffer = append(parser.buffer, encoded[:n]...)
}

func (parser *DocumentParser) appendTag(tag Tag) {
	parser.page.Append(Hunk{
		Kind:     HunkTag,
		Line:     parser.line,
		Data:     tag,
		Children: nil,
	})
}

func (parser *DocumentParser) appendBuffer() {
	parser.appendRaw(parser.buffer)
}

func (parser *DocumentParser) appendRaw(buffer []byte) {
	if parser.StripWhitespace {
		buffer = bytes.TrimSpace(buffer)
		if len(buffer) == 0 {
			return // skip
		}
	}
	parser.page.Append(Hunk{
		Kind:     HunkRaw,
		Data:     buffer,
		Line:     parser.line,
		Children: nil,
	})
}

func (parser *DocumentParser) appendPlaceholder() {
	parser.page.Append(Hunk{
		Kind:     HunkPlaceholder,
		Line:     parser.line,
		Data:     nil,
		Children: nil,
	})
}

func (parser *DocumentParser) appendVariable(path string) {
	parser.page.Append(Hunk{
		Kind:     HunkVariable,
		Line:     parser.line,
		Data:     path,
		Children: nil,
	})
}

func (parser *DocumentParser) parseNestedDocument(tag Tag) (ParsedPage, error) {
	path := tag.FindNamedArg("path", 0)
	if path == "" {
		return ParsedPage{}, errors.New(
			"'include' tag requires a 'path' argument pointing to the file (relatively to current file location)",
		)
	}

	path = filepath.Join(filepath.Dir(parser.file.Name()), path)
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = os.OpenFile(path+".html", os.O_RDONLY, os.ModePerm)
			if err != nil {
				return ParsedPage{}, err
			}
		} else {
			return ParsedPage{}, err
		}
	}
	defer file.Close()

	return Parse(parser.reader, file, parser.StripWhitespace, tag.ArgsExcluding("path"))
}

func (parse *DocumentParser) findMatchingOpeningTag(name string) *OpeningTag {
	return findMatchingOpeningTag(parse.openingTags, name)
}

func findMatchingOpeningTag(openingTags []OpeningTag, name string) *OpeningTag {
	if len(openingTags) == 0 {
		return nil
	}

	last := len(openingTags) - 1
	if openingTags[last].Hunk.AsTag().Name == name {
		return &openingTags[last]
	}

	return findMatchingOpeningTag(openingTags[:last], name)
}
