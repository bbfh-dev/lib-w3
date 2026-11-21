package w3html_test

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3data"
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

var Files = map[string]string{
	"include.1":   "<span>\n\tHello World!\n</span>",
	"include.2":   "{{ include 'include.2' }}",
	"include.3":   "{{ .Root }}{{ include 'include.4' ..Nested }}",
	"include.4":   "{{ ..This() }}",
	"extend.1":    "<section>\n\t{{ place 'default' }}\n</section>",
	"favicon.svg": "<svg></svg>",
}

var FS = MockFS{
	StaticFiles: Files,
}

var TestCases = []TestCase{
	{
		File: MockFile{
			Filename: "AccessPath_CompileTime",
			Contents: `<span>Hello World! from {{ ._filename.base }}</span>`,
		},
		Expected: w3html.ParsedPage{
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("<span>Hello World! from AccessPath_CompileTime</span>"),
					SourceCodeLine: 1,
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
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("{ this should stay as text }"),
					SourceCodeLine: 1,
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
								Data: w3data.Path{
									{
										Word:    "Path",
										Index:   0,
										Call:    false,
										Iterate: false,
									},
									{
										Word:    "To",
										Index:   0,
										Call:    false,
										Iterate: false,
									},
									{
										Word:    "Value",
										Index:   2,
										Call:    true,
										Iterate: false,
									},
								},
							},
						},
					},
					SourceCodeLine: 1,
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
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("<div>\n<span>\nHello World!\n</span>\n</div>"),
					SourceCodeLine: 1,
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
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("<div>\n"),
					SourceCodeLine: 1,
				},
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdVariable,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Key:  "",
								Data: w3data.Path{
									{
										Word:    "Root",
										Index:   2,
										Call:    false,
										Iterate: false,
									},
								},
							},
						},
					},
					SourceCodeLine: 1,
				},
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdVariable,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Key:  "",
								Data: w3data.Path{
									{
										Word:    "My",
										Index:   1,
										Call:    false,
										Iterate: false,
									},
									{
										Word:    "Scope",
										Index:   0,
										Call:    false,
										Iterate: false,
									},
									{
										Word:    "Nested",
										Index:   0,
										Call:    false,
										Iterate: false,
									},
									{
										Word:    "This",
										Index:   1,
										Call:    true,
										Iterate: false,
									},
								},
							},
						},
					},
					SourceCodeLine: 1,
				},
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("\n</div>"),
					SourceCodeLine: 2,
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
			PlacedSlots: map[string]w3html.NodeLocation{
				"default": {ChildIndex: 1, Line: 2},
			},
			Nodes: []w3html.Node{
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("<div>\n\n</div>"),
					SourceCodeLine: 1,
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Extend",
			Contents: "{{ extend 'extend.1' }}Hello World!{{ /extend }}",
		},
		Expected: w3html.ParsedPage{
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("<section>\nHello World!\n</section>"),
					SourceCodeLine: 1,
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Extend_Slot",
			Contents: "{{ extend 'extend.1' }}{{ slot 'default' }}Hello World!{{ /slot }}{{ /extend }}",
		},
		Expected: w3html.ParsedPage{
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("<section>\nHello World!\n</section>"),
					SourceCodeLine: 1,
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Range",
			Contents: "{{ range .Buttons[] }}Hello {{ .. }}!<br>{{ /range }}",
		},
		Expected: w3html.ParsedPage{
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdRange,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Key:  "",
								Data: w3data.Path{
									{
										Word:    "Buttons",
										Index:   3,
										Call:    false,
										Iterate: true,
									},
								},
							},
						},
					},
					SourceCodeLine: 1,
					Children: []w3html.Node{
						{
							Kind:           w3html.NodeHunk,
							Data:           []byte("Hello "),
							SourceCodeLine: 1,
						},
						{
							Kind: w3html.NodeTag,
							Data: w3html.Tag{
								Kind: w3html.IdVariable,
								Arguments: []w3html.Argument{
									{
										Kind: w3html.ArgPath,
										Key:  "",
										Data: w3data.Path{
											{
												Word:    "Buttons",
												Index:   3,
												Call:    false,
												Iterate: true,
											},
										},
									},
								},
							},
							SourceCodeLine: 1,
						},
						{
							Kind:           w3html.NodeHunk,
							Data:           []byte("!<br>"),
							SourceCodeLine: 1,
						},
					},
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "If",
			Contents: "{{ if .Root > 60 }}Hello World!{{ /if }}",
		},
		Expected: w3html.ParsedPage{
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdIf,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Key:  "",
								Data: w3data.Path{
									{
										Word:    "Root",
										Index:   2,
										Call:    false,
										Iterate: false,
									},
								},
							},
							{
								Kind: w3html.ArgOperation,
								Key:  "",
								Data: w3html.OpMoreThanStrict,
							},
							{
								Kind: w3html.ArgInt,
								Key:  "",
								Data: 60,
							},
						},
					},
					SourceCodeLine: 1,
					Children: []w3html.Node{
						{
							Kind:           w3html.NodeHunk,
							Data:           []byte("Hello World!"),
							SourceCodeLine: 1,
							Children:       nil,
						},
					},
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Complex",
			Contents: `
				{{ extend "extend.1" title="Hello World!" }}
				{{ slot "default" }}
				<main>
					{{ range .Buttons[] }}
					<span>{{ .. }}</span>
					{{ /range }}
				</main>
				{{ /slot }}
				{{ /extend }}`,
		},
		Expected: w3html.ParsedPage{
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("<section>\n\n<main>\n"),
					SourceCodeLine: 1,
				},
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdRange,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Key:  "",
								Data: w3data.Path{
									{
										Word:    "Buttons",
										Index:   3,
										Call:    false,
										Iterate: true,
									},
								},
							},
						},
					},
					SourceCodeLine: 5,
					Children: []w3html.Node{
						{
							Kind:           w3html.NodeHunk,
							Data:           []byte("\n<span>"),
							SourceCodeLine: 5,
						},
						{
							Kind: w3html.NodeTag,
							Data: w3html.Tag{
								Kind: w3html.IdVariable,
								Arguments: []w3html.Argument{
									{
										Kind: w3html.ArgPath,
										Key:  "",
										Data: w3data.Path{
											{
												Word:    "Buttons",
												Index:   3,
												Call:    false,
												Iterate: true,
											},
										},
									},
								},
							},
							SourceCodeLine: 6,
						},
						{
							Kind:           w3html.NodeHunk,
							Data:           []byte("</span>\n"),
							SourceCodeLine: 6,
						},
					},
				},
				{
					Kind:           w3html.NodeHunk,
					Data:           []byte("\n</main>\n\n</section>"),
					SourceCodeLine: 7,
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "FormatString",
			Contents: "{{ if .Root != `Hello {Root}` }}{{ /if }}",
		},
		Expected: w3html.ParsedPage{
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeTag,
					Data: w3html.Tag{
						Kind: w3html.IdIf,
						Arguments: []w3html.Argument{
							{
								Kind: w3html.ArgPath,
								Data: w3data.Path{{
									Word:  "Root",
									Index: 2,
								}},
							},
							{
								Kind: w3html.ArgOperation,
								Data: w3html.OpNotEquals,
							},
							{
								Kind: w3html.ArgFormatString,
								Data: w3html.FormatString{
									{
										Kind: w3html.FragmentString,
										Data: "Hello ",
									},
									{
										Kind: w3html.FragmentPath,
										Data: w3data.Path{{
											Word:  "Root",
											Index: 2,
										}},
									},
								},
							},
						},
					},
					SourceCodeLine: 1,
				},
			},
		},
	},
	{
		File: MockFile{
			Filename: "Static",
			Contents: "<link href='{{ static 'favicon.svg' }}'>",
		},
		Expected: w3html.ParsedPage{
			PlacedSlots: map[string]w3html.NodeLocation{},
			Nodes: []w3html.Node{
				{
					Kind: w3html.NodeHunk,
					Data: []byte(
						"<link href='/static/favicon.svg?checksum=7b56e1eab00ec8000da9331a4888cb35'>",
					),
					SourceCodeLine: 1,
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
	{
		File: MockFile{
			Filename: "Fail_PairedTagNotClosed",
			Contents: "{{ if 10 == 10 }}",
		},
		ExpectError: func(err w3errs.DocumentError) bool {
			return strings.Contains(
				err.Message,
				"paired tag \"if\" was never closed",
			)
		},
	},
}

type MockFS struct {
	StaticFiles map[string]string
}

func (fsys MockFS) Open(filename string) (fs.File, error) {
	contents, ok := fsys.StaticFiles[filename]
	if !ok {
		return nil, fmt.Errorf("%q not declared in the test file", filename)
	}
	file := &MockFile{
		Filename: filename,
		Contents: contents,
	}
	file.Init()
	return file, nil
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

func (file *MockFile) Stat() (fs.FileInfo, error) {
	return file, nil
}

func (file *MockFile) Read(dest []byte) (int, error) {
	return file.reader.Read(dest)
}

func (file *MockFile) Size() int64 {
	return int64(len(file.Contents))
}

func (file *MockFile) Mode() fs.FileMode {
	return fs.ModePerm
}

func (file *MockFile) ModTime() time.Time {
	return time.Now()
}

func (file *MockFile) IsDir() bool {
	return false
}

func (file *MockFile) Sys() any {
	return nil
}

type PageData struct {
	Path    PathData
	My      MyData
	Root    uint
	Buttons []Hashable
}

type Hashable string

func (str Hashable) Hash() int {
	hash := sha1.Sum([]byte(str))
	return int(binary.LittleEndian.Uint32(hash[:]))
}

func (data PageData) This() string {
	return "HI"
}

type PathData struct {
	To ToData
}

type ToData string

func (data ToData) Value() string {
	return string(data)
}

func (data ToData) This() string {
	return data.Value()
}

func (data ToData) String() string {
	return data.Value()
}

type MyData struct {
	Scope ScopeData
}

type ScopeData struct {
	Nested ToData
}

func MockFileOpener(absPath string) (fs.File, error) {
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
			parser := w3html.NewParser[PageData](&testCase.File, map[string]string{}, FS)
			parser.WithFileOpener(MockFileOpener)

			result, err := parser.Parse()
			if err != nil {
				if printResult {
					fmt.Fprintf(
						tt.Output(),
						"%s%s%s\n",
						libclog.AnsiColor(libclog.ColorBrightBlack),
						err.Error(),
						libclog.AnsiReset,
					)
				}
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
							return
						}
					} else {
						tt.Fatalf("error is not a document error: %s", err.Error())
					}
				}
			}

			if printResult {
				for _, node := range result.Nodes {
					node.Write(tt.Output(), 0)
				}
			}

			// Spent an hour debugging, no fucking idea why I'm forced to do this.
			// I've tried:
			// — cmpopts.IgnoreFields(...)
			// — cmpopts.IgnoreUnexported(...)
			// — cmp.Comparer(...)
			// — cmp.Diff()
			// any time [w3data.Path] is a nested type it tries to compare [w3data.Segment.dataType]. Stupid.
			for _, node := range result.Nodes {
				if node.Kind == w3html.NodeTag {
					data := node.AsTag()
					for _, arg := range data.Arguments {
						if arg.Kind == w3html.ArgPath {
							path := arg.AsPath()
							for i := range path {
								path[i].ClearParserMetadata()
							}
						}
					}
				}
			}

			assert.DeepEqual(
				tt,
				testCase.Expected,
				result,
				cmpopts.IgnoreFields(w3html.NodeLocation{}, "Parent", "ReaderOffset"),
			)
		})
	}
}
