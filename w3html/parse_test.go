package w3html_test

import (
	"os"
	"testing"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"gotest.tools/assert"
)

func TestParse(t *testing.T) {
	file, err := os.OpenFile("parse_test.01.html", os.O_RDONLY, os.ModePerm)
	assert.NilError(t, err)
	defer file.Close()

	page, err := w3html.Parse(file)
	if err != nil {
		w3errs.Write(err, os.Stderr)
		t.FailNow()
	}

	assert.DeepEqual(t, page, w3html.ParsedPage{
		Hunks: []w3html.HunkEnum{
			w3html.HunkBlock,
			w3html.HunkRaw,
			w3html.HunkBlock,
			w3html.HunkRaw,
			w3html.HunkBlock,
			w3html.HunkRaw,
			w3html.HunkBlock,
			w3html.HunkRaw,
			w3html.HunkBlock,
			w3html.HunkRaw,
			w3html.HunkBlock,
			w3html.HunkRaw,
			w3html.HunkBlock,
			w3html.HunkRaw,
			w3html.HunkBlock,
			w3html.HunkRaw,
			w3html.HunkBlock,
			w3html.HunkRaw,
		},
		HunkData: []any{
			w3html.Tag{
				Name: "include",
				Args: []w3html.Argument{
					{Key: "", Value: "_root"},
					{Key: "title", Value: "Hello World!"},
				},
				IsClosing: false,
			},
			[]byte("\n"),
			w3html.Tag{
				Name:      "slot",
				Args:      []w3html.Argument{{Key: "", Value: "body"}},
				IsClosing: false,
			},
			[]byte("\n<main>\n    "),
			w3html.Tag{
				Name: "if",
				Args: []w3html.Argument{
					{Key: "", Value: ".Page.String()"},
					{Key: "", Value: "=="},
					{
						Key:   "",
						Value: "home",
					},
					{Key: "", Value: "&&"},
					{Key: "", Value: ".Index"},
					{Key: "", Value: ">"},
					{Key: "", Value: "1"},
				},
				IsClosing: false,
			},
			[]byte("\n    "),
			w3html.Tag{
				Name: "include",
				Args: []w3html.Argument{
					{Key: "", Value: "_button"},
					{Key: "label", Value: "My Button"},
				},
				IsClosing: false,
			},
			[]byte("\n    "),
			w3html.Tag{
				Name:      "include",
				Args:      []w3html.Argument{{Key: "", Value: "icons/eyecons.svg"}},
				IsClosing: false,
			},
			[]byte("\n    "),
			w3html.Tag{Name: "include", Args: []w3html.Argument{}, IsClosing: true},
			[]byte("\n    "),
			w3html.Tag{Name: "if", Args: []w3html.Argument{}, IsClosing: true},
			[]byte("\n</main>\n"),
			w3html.Tag{Name: "slot", Args: []w3html.Argument{}, IsClosing: true},
			[]byte("\n"),
			w3html.Tag{Name: "include", Args: []w3html.Argument{}, IsClosing: true},
			[]byte("\n"),
		},
	})
}
