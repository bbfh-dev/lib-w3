package w3html

import (
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

			switch tag.Name {

			case "":
				if len(tag.Args) == 0 {
					continue // don't keep empty tags
				}

				// inline parameters:
				value := tag.Args[0].Value
				after, ok := strings.CutPrefix(value, ".")
				if ok && strings.ToLower(after) == after {
					param, ok := proc.Params[after]
					if !ok {
						return newTagError(
							proc.In.Filename,
							hunk,
							fmt.Sprintf(
								"undefined parameter %q.\nMake sure you 1) provide it in the tag that includes/extends this file. 2) didn't mean to access a struct field (use capitalized name in that case)",
								after,
							),
						)
					}
					proc.Out.Hunks = append(proc.Out.Hunks, Hunk{
						Kind: HunkRaw,
						Data: param,
						Line: hunk.Line,
					})
					continue
				}
				proc.Out.Hunks = append(proc.Out.Hunks, hunk)

			case "include":
				path := tag.FindNamedArg("path", 0)
				if path == "" {
					return newTagError(
						proc.In.Filename,
						hunk,
						"'include' tag requires 'path' argument",
					)
				}

			case "extend":
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
								return newTagError(
									proc.In.Filename,
									child,
									"'slot' tag requires 'name' argument",
								)
							}
							if _, ok := insertions[name]; ok {
								return newTagError(
									proc.In.Filename,
									child,
									fmt.Sprintf("'slot' with name %q is already placed", name),
								)
							}
							insertions[name] = child.Children
						} else {
							insertions["default"] = append(insertions["default"], child)
						}
					}
				}

				path := tag.FindNamedArg("path", 0)
				if path == "" {
					return newTagError(
						proc.In.Filename,
						hunk,
						"'extend' tag requires 'path' argument",
					)
				}

				path = filepath.Join(filepath.Dir(proc.In.Filename), path)
				file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
				if err != nil {
					return newTagError(proc.In.Filename, hunk, err.Error())
				}
				defer file.Close()

				rawExtendPage, err := Parse(file)
				if err != nil {
					return err
				}

				extendPage, err := Process(rawExtendPage, tag.ArgsExcluding("path", "at"))
				if err != nil {
					return err
				}

				// FIXME: instead of "at" there should be a "slot" tag inside
				at := tag.FindNamedArg("at", 1)
				if at == "" {
					at = "default"
				}
				slotPath := extendPage.FindSlot(at)
				if slotPath == nil {
					return newTagError(proc.In.Filename, hunk, fmt.Sprintf(
						"unable to find a matching '{{ place %q }}' in the extended document",
						at,
					))
				}
				// TODO: this
				fmt.Println(extendPage)

			case "slot", "if":
				proc.Out.Hunks = append(proc.Out.Hunks, hunk)

			default:
				return newTagError(
					proc.In.Filename,
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
