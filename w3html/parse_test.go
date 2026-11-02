package w3html_test

import (
	"os"
	"testing"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"gotest.tools/assert"
)

func TestParse(t *testing.T) {
	outputMode := libclog.OutputAuto
	if os.Getenv("COLOR") == "1" {
		outputMode = libclog.OutputColored
	}

	file, err := os.OpenFile("parse_test.01.html", os.O_RDONLY, os.ModePerm)
	assert.NilError(t, err)
	defer file.Close()

	page, err := w3html.Parse(file)
	if err != nil {
		w3errs.Write(err, os.Stderr, outputMode)
		t.FailNow()
	}

	assert.DeepEqual(t, page, w3html.ParsedPage{
		Filename: "parse_test.01.html",
		Hunks: []w3html.Hunk{
			{
				Kind: w3html.HunkTag,
				Line: 1,
				Data: w3html.Tag{
					Name: "include",
					Args: []w3html.Argument{
						{Key: "", Value: "_root"},
						{Key: "title", Value: "Hello World!"},
					},
					IsClosing: false,
				},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 1,
				Data: []byte("\n"),
			},
			{
				Kind: w3html.HunkTag,
				Line: 2,
				Data: w3html.Tag{
					Name:      "slot",
					Args:      []w3html.Argument{{Key: "", Value: "body"}},
					IsClosing: false,
				},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 2,
				Data: []byte("\n<main>\n    "),
			},
			{
				Kind: w3html.HunkTag,
				Line: 4,
				Data: w3html.Tag{
					Name: "if",
					Args: []w3html.Argument{
						{Key: "", Value: ".Page.String()"},
						{Key: "", Value: "=="},
						{Key: "", Value: "home"},
						{Key: "", Value: "&&"},
						{Key: "", Value: ".Index"},
						{Key: "", Value: ">"},
						{Key: "", Value: "1"},
					},
					IsClosing: false,
				},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 4,
				Data: []byte("\n    "),
			},
			{
				Kind: w3html.HunkTag,
				Line: 5,
				Data: w3html.Tag{
					Name: "include",
					Args: []w3html.Argument{
						{Key: "", Value: "_button"},
						{Key: "label", Value: "My Button"},
					},
					IsClosing: false,
				},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 5,
				Data: []byte("\n    "),
			},
			{
				Kind: w3html.HunkTag,
				Line: 6,
				Data: w3html.Tag{
					Name:      "include",
					Args:      []w3html.Argument{{Key: "", Value: "icons/eyecons.svg"}},
					IsClosing: false,
				},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 6,
				Data: []byte("\n    "),
			},
			{
				Kind: w3html.HunkTag,
				Line: 7,
				Data: w3html.Tag{Name: "include", Args: []w3html.Argument{}, IsClosing: true},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 7,
				Data: []byte("\n    "),
			},
			{
				Kind: w3html.HunkTag,
				Line: 8,
				Data: w3html.Tag{Name: "if", Args: []w3html.Argument{}, IsClosing: true},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 8,
				Data: []byte("\n</main>\n"),
			},
			{
				Kind: w3html.HunkTag,
				Line: 10,
				Data: w3html.Tag{Name: "slot", Args: []w3html.Argument{}, IsClosing: true},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 10,
				Data: []byte("\n"),
			},
			{
				Kind: w3html.HunkTag,
				Line: 11,
				Data: w3html.Tag{Name: "include", Args: []w3html.Argument{}, IsClosing: true},
			},
			{
				Kind: w3html.HunkRaw,
				Line: 11,
				Data: []byte("\n"),
			},
		},
	})
}
