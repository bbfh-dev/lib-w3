package w3html_test

import (
	"fmt"
	"os"
	"testing"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
)

type TestCase struct {
	Name     string
	Filename string
	Expected w3html.ParsedPage
}

// TODO: Fix indentation / whitespace
var TestCases = []TestCase{
	{
		Name:     "Extend_OnlyRaw",
		Filename: "01_only_raw.html",
		Expected: w3html.ParsedPage{
			MentionedPaths: []w3html.StringPath{},
			Slots:          map[string]w3html.HunkLocation{},
			Root: w3html.Hunk{
				Kind: w3html.HunkRoot,
				Line: 1,
				Data: nil,
				Children: []w3html.Hunk{
					{
						Kind: w3html.HunkRaw,
						Line: 1,
						Data: []byte(`<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hello World! inside 01_only_raw.html</title>
</head>

<body>
    
<main>
    Hello World!
</main>

</body>

</html>

`),
						Children: nil,
					},
				},
			},
		},
	},
	{
		Name:     "Extend_Nested",
		Filename: "02_nested_extend.html",
		Expected: w3html.ParsedPage{
			MentionedPaths: []w3html.StringPath{},
			Slots:          map[string]w3html.HunkLocation{},
			Root: w3html.Hunk{
				Kind:     w3html.HunkRoot,
				Line:     1,
				Data:     nil,
				Children: []w3html.Hunk{},
			},
		},
	},
}

func TestParser(t *testing.T) {
	outputMode := libclog.OutputAuto
	if os.Getenv("COLOR") == "1" {
		outputMode = libclog.OutputColored
	}

	printResult := os.Getenv("DEBUG") == "1"

	for _, testCase := range TestCases {
		t.Run(testCase.Name, func(tt *testing.T) {
			file, err := os.OpenFile(testCase.Filename, os.O_RDONLY, os.ModePerm)
			assert.NilError(t, err)
			defer file.Close()

			parser, err := w3html.ParseDocument(file)
			if err != nil {
				w3errs.Write(err, os.Stderr, outputMode)
				tt.FailNow()
				return
			}

			if printResult {
				fmt.Println(parser.Result().String())
			}

			assert.DeepEqual(
				tt,
				testCase.Expected,
				parser.Result(),
				cmpopts.IgnoreUnexported(w3html.ParsedPage{}),
				cmpopts.IgnoreFields(w3html.HunkLocation{}, "Parent"),
			)
		})
	}
}
