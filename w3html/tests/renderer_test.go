package w3html_test

import (
	"bytes"
	"reflect"
	"testing"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3html"
	"codeberg.org/bbfh/lib-w3/w3utils"
	"gotest.tools/assert"
)

var Data = PageData{
	Path: PathData{},
	My:   MyData{},
	Root: 69,
}

type RenderTestCase struct {
	Name     string
	Contents string
	Expected string
}

var RenderTestCases = []RenderTestCase{
	{
		Name:     "HelloWorld",
		Contents: "Hello World!\n",
		Expected: "Hello World!\n",
	},
	{
		Name:     "CompileTimeVariable",
		Contents: "Hello {{ ._filename.base }}!",
		Expected: "Hello CompileTimeVariable.html!",
	},
	{
		Name:     "RootPath",
		Contents: "Hello {{ .Root }}!",
		Expected: "Hello 69!",
	},
}

func TestRender(t *testing.T) {
	w3utils.Logger.SetLevel(libclog.LogWarning)

	for _, testCase := range RenderTestCases {
		t.Run(testCase.Name, func(tt *testing.T) {
			file := &MockFile{
				Filename: testCase.Name + ".html",
				Contents: testCase.Contents,
			}
			file.Init()
			parser := w3html.NewParser[PageData](file, map[string]string{})
			parser.WithFileOpener(MockFileOpener)

			result, err := parser.Parse()
			assert.NilError(t, err)

			var buffer bytes.Buffer
			buffer.Grow(len(testCase.Expected))
			renderer := w3html.Bake(result)
			renderer.Render(w3html.RenderContext{
				Writer: &buffer,
				Data:   reflect.ValueOf(Data),
			})
			assert.DeepEqual(t, testCase.Expected, buffer.String())
		})
	}
}
