package w3html_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"codeberg.org/bbfh/lib-w3/w3utils"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
)

type TestCase struct {
	File        MockFile
	ExpectError func(err w3errs.DocumentError) bool
	Expected    w3html.ParsedPage
}

var Files = map[string]string{
	"include.1": "<span>\n\tHello World!\n</span>",
	"include.2": "{{ include 'include.2' }}",
	"include.3": "{{ .Root }}{{ ..This() }}",
	"extend.1":  "<section>\n\t{{ place 'default' }}\n</section>",
}

var TestCases = []TestCase{
	{
		File: MockFile{
			Filename: "AccessPath_CompileTime",
			Contents: `<span>Hello World! from {{ ._filename.base }}</span>`,
		},
		Expected: w3html.ParsedPage{
			AccessPaths: []w3html.AccessPath{},
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeHunk,
					Data: []byte("<span>Hello World! from AccessPath_CompileTime</span>"),
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "NotTag",
			Contents: `{ this should stay as text }`,
		},
		Expected: w3html.ParsedPage{
			AccessPaths: []w3html.AccessPath{},
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeHunk,
					Data: []byte("{ this should stay as text }"),
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Variable",
			Contents: `{{ .Path.To.Value() }}`,
		},
		Expected: w3html.ParsedPage{
			AccessPaths: []w3html.AccessPath{},
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdVariable,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Key:  "",
								Data: w3html.AccessPath{"Path", "To", "Value()"},
							},
						},
					},
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Include",
			Contents: "<div>\n{{ include 'include.1' }}\n</div>",
		},
		Expected: w3html.ParsedPage{
			AccessPaths: []w3html.AccessPath{},
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeHunk,
					Data: []byte("<div>\n<span>\nHello World!\n</span></div>"),
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Include_Scope",
			Contents: "<div>\n{{ include 'include.3' .My.Scope }}\n</div>",
		},
		Expected: w3html.ParsedPage{
			AccessPaths: []w3html.AccessPath{},
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeHunk,
					Data: []byte("<div>\n"),
				},
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdVariable,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Key:  "",
								Data: w3html.AccessPath{"Root"},
							},
						},
					},
				},
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdVariable,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Key:  "",
								Data: w3html.AccessPath{"My", "Scope", "This()"},
							},
						},
					},
				},
				{
					Kind: w3html.NodeHunk,
					Data: []byte("</div>"),
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Place",
			Contents: "<div>\n{{ place 'default' }}\n</div>",
		},
		Expected: w3html.ParsedPage{
			AccessPaths: []w3html.AccessPath{},
			PlacedSlots: map[string]w3html.NodeLocation{
				"default": {
					ChildIndex: 1,
					Line:       2,
				},
			},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeHunk,
					Data: []byte("<div>\n"),
				},
				{
					Kind: w3html.NodePlaceholder,
					Data: nil,
				},
				{
					Kind: w3html.NodeHunk,
					Data: []byte("</div>"),
				},
			},
		},
	},
	// ————————————————————————————————
	{
		File: MockFile{
			Filename: "Fail_NeverClosed",
			Contents: `<span>Hello World! from {{ ._filename.base`,
		},
		ExpectError: func(err w3errs.DocumentError) bool {
			return strings.Contains(err.Message, "reading access path: EOF")
		},
	},
	{
		File: MockFile{
			Filename: "Fail_AccessPath_CompileTime_Undefined",
			Contents: `<span>Hello World! from {{ ._filename.undefined }}</span>`,
		},
		ExpectError: func(err w3errs.DocumentError) bool {
			return strings.Contains(
				err.Message,
				"unknown compile-time variable \"_filename.undefined\"",
			)
		},
	},
	{
		File: MockFile{
			Filename: "Fail_UnnamedWithKey",
			Contents: `{{ key=.Path.To().Value }}`,
		},
		ExpectError: func(err w3errs.DocumentError) bool {
			return strings.Contains(
				err.Message,
				"reading name: unexpected character: '='",
			)
		},
	},
	{
		File: MockFile{
			Filename: "Fail_IncludeRecursive",
			Contents: "<div>\n{{ include 'include.2' }}\n</div>",
		},
		ExpectError: func(err w3errs.DocumentError) bool {
			return strings.Contains(
				err.Message,
				"trying to parse the same document recursively",
			)
		},
	},
}

type MockFile struct {
	Filename string
	Contents string
	reader   io.Reader
}

func (file *MockFile) Init() {
	file.reader = strings.NewReader(file.Contents)
}

func (file *MockFile) Name() string {
	return file.Filename
}

func (file *MockFile) Close() error {
	return nil
}

func (file *MockFile) Read(dest []byte) (int, error) {
	return file.reader.Read(dest)
}

func TestParser(t *testing.T) {
	outputMode := libclog.OutputAuto
	if os.Getenv("COLOR") == "1" {
		outputMode = libclog.OutputColored
	}

	printResult := os.Getenv("DEBUG") == "1"

	for _, testCase := range TestCases {
		testCase.File.Init()
		t.Run(testCase.File.Name(), func(tt *testing.T) {
			parser := w3html.NewParser(&testCase.File, map[string]string{})
			parser.WithFileOpener(func(absPath string) (w3utils.FileLike, error) {
				path := filepath.Base(absPath)
				contents, ok := Files[path]
				if !ok {
					return nil, fmt.Errorf("%s: no such file is supplied by the test file", path)
				}
				file := MockFile{
					Filename: path,
					Contents: contents,
				}
				file.Init()
				return &file, nil
			})

			result, err := parser.Parse()
			if err != nil {
				if testCase.ExpectError == nil {
					w3errs.Write(err, os.Stderr, outputMode)
					tt.FailNow()
					return
				} else {
					if derr, ok := err.(w3errs.DocumentError); ok {
						if testCase.ExpectError(derr) {
							return
						} else {
							w3errs.Write(err, os.Stderr, outputMode)
							tt.FailNow()
						}
					} else {
						tt.Fatalf("error is not a document error: %s", err.Error())
					}
				}
			}

			if printResult {
				for _, node := range result.Nodes {
					node.Write(tt.Output())
				}
			}

			assert.DeepEqual(
				tt,
				testCase.Expected,
				result,
				cmpopts.IgnoreUnexported(w3html.ParsedPage{}),
				cmpopts.IgnoreFields(w3html.NodeLocation{}, "Parent", "ReaderOffset"),
			)
		})
	}
}
