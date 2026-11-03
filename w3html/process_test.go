package w3html_test

import (
	"fmt"
	"os"
	"testing"

	libclog "codeberg.org/bbfh/lib-clog"
	"codeberg.org/bbfh/lib-w3/w3errs"
	"codeberg.org/bbfh/lib-w3/w3html"
	"gotest.tools/assert"
)

func TestProcess(t *testing.T) {
	outputMode := libclog.OutputAuto
	if os.Getenv("COLOR") == "1" {
		outputMode = libclog.OutputColored
	}

	file, err := os.OpenFile("parse_test.01.html", os.O_RDONLY, os.ModePerm)
	assert.NilError(t, err)
	defer file.Close()

	page, err := w3html.Parse(file)
	if err != nil {
		w3errs.Write(err, os.Stderr, outputMode)
		t.FailNow()
	}

	proc := w3html.NewProcessor(&page, map[string]string{})
	if err := proc.Process(); err != nil {
		w3errs.Write(err, os.Stderr, outputMode)
		t.FailNow()
	}

	for _, hunk := range proc.Out.Hunks {
		fmt.Printf("%s", hunk.Data)
	}
}
