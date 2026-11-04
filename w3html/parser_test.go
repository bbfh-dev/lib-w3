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

var expected = w3html.ParsedPage{
	Slots: map[string]w3html.HunkLocation{
		"default": {Parent: nil, ChildIndex: 1},
	},
	Root: w3html.Hunk{
		Kind:     w3html.HunkRoot,
		Line:     1,
		Data:     nil,
		Children: []w3html.Hunk{},
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

	page, err := w3html.Parse(nil, file, true, map[string]string{})
	if err != nil {
		w3errs.Write(err, os.Stderr, outputMode)
		t.FailNow()
	}

	fmt.Println(page.String())

	assert.DeepEqual(
		t,
		expected,
		page,
		cmpopts.IgnoreUnexported(w3html.ParsedPage{}),
		cmpopts.IgnoreFields(w3html.HunkLocation{}, "Parent"),
	)
}
