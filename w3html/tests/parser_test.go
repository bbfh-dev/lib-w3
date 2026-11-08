package w3html_test

import (
	"io"
	"os"
	"strings"
	"testing"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
)

type TestCase struct {
	File        MockFile
	ExpectError func(err w3errs.DocumentError) bool
	Expected    w3html.ParsedPage
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
			Filename: "SingleBraces",
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
	// ————————————————————————————————
	{
		File: MockFile{
			Filename: "Fail_NeverClosed",
			Contents: `<span>Hello World! from {{ ._filename.base`,
		},
		ExpectError: func(err w3errs.DocumentError) bool {
			return strings.HasSuffix(err.Message, "reading access path: EOF")
		},
	},
	{
		File: MockFile{
			Filename: "UnnamedWithKey",
			Contents: `{{ key=.Path.To().Value }}`,
		},
		ExpectError: func(err w3errs.DocumentError) bool {
			return strings.HasSuffix(
				err.Message,
				"reading tag: reading name: unexpected character: '='",
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
				cmpopts.IgnoreFields(w3html.NodeLocation{}, "Parent"),
			)
		})
	}
}
