package w3html

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"codeberg.org/bbfh/lib-w3/w3errs"
)

type Processor struct {
	In     *ParsedPage
	Out    *ParsedPage
	Params map[string]string
}

func NewProcessor(page *ParsedPage, params map[string]string) Processor {
	outPage := ParsedPage{
		Filename: page.Filename,
		Slots:    map[string][]uint{},
		Hunks:    make([]Hunk, 0, len(page.Hunks)),
	}
	return Processor{
		In:     page,
		Out:    &outPage,
		Params: params,
	}
}

func (proc Processor) Process() error {
	for _, hunk := range proc.In.Hunks {
		if err := proc.process(hunk); err != nil {
			if _, ok := err.(w3errs.WritableError); ok {
				return err
			}
			return newTagError(proc.In.Filename, hunk, err.Error())
		}
	}
	return nil
}

func (proc Processor) process(hunk Hunk) (err error) {
	switch hunk.Kind {

	case HunkRaw:
		proc.Out.appendOrMergeRaw(hunk.Data.([]byte), hunk.Line)

	case HunkTag:
		tag := hunk.Tag()
		if tag.IsClosing {
			proc.Out.breadcrumbs = proc.Out.breadcrumbs[:len(proc.Out.breadcrumbs)-1]
			return nil // ParsedPage has already validated these have pairs
		}

		if tag.IsPaired() {
			proc.Out.breadcrumbs = append(proc.Out.breadcrumbs, &hunk)
		}

		switch tag.Name {
		case "":
			err = proc.handleEmpty(hunk)
		case "include":
			err = proc.handleInclude(hunk)
		case "extend":
			err = proc.handleExtend(hunk)
		case "if":
			// TODO: this
		case "place":
			name := tag.FindNamedArg("name", 0)
			if name == "" {
				name = "default"
				return errors.New(
					"'place' tag requires a 'name' argument. Please, explicitly specify 'default' if that was your intent.",
				)
			}

			location := make([]uint, 0, len(proc.Out.Hunks)+len(proc.Out.breadcrumbs))
			location = append(location, uint(len(proc.Out.Hunks)))
			for _, tag := range proc.Out.breadcrumbs {
				location = append(location, uint(len(tag.Children)-1))
			}
			proc.Out.Slots[name] = location
			proc.Out.Hunks = append(proc.Out.Hunks, Hunk{
				Kind:     HunkEmpty,
				Data:     nil,
				Line:     hunk.Line,
				Children: nil,
			})
		case "slot":
			// ignore
		default:
			err = fmt.Errorf(
				"unknown tag function %q. Available are: %s",
				tag.Name,
				strings.Join([]string{"include", "extend", "slot", "if"}, ", "),
			)
		}

	case HunkEmpty:
		for _, child := range hunk.Children {
			if err := proc.process(child); err != nil {
				return err
			}
		}
	}

	return err
}

func (proc Processor) handleEmpty(hunk Hunk) error {
	if len(hunk.Tag().Args) == 0 {
		return nil // don't keep empty tags
	}

	// inline parameters:
	value := hunk.Tag().Args[0].Value
	after, ok := strings.CutPrefix(value, ".")
	if ok && strings.ToLower(after) == after {
		param, ok := proc.Params[after]
		if !ok {
			return fmt.Errorf(
				"undefined parameter %q.\n\tMake sure you\n\t1) provide it in the tag that includes/extends this file.\n\t2) didn't mean to access a struct field (use capitalized name in that case)",
				after,
			)
		}
		proc.Out.appendOrMergeRaw([]byte(param), hunk.Line)
		return nil
	}

	proc.Out.Hunks = append(proc.Out.Hunks, hunk)
	return nil
}

func (proc Processor) openMentionedDocument(tag Tag) (*os.File, error) {
	path := tag.FindNamedArg("path", 0)
	if path == "" {
		return nil, fmt.Errorf("%q tag requires 'path' argument", tag.Name)
	}
	path = filepath.Join(filepath.Dir(proc.In.Filename), path)
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = os.OpenFile(path+".html", os.O_RDONLY, os.ModePerm)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return file, nil
}

func (proc Processor) processMentionedDocument(tag Tag) (Processor, error) {
	file, err := proc.openMentionedDocument(tag)
	if err != nil {
		return proc, err
	}
	defer file.Close()

	nestedParsedDoc, err := Parse(file)
	if err != nil {
		return proc, err
	}

	nestedProcessor := NewProcessor(&nestedParsedDoc, tag.ArgsExcluding("path"))
	if err := nestedProcessor.Process(); err != nil {
		return proc, err
	}

	return nestedProcessor, nil
}

func (proc Processor) handleInclude(hunk Hunk) error {
	includedPage, err := proc.processMentionedDocument(hunk.Tag())
	if err != nil {
		return err
	}

	proc.Out.Hunks = append(proc.Out.Hunks, includedPage.Out.Hunks...)
	return nil
}

func (proc Processor) handleExtend(hunk Hunk) error {
	insertions := map[string][]Hunk{}
	for _, child := range hunk.MeaningfulChildren() {
		switch child.Kind {
		case HunkRaw:
			insertions["default"] = append(insertions["default"], child)
		case HunkTag:
			tag := child.Data.(Tag)
			if tag.Name == "slot" {
				name := tag.FindNamedArg("name", 0)
				if name == "" {
					return errors.New("'slot' tag requires 'name' argument")
				}
				if _, ok := insertions[name]; ok {
					return fmt.Errorf("'slot' with name %q is already placed", name)
				}
				insertions[name] = child.Children
			} else {
				insertions["default"] = append(insertions["default"], child)
			}
		}
	}

	extendedPage, err := proc.processMentionedDocument(hunk.Tag())
	if err != nil {
		return err
	}

	for slot, hunks := range insertions {
		path, ok := extendedPage.Out.Slots[slot]
		if !ok {
			return fmt.Errorf(
				"trying to inject into slot named %q, but it's undefined among [%s]",
				slot,
				strings.Join(slices.Collect(maps.Keys(extendedPage.Out.Slots)), ", "),
			)
		}

		extendedPage.Out.Replace(path, hunks...)
	}
	proc.Out.Hunks = append(proc.Out.Hunks, extendedPage.Out.Hunks...)

	return nil
}

func newTagError(filename string, hunk Hunk, message string) w3errs.TagError {
	return w3errs.TagError{
		Filename:   filename,
		LineNumber: hunk.Line,
		Context:    fmt.Sprintf("%s", hunk.Data),
		Message:    message,
	}
}
