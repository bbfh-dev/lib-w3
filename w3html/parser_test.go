package w3html_test

import (
	"os"
	"testing"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
)

var expected = w3html.ParsedPage{
	MentionedPaths: []w3html.StringPath{
		{"Page", "String()"},
		{"Index"},
		{"User", "Name()"},
		{"Buttons"},
		{"", "Name"},
	},
	Slots: map[string]w3html.HunkLocation{
		"default": {Parent: nil, ChildIndex: 1},
	},
	Root: w3html.Hunk{
		Kind: w3html.HunkRoot,
		Line: 1,
		Data: nil,
		Children: []w3html.Hunk{
			{
				Kind: w3html.HunkRoot,
				Line: 1,
				Data: nil,
				Children: []w3html.Hunk{
					{
						Kind: w3html.HunkRaw,
						Line: 1,
						Data: []byte(
							`<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hello World! inside parser_test.01.html</title>
    `,
						),
						Children: nil,
					},
					{
						Kind:     w3html.HunkPlaceholder,
						Line:     8,
						Data:     nil,
						Children: nil,
					},
					{
						Kind: w3html.HunkRaw,
						Line: 8,
						Data: []byte(`
</head>

<body>
    <header>
        Welcome `),
						Children: nil,
					},
					{
						Kind:     w3html.HunkVariable,
						Line:     13,
						Data:     w3html.StringPath{"User", "Name()"},
						Children: nil,
					},
					{
						Kind:     w3html.HunkRaw,
						Line:     13,
						Data:     []byte("\n    </header>\n    "),
						Children: nil,
					},
					{
						Kind: w3html.HunkRoot,
						Line: 15,
						Data: nil,
						Children: []w3html.Hunk{
							{
								Kind:     w3html.HunkRaw,
								Line:     2,
								Data:     []byte("\n<main>\n    "),
								Children: nil,
							},
							{
								Kind: w3html.HunkTag,
								Line: 4,
								Data: w3html.Tag{
									Name: "if",
									Args: []w3html.Argument{
										{
											Kind: w3html.ArgPath,
											Key:  "",
											Data: w3html.StringPath{"Page", "String()"},
										},
										{
											Kind: w3html.ArgOperation,
											Key:  "",
											Data: w3html.OpEquals,
										},
										{
											Kind: w3html.ArgString,
											Key:  "",
											Data: "home",
										},
										{
											Kind: w3html.ArgOperation,
											Key:  "",
											Data: w3html.OpAnd,
										},
										{
											Kind: w3html.ArgPath,
											Key:  "",
											Data: w3html.StringPath{"Index"},
										},
										{
											Kind: w3html.ArgOperation,
											Key:  "",
											Data: w3html.OpMoreThanStrict,
										},
										{
											Kind: w3html.ArgInt,
											Key:  "",
											Data: 1,
										},
									},
								},
								Children: []w3html.Hunk{
									{
										Kind: w3html.HunkRaw,
										Line: 4,
										Data: []byte(`
    <svg width="16" height="16" fill="currentColor" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
	<path
		d="m6 0c-3.3137 0-6 2.6863-6 6 0.0038 2.9248 2.1161 5.4207 5 5.9082v1.0918c0 1.662 1.338 3 3 3h5c1.662 0 3-1.338 3-3v-5c0-1.662-1.338-3-3-3h-1.0859c-0.48784-2.8861-2.987-4.999-5.9141-5zm0 2c1.8233 8.614e-4 3.4153 1.2346 3.8711 3h-3.8711c-0.55228 0-1 0.44772-1 1s0.44772 1 1 1h7c0.554 0 1 0.446 1 1v5c0 0.554-0.446 1-1 1h-5c-0.554 0-1-0.446-1-1v-1.0898c1.762-0.29696 3.3-1.3638 4.1953-2.9102h-2.5684c-0.72584 0.64015-1.6592 0.99544-2.627 1-2.2091 0-4-1.7909-4-4s1.7909-4 4-4z"
	/>
</svg>

    `),
										Children: nil,
									},
									{
										Kind:     w3html.HunkPlaceholder,
										Line:     6,
										Data:     nil,
										Children: nil,
									},
									{
										Kind:     w3html.HunkRaw,
										Line:     6,
										Data:     []byte("\n    "),
										Children: nil,
									},
								},
							},
							{
								Kind:     w3html.HunkRaw,
								Line:     7,
								Data:     []byte("\n</main>\n"),
								Children: nil,
							},
						},
					},
					{
						Kind:     w3html.HunkRaw,
						Line:     15,
						Data:     []byte("\n    <footer>\n        "),
						Children: nil,
					},
					{
						Kind: w3html.HunkLoop,
						Line: 17,
						Data: w3html.StringPath{"Buttons"},
						Children: []w3html.Hunk{
							{
								Kind:     w3html.HunkRaw,
								Line:     17,
								Data:     []byte("\n        <label>"),
								Children: nil,
							},
							{
								Kind:     w3html.HunkVariable,
								Line:     18,
								Data:     w3html.StringPath{"", "Name"},
								Children: nil,
							},
							{
								Kind: w3html.HunkRaw,
								Line: 18,
								Data: []byte(`</label>
        <button class="my-button">
    <svg width="16" height="16" fill="currentColor" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
	<path
		d="m6 0c-3.3137 0-6 2.6863-6 6 0.0038 2.9248 2.1161 5.4207 5 5.9082v1.0918c0 1.662 1.338 3 3 3h5c1.662 0 3-1.338 3-3v-5c0-1.662-1.338-3-3-3h-1.0859c-0.48784-2.8861-2.987-4.999-5.9141-5zm0 2c1.8233 8.614e-4 3.4153 1.2346 3.8711 3h-3.8711c-0.55228 0-1 0.44772-1 1s0.44772 1 1 1h7c0.554 0 1 0.446 1 1v5c0 0.554-0.446 1-1 1h-5c-0.554 0-1-0.446-1-1v-1.0898c1.762-0.29696 3.3-1.3638 4.1953-2.9102h-2.5684c-0.72584 0.64015-1.6592 0.99544-2.627 1-2.2091 0-4-1.7909-4-4s1.7909-4 4-4z"
	/>
</svg>

    <span>My Button</span>
</button>

        `),
								Children: nil,
							},
						},
					},
					{
						Kind:     w3html.HunkRaw,
						Line:     20,
						Data:     []byte("\n    </footer>\n</body>\n\n</html>\n"),
						Children: nil,
					},
				},
			},
			{
				Kind:     w3html.HunkRaw,
				Line:     10,
				Data:     []byte("\n"),
				Children: nil,
			},
		},
	},
}

func TestParser(t *testing.T) {
	outputMode := libclog.OutputAuto
	if os.Getenv("COLOR") == "1" {
		outputMode = libclog.OutputColored
	}

	file, err := os.OpenFile("parser_test.01.html", os.O_RDONLY, os.ModePerm)
	assert.NilError(t, err)
	defer file.Close()

	parser, err := w3html.Parse(nil, file, map[string]string{})
	if err != nil {
		w3errs.Write(err, os.Stderr, outputMode)
		t.FailNow()
	}

	// fmt.Println(parser.Result().String())

	assert.DeepEqual(
		t,
		expected,
		parser.Result(),
		cmpopts.IgnoreUnexported(w3html.ParsedPage{}),
		cmpopts.IgnoreFields(w3html.HunkLocation{}, "Parent"),
	)
}
