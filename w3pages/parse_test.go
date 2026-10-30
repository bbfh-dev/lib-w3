package w3pages_test

import (
	"fmt"
	"os"
	"testing"

	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3pages"
	"gotest.tools/assert"
)

func TestParse(t *testing.T) {
	file, err := os.OpenFile("../examples/99_complete/www/index.html", os.O_RDONLY, os.ModePerm)
	assert.NilError(t, err)
	defer file.Close()

	page, err := w3pages.Parse(file)
	if err != nil {
		w3errs.Write(err, os.Stderr)
		t.FailNow()
	}

	for i, hunkKind := range page.Hunks {
		switch hunkKind {
		case w3pages.HunkBlock:
			fmt.Printf("%#v\n", page.HunkData[i])
		case w3pages.HunkRaw:
			fmt.Printf("%q\n", page.HunkData[i])
		}
	}
}
