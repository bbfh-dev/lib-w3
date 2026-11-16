package w3html_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3data"
	"codeberg.org/bbfh/lib-w3/w3html"
	"codeberg.org/bbfh/lib-w3/w3utils"
	"gotest.tools/assert"
)

var Data = PageData{
	Path: PathData{
		To: "here.html",
	},
	My: MyData{
		Scope: ScopeData{
			Nested: "<span>Hi!</span>",
		},
	},
	Root: 69,
	Buttons: []Hashable{
		Hashable("first"),
		Hashable("second"),
		Hashable("third"),
	},
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
		Expected: "<div>\n<span>\nHello World!\n</span>\n</div>",
	},
	{
		Name:     "IncludeNestedPath",
		Contents: "<div>\n{{ include 'include.3' .My.Scope }}\n</div>",
		Expected: "<div>\n69<span>Hi!</span>\n</div>",
	},
	{
		Name:     "Extend",
		Contents: "{{ extend 'extend.1' }}Hello World!{{ /extend }}",
		Expected: "<section>\nHello World!\n</section>",
	},
	{
		Name:     "Range",
		Contents: "{{ range .Buttons[] }}Hello {{ .. }}!<br>{{ /range }}",
		Expected: "Hello first!<br>Hello second!<br>Hello third!<br>",
	},
	{
		Name:     "RangeMethod",
		Contents: "{{ range .Buttons[] }}Hello {{ ..Hash() }}!<br>{{ /range }}",
		Expected: "Hello 929733088!<br>Hello 695742261!<br>Hello 3406644!<br>",
	},
}

func TestRender(t *testing.T) {
	printResult := os.Getenv("DEBUG") == "1"
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
			renderer := w3html.Bake(result.Nodes)
			bakeDuration := time.Since(start)

			start = time.Now()
			value := reflect.ValueOf(Data)
			renderer.Render(w3html.RenderContext{
				Writer:     &buffer,
				GlobalData: value,
				LocalData:  value,
				Scope:      w3data.Path{},
			})
			renderDuration := time.Since(start)
			assert.DeepEqual(t, testCase.Expected, buffer.String())

			renderSteps := 1
			if renderer, ok := renderer.(w3html.Renderer); ok {
				renderSteps = len(renderer)
			}

			tt.Logf(
				"%sParse: %-8s | Bake: %-8s | Render: %-8s in %d step(s)%s",
				libclog.AnsiColor(libclog.ColorBrightBlue),
				parseDuration,
				bakeDuration,
				renderDuration,
				renderSteps,
				libclog.AnsiReset,
			)
			if printResult {
				fmt.Printf(
					"%s```html\n%s%s%s\n```\n%s",
					libclog.AnsiColor(libclog.ColorBrightYellow),
					libclog.AnsiColor(libclog.ColorBrightBlack),
					buffer.String(),
					libclog.AnsiColor(libclog.ColorBrightYellow),
					libclog.AnsiReset,
				)
			}
		})
	}
}
