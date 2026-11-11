package w3data_test

import (
	"os"
	"reflect"
	"testing"

	"codeberg.org/bbfh/lib-w3/w3data"
	"gotest.tools/assert"
)

type A struct {
	B int
	C []string
}

func (a A) D() []E {
	return []E{
		{Id: "first", F: true},
		{Id: "second", F: false},
	}
}

type E struct {
	Embeded
	F bool
}

func (e E) G() bool {
	return !e.F
}

type Embeded struct {
	Id string
}

var Instance = A{
	B: 0xf006a,
	C: []string{
		"first",
		"second",
		"third",
	},
}

// ————————————————————————————————

type WalkTestCase struct {
	Path     w3data.Path
	Expected []any
}

var WalkTestCases = []WalkTestCase{
	{
		Path: w3data.Path{
			{
				Word:    "B",
				Index:   0,
				Call:    false,
				Iterate: false,
			},
		},
		Expected: []any{0xf006a},
	},
	{
		Path: w3data.Path{
			{
				Word:    "C",
				Index:   1,
				Call:    false,
				Iterate: false,
			},
		},
		Expected: []any{[]string{"first", "second", "third"}},
	},
	{
		Path: w3data.Path{
			{
				Word:    "D",
				Index:   0,
				Call:    true,
				Iterate: false,
			},
		},
		Expected: []any{[]E{{Id: "first", F: true}, {Id: "second", F: false}}},
	},
	{
		Path: w3data.Path{
			{
				Word:    "D",
				Index:   0,
				Call:    true,
				Iterate: true,
			},
		},
		Expected: []any{E{Id: "first", F: true}, E{Id: "second", F: false}},
	},
	{
		Path: w3data.Path{
			{
				Word:    "D",
				Index:   0,
				Call:    true,
				Iterate: true,
			},
			{
				Word:    "F",
				Index:   1,
				Call:    false,
				Iterate: false,
			},
		},
		Expected: []any{true, false},
	},
	{
		Path: w3data.Path{
			{
				Word:    "D",
				Index:   0,
				Call:    true,
				Iterate: true,
			},
			{
				Word:    "G",
				Index:   0,
				Call:    true,
				Iterate: false,
			},
		},
		Expected: []any{false, true},
	},
	{
		Path: w3data.Path{
			{
				Word:    "D",
				Index:   0,
				Call:    true,
				Iterate: true,
			},
			{
				Word:    "Value",
				Index:   0,
				Call:    false,
				Iterate: false,
			},
			{
				Word:    "Id",
				Index:   0,
				Call:    false,
				Iterate: false,
			},
		},
		Expected: []any{"first", "second"},
	},
}

func TestWalk(t *testing.T) {
	debug := os.Getenv("DEBUG") == "1"
	for _, testCase := range WalkTestCases {
		t.Run("A"+testCase.Path.String(), func(tt *testing.T) {
			returned := w3data.Walk(reflect.ValueOf(Instance), testCase.Path)
			if len(returned) != len(testCase.Expected) {
				for _, value := range testCase.Expected {
					tt.Logf("- %#v", value)
				}
				for _, value := range returned {
					tt.Logf("+ %#v", value)
				}
				tt.Fatalf(
					"returned %d elements, but expected %d",
					len(returned),
					len(testCase.Expected),
				)
				return
			}
			for i, value := range returned {
				expected := reflect.ValueOf(testCase.Expected[i])
				if expected.Kind() == reflect.Slice {
					tt.Log("ASSERT slice lengths")
					assert.Equal(tt, expected.Len(), value.Len())
					for j := range expected.Len() {
						if debug {
							tt.Logf(
								"ASSERT %v (got) == %v (expected)",
								value.Index(j),
								expected.Index(j),
							)
						}
						assert.Assert(tt, expected.Index(j).Equal(value.Index(j)))
					}
				} else {
					if debug {
						tt.Logf("ASSERT %v (got) == %v (expected)", value, expected)
					}
					assert.Assert(tt, expected.Equal(value))
				}
			}
		})
	}
}
