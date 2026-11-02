package w3html

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/bbfh/lib-w3/w3errs"
)

type Processor struct {
	In     *ParsedPage
	Out    *ParsedPage
	Params map[string]string
}

func Process(inPage ParsedPage, params map[string]string) (ParsedPage, error) {
	outPage := ParsedPage{
		Filename: inPage.Filename,
		Slots:    inPage.Slots,
		Hunks:    make([]Hunk, 0, len(inPage.Hunks)),
	}
	proc := Processor{
		In:     &inPage,
		Out:    &outPage,
		Params: params,
	}
	err := proc.Run()
	return outPage, err
}

func (proc Processor) Run() error {
	for _, hunk := range proc.In.Hunks {
		switch hunk.Kind {

		case HunkRaw:
			proc.Out.Hunks = append(proc.Out.Hunks, hunk)

		case HunkTag:
			tag := hunk.Data.(Tag)
			if tag.IsClosing {
				continue // ParsedPage has already validated these have pairs
			}

			var err error
			switch tag.Name {
			case "":
				err = proc.handleEmpty(hunk)
			case "include":
				err = proc.handleInclude(hunk)
			case "extend":
				err = proc.handleExtend(hunk)
			case "if":
				// TODO: this
			case "slot":
				// ignore
			default:
				err = fmt.Errorf(
					"unknown tag function %q. Available are: %s",
					tag.Name,
					strings.Join([]string{"include", "extend", "slot", "if"}, ", "),
				)
			}

			if err != nil {
				if _, ok := err.(w3errs.WritableError); ok {
					return err
				}
				return newTagError(proc.In.Filename, hunk, err.Error())
			}
		}
	}

	return nil
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
				"undefined parameter %q.\nMake sure you 1) provide it in the tag that includes/extends this file. 2) didn't mean to access a struct field (use capitalized name in that case)",
				after,
			)
		}
		proc.Out.Hunks = append(proc.Out.Hunks, Hunk{
			Kind: HunkRaw,
			Data: param,
			Line: hunk.Line,
		})
		return nil
	}

	proc.Out.Hunks = append(proc.Out.Hunks, hunk)
	return nil
}

func (proc Processor) handleInclude(hunk Hunk) error {
	path := hunk.Tag().FindNamedArg("path", 0)
	if path == "" {
		return errors.New("'include' tag requires 'path' argument")
	}
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

	path := hunk.Tag().FindNamedArg("path", 0)
	if path == "" {
		return errors.New("'extend' tag requires 'path' argument")
	}

	path = filepath.Join(filepath.Dir(proc.In.Filename), path)
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	rawExtendPage, err := Parse(file)
	if err != nil {
		return err
	}

	extendPage, err := Process(rawExtendPage, hunk.Tag().ArgsExcluding("path", "at"))
	if err != nil {
		return err
	}

	// FIXME: instead of "at" there should be a "slot" tag inside
	at := hunk.Tag().FindNamedArg("at", 1)
	if at == "" {
		at = "default"
	}
	slotPath := extendPage.FindSlot(at)
	if slotPath == nil {
		return fmt.Errorf(
			"unable to find a matching '{{ place %q }}' in the extended document",
			at,
		)
	}
	// TODO: this
	fmt.Println(extendPage)
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
