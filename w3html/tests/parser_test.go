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
	File     MockFile
	Expected w3html.ParsedPage
}

var TestCases = []TestCase{
	{
		File: MockFile{
			Filename: "Simple_CompileTimeVariables",
			Contents: `<span>Hello World! from {{ ._filename.base }}</span>`,
		},
		Expected: w3html.ParsedPage{
			AccessPaths: []w3html.AccessPath{},
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeHunk,
					Data: []byte("<span>Hello World! from Simple_CompileTimeVariables</span>"),
				},
			},
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
				w3errs.Write(err, os.Stderr, outputMode)
				tt.FailNow()
				return
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
