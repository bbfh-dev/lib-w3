package w3html

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/bbfh/lib-w3/w3errs"
)

type Source struct {
	Filename string
	Line     uint
}

func Process(inPage ParsedPage, params map[string]string) (ParsedPage, error) {
	outPage := ParsedPage{
		Filename: inPage.Filename,
		Hunks:    make([]Hunk, 0, len(inPage.Hunks)),
	}
	err := process(&inPage, &outPage, params)
	return outPage, err
}

func process(inPage, outPage *ParsedPage, params map[string]string) error {
	for _, hunk := range inPage.Hunks {
		switch hunk.Kind {

		case HunkRaw:
			outPage.Hunks = append(outPage.Hunks, hunk)

		case HunkTag:
			tag := hunk.Data.(Tag)
			if tag.IsClosing {
				continue // ParsedPage has already validated these have pairs
			}

			switch tag.Name {

			case "":
				if len(tag.Args) == 0 {
					continue // don't keep empty tags
				}

				// inline parameters:
				value := tag.Args[0].Value
				after, ok := strings.CutPrefix(value, ".")
				if ok && strings.ToLower(after) == after {
					param, ok := params[after]
					if !ok {
						return newTagError(
							inPage.Filename,
							hunk,
							fmt.Sprintf(
								"undefined parameter %q.\nMake sure you 1) provide it in the tag that includes/extends this file. 2) didn't mean to access a struct field (use capitalized name in that case)",
								after,
							),
						)
					}
					outPage.Hunks = append(outPage.Hunks, Hunk{
						Kind: HunkRaw,
						Data: param,
						Line: hunk.Line,
					})
					continue
				}
				outPage.Hunks = append(outPage.Hunks, hunk)

			case "include":
				path := tag.FindNamedArg("path", 0)
				if path == "" {
					return newTagError(
						inPage.Filename,
						hunk,
						"'include' tag requires 'path' argument",
					)
				}

			case "extend":
				path := tag.FindNamedArg("path", 0)
				if path == "" {
					return newTagError(
						inPage.Filename,
						hunk,
						"'extend' tag requires 'path' argument",
					)
				}

				path = filepath.Join(filepath.Dir(inPage.Filename), path)
				file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
				if err != nil {
					return newTagError(inPage.Filename, hunk, err.Error())
				}
				defer file.Close()

				extendPage, err := Parse(file)
				if err != nil {
					return err
				}

				at := tag.FindNamedArg("at", 1)
				if at == "" {
					at = "default"
				}
				slot := extendPage.FindSlot(at)
				if slot == -1 {
					return newTagError(inPage.Filename, hunk, fmt.Sprintf(
						"unable to find a matching '{{ place %q }}' in the extended document",
						at,
					))
				}
				// hunksBefore := extendPage.Hunks[:slot]
				// hunksAfter := extendPage.Hunks[slot+1:]
				fmt.Println(extendPage)

			case "slot", "if":
				outPage.Hunks = append(outPage.Hunks, hunk)

			default:
				return newTagError(
					inPage.Filename,
					hunk,
					fmt.Sprintf(
						"unknown tag function %q. Available are: %s",
						tag.Name,
						strings.Join([]string{"include", "extend", "slot", "if"}, ", "),
					),
				)
			}
		}
	}

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
