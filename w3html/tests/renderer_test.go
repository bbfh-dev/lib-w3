package w3html_test

import (
	"bytes"
	"reflect"
	"testing"
	"time"

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
	{
		Name:     "Include",
		Contents: "<div>\n{{ include 'include.1' }}\n</div>",
		Expected: "<div>\n<span>\nHello World!\n</span></div>",
	},
}

func TestRender(t *testing.T) {
	w3utils.Logger.SetLevel(libclog.LogWarning)
	var buffer bytes.Buffer

	for _, testCase := range RenderTestCases {
		t.Run(testCase.Name, func(tt *testing.T) {
			buffer.Reset()
			buffer.Grow(len(testCase.Expected))

			file := &MockFile{
				Filename: testCase.Name + ".html",
				Contents: testCase.Contents,
			}
			file.Init()

			start := time.Now()
			parser := w3html.NewParser[PageData](file, map[string]string{})
			parser.WithFileOpener(MockFileOpener)

			result, err := parser.Parse()
			assert.NilError(t, err)
			parseDuration := time.Since(start)

			start = time.Now()
			renderer := w3html.Bake(result)
			bakeDuration := time.Since(start)

			start = time.Now()
			renderer.Render(w3html.RenderContext{
				Writer: &buffer,
				Data:   reflect.ValueOf(Data),
			})
			renderDuration := time.Since(start)
			assert.DeepEqual(t, testCase.Expected, buffer.String())

			tt.Logf(
				"%sParse: %-8s | Bake: %-8s | Render: %-8s | Saved: %-8s%s",
				libclog.AnsiColor(libclog.ColorBrightBlue),
				parseDuration,
				bakeDuration,
				renderDuration,
				parseDuration+bakeDuration-renderDuration,
				libclog.AnsiReset,
			)
		})
	}
}
