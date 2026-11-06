package w3html

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type DocumentParser struct {
	Params      map[string]string
	file        *os.File
	reader      *w3utils.DetailedReader
	buffer      []byte
	line        uint
	page        ParsedPage
	openingTags []HunkLocation
	scopes      []StringPath
}

type OpeningTag struct {
	Hunk *Hunk
	Mark uint
	Line uint
}

func ParseDocument(file *os.File) (*DocumentParser, error) {
	return Parse(nil, file, map[string]string{})
}

func Parse(
	parent *w3utils.DetailedReader,
	file *os.File,
	params map[string]string,
) (*DocumentParser, error) {
	reader := w3utils.NewDetailedReader(parent, file)
	filename := reader.Origin().FileName()
	params["_filename"] = filename
	params["_filename.base"] = filepath.Base(filename)
	parser := DocumentParser{
		Params:      params,
		file:        file,
		reader:      reader,
		buffer:      []byte{},
		line:        1,
		openingTags: []HunkLocation{},
		scopes:      []StringPath{},
	}
	err := parser.Parse()
	if err != nil {
		if _, ok := err.(w3errs.DocumentError); ok {
			return nil, err
		}
		return nil, parser.reader.WrapError(parser.file.Name(), err)
	}
	return &parser, nil
}

func (parser *DocumentParser) Parse() error {
	parser.page = ParsedPage{
		MentionedPaths: []StringPath{},
		Slots:          map[string]HunkLocation{},
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
		// at this point it's expected to be a tag
		parser.flush()
		parser.line = parser.reader.Line

		tag, closing, err := parser.readTag()
		if err != nil {
			return fmt.Errorf("reading tag: %w", err)
		}
		if closing {
			if !tag.IsPaired() {
				parser.reader.MoveMarkTo(tagBeginning)
				return fmt.Errorf("%q tag is not a closable tag", tag.Name)
			}

			openingTag := parser.findMatchingOpeningTag(tag.Name)
			if openingTag.IsNull() {
				return fmt.Errorf("tag %q was never opened", tag.Name)
			}

			err := parser.page.OnLeaveTagBody(tag)
			if err != nil {
				parser.reader.MoveMarkTo(tagBeginning)
				return err
			}

			line := parser.reader.Line
			parser.reader.Line = openingTag.Line
			if err := parser.processClosing(openingTag); err != nil {
				parser.reader.MoveMarkTo(openingTag.ByteOffset)
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
			if tag.IsPaired() {
				parser.page.OnEnterTagBody()
			}
		}
	}
}

func (parser *DocumentParser) Result() ParsedPage {
	// TODO: flatten the tree
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

func (parser *DocumentParser) appendVariable(path StringPath) {
	parser.page.Append(Hunk{
		Kind:     HunkVariable,
		Line:     parser.line,
		Data:     path,
		Children: nil,
	})
}

func (parser *DocumentParser) parseNestedDocument(tag Tag) (ParsedPage, error) {
	pathArg, err := tag.RequireNamedArg(ArgString, "path", 0)
	if err != nil {
		return ParsedPage{}, fmt.Errorf("%w (the path is relative to THIS document)", err)
	}
	path := pathArg.AsString()

	placeholderCount := strings.Count(path, "%s")
	if placeholderCount != 0 {
		formatArgs := tag.FormatArgs()
		if placeholderCount != len(formatArgs) {
			return ParsedPage{}, fmt.Errorf(
				"'path' contains %d formatting placeholders ('%%s'), but %d (%q) compile-time paths are provided for substitution among %d other ignored arguments",
				placeholderCount,
				len(formatArgs),
				formatArgs,
				len(tag.Args)-len(formatArgs),
			)
		}
		args := make([]any, len(formatArgs))
		for i, arg := range formatArgs {
			value, ok := parser.Params[arg.Flatten()]
			if !ok {
				return ParsedPage{}, fmt.Errorf(
					"unknown compile-time variable %q. Make sure the variable is set in the 'include' or 'extend' tag calling this file",
					path,
				)
			}
			args[i] = value
		}
		path = fmt.Sprintf(path, args...)
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

	nestedParser, err := Parse(parser.reader, file, tag.StringArgsExcluding("path"))
	if err != nil {
		return ParsedPage{}, err
	}
	parser.page.MentionedPaths = append(
		parser.page.MentionedPaths,
		nestedParser.page.MentionedPaths...)
	return nestedParser.Result(), nil
}

func (parse *DocumentParser) findMatchingOpeningTag(name string) HunkLocation {
	return findMatchingOpeningTag(parse.openingTags, name)
}

func findMatchingOpeningTag(openingTags []HunkLocation, name string) HunkLocation {
	if len(openingTags) == 0 {
		return HunkLocation{}
	}

	last := len(openingTags) - 1
	if openingTags[last].Hunk().AsTag().Name == name {
		return openingTags[last]
	}

	return findMatchingOpeningTag(openingTags[:last], name)
}
