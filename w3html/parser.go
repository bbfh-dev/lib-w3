package w3html

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"codeberg.org/bbfh/lib-w3/w3utils"
)

type DocumentParser struct {
	// TODO: Make this do something
	StripWhitespace bool
	file            *os.File
	reader          *w3utils.DetailedReader
	buffer          []byte
	line            uint
	page            ParsedPage
}

func Parse(file *os.File, stripWhitespace bool) (ParsedPage, error) {
	parser := DocumentParser{
		StripWhitespace: stripWhitespace,
		file:            file,
		reader:          w3utils.NewDetailedReader(nil, file),
		buffer:          []byte{},
		line:            1,
	}
	err := parser.Parse()
	if err != nil {
		return ParsedPage{}, parser.reader.WrapError(parser.file.Name(), err)
	}
	return parser.Result(), nil
}

func (parser *DocumentParser) Parse() error {
	parser.page = ParsedPage{
		Slots: map[string]SlotLocation{},
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
		parser.line = parser.reader.Line()

		// at this point it's expected to be a tag
		tag, closing, err := parser.readTag()
		if err != nil {
			return fmt.Errorf("reading tag: %w", err)
		}
		if !closing {
			if err := parser.process(tag); err != nil {
				parser.reader.MoveMarkTo(tagBeginning)
				return fmt.Errorf("processing tag: %w", err)
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
		parser.page.Append(Hunk{
			Kind:     HunkRaw,
			Data:     parser.buffer,
			Line:     parser.line,
			Children: nil,
		})
	}
	parser.buffer = []byte{}
}

func (parser *DocumentParser) writeRune(char rune) {
	encoded := make([]byte, 4)
	n := utf8.EncodeRune(encoded, char)
	parser.buffer = append(parser.buffer, encoded[:n]...)
}
