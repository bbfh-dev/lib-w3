package w3html_test

import (
	"fmt"
	"os"
	"testing"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"gotest.tools/assert"
)

func TestParse(t *testing.T) {
	file, err := os.OpenFile("../examples/99_complete/www/index.html", os.O_RDONLY, os.ModePerm)
	assert.NilError(t, err)
	defer file.Close()

	page, err := w3html.Parse(file)
	if err != nil {
		w3errs.Write(err, os.Stderr)
		t.FailNow()
	}

	for i, hunkKind := range page.Hunks {
		switch hunkKind {
		case w3html.HunkBlock:
			fmt.Printf("%#v\n", page.HunkData[i])
		case w3html.HunkRaw:
			fmt.Printf("%q\n", page.HunkData[i])
		}
	}
}
