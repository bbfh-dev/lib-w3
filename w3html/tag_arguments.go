package w3html

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strings"

	"codeberg.org/bbfh/lib-w3/w3data"
)

// read-only
type ArgumentEnum uint8

const (
	ArgString ArgumentEnum = iota
	ArgFormatString
	ArgOperation
	ArgInt
	ArgFloat
	ArgBool
	ArgPath
	ArgMatchAnyString = math.MaxUint8 - 1
	ArgMatchAny       = math.MaxUint8
)

func (enum ArgumentEnum) String() string {
	switch enum {
	case ArgString:
		return "String"
	case ArgFormatString:
		return "FormatString"
	case ArgOperation:
		return "Operation"
	case ArgInt:
		return "Integer"
	case ArgFloat:
		return "Float"
	case ArgBool:
		return "Boolean"
	case ArgPath:
		return "Path"
	case ArgMatchAnyString:
		return "String/FormatString"
	case ArgMatchAny:
		return "*"
	default:
		return fmt.Sprintf("ArgumentEnum(%d)", enum)
	}
}

// IsNumeric returns whether the argument type can be meaningfully converted into a number
func (enum ArgumentEnum) IsNumeric() bool {
	switch enum {
	case ArgInt, ArgFloat, ArgBool:
		return true
	default:
		return false
	}
}

// ————————————————————————————————

const (
	ArgNamePath   = "path"
	ArgNameScope  = "scope"
	ArgNameSlotId = "slot_id"
)

var NoArguments = []ArgumentDefinition{}
var SkipArgumentCheck []ArgumentDefinition = nil

type ArgumentDefinition struct {
	Name string
	Kind ArgumentEnum
	// Important: all optional arguments must be at the end to prevent mistakes.
	IsOptional bool
}

// ————————————————————————————————

// read-only
type Argument struct {
	Kind ArgumentEnum
	Key  string
	Data any
}

func (arg Argument) String() string {
	var builder strings.Builder
	arg.Write(&builder)
	return builder.String()
}

func (arg Argument) Write(w io.Writer) {
	if arg.Key != "" {
		fmt.Fprintf(w, "%s=", arg.Key)
	}
	switch arg.Kind {
	case ArgInt:
		fmt.Fprintf(w, "%d", arg.Data)
	case ArgFloat:
		fmt.Fprintf(w, "%f", arg.Data)
	case ArgString:
		fmt.Fprintf(w, "%q", arg.Data)
	case ArgFormatString:
		fmt.Fprintf(w, "`%s`", arg.Data)
	default:
		fmt.Fprintf(w, "%s", arg.Data)
	}
}

func (arg Argument) ToString() string {
	switch arg.Kind {
	case ArgInt:
		return fmt.Sprintf("%d", arg.Data)
	case ArgFloat:
		return fmt.Sprintf("%f", arg.Data)
	default:
		return fmt.Sprintf("%s", arg.Data)
	}
}

func (arg Argument) ToNumber() float64 {
	switch arg.Kind {
	case ArgFloat:
		return arg.AsFloat()
	case ArgInt:
		return float64(arg.AsInt())
	case ArgBool:
		if arg.AsBool() {
			return 1.0
		}
		return 0.0
	default:
		return 0.0
	}
}

func (arg Argument) ToBoolean() bool {
	switch arg.Kind {
	case ArgString:
		return len(arg.AsString()) != 0
	case ArgFormatString:
		return len(arg.AsFormatString()) != 0
	case ArgInt:
		return arg.AsInt() != 0
	case ArgFloat:
		return arg.AsFloat() != 0.0
	case ArgBool:
		return arg.AsBool()
	case ArgPath:
		return len(arg.AsPath()) != 0
	default:
		return false
	}
}

func (arg Argument) AsString() string {
	return arg.Data.(string)
}

func (arg Argument) AsFormatString() FormatString {
	return arg.Data.(FormatString)
}

func (arg Argument) AsInt() int {
	return arg.Data.(int)
}

func (arg Argument) AsFloat() float64 {
	return arg.Data.(float64)
}

func (arg Argument) AsBool() bool {
	return arg.Data.(bool)
}

func (arg Argument) AsPath() w3data.Path {
	return arg.Data.(w3data.Path)
}

func (arg Argument) AsOperation() Operation {
	return arg.Data.(Operation)
}

func (arg Argument) Equal(other Argument) bool {
	if arg.Kind != other.Kind {
		return false
	}
	switch arg.Kind {
	case ArgFormatString:
		return slices.Equal(arg.AsFormatString(), other.AsFormatString())
	case ArgOperation:
		return arg.AsOperation().Token == other.AsOperation().Token
	case ArgPath:
		return slices.Equal(arg.AsPath(), other.AsPath())
	default:
		return arg.Data == other.Data
	}
}

// ————————————————————————————————

// read-only
type FormatString []formatFragment

func (str FormatString) String() string {
	var builder strings.Builder
	for _, fragment := range str {
		builder.WriteString(fragment.String())
	}
	return builder.String()
}

type formatFragmentEnum uint8

const (
	FragmentString formatFragmentEnum = iota
	FragmentPath
)

type formatFragment struct {
	Kind formatFragmentEnum
	Data any
}

func (fragment formatFragment) String() string {
	switch fragment.Kind {
	case FragmentString:
		return fragment.Data.(string)
	case FragmentPath:
		return "{" + fragment.Data.(w3data.Path).Flatten() + "}"
	default:
		return "{<?>}"
	}
}

// ————————————————————————————————

type Path []PathSegment

func (path Path) String() string {
	if len(path) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, segment := range path {
		builder.WriteRune('.')
		builder.WriteString(string(segment.Word))
	}
	return builder.String()
}

type SegmentEnum uint8

const (
	SegField SegmentEnum = iota
	SegMethod
	SegArray
)

type PathSegment struct {
	Kind  SegmentEnum
	Word  string
	Index uint
}

// ————————————————————————————————

type Operation struct {
	Token      string
	Comparator func(a, b Argument) bool
	Validator  func(a, b Argument) error
}

func (op Operation) String() string {
	return op.Token
}

var (
	OpEquals = Operation{
		Token:      "==",
		Validator:  nil,
		Comparator: Argument.Equal,
	}
	OpNotEquals = Operation{
		Token:     "!=",
		Validator: nil,
		Comparator: func(a, b Argument) bool {
			return !a.Equal(b)
		},
	}
	OpMoreThanStrict = Operation{
		Token:     ">",
		Validator: argBothAreNumbers,
		Comparator: func(a, b Argument) bool {
			return a.ToNumber() > b.ToNumber()
		},
	}
	OpMoreThan = Operation{
		Token:     ">=",
		Validator: argBothAreNumbers,
		Comparator: func(a, b Argument) bool {
			return a.ToNumber() >= b.ToNumber()
		},
	}
	OpLessThanStrict = Operation{
		Token:     "<",
		Validator: argBothAreNumbers,
		Comparator: func(a, b Argument) bool {
			return a.ToNumber() < b.ToNumber()
		},
	}
	OpLessThan = Operation{
		Token:     "<=",
		Validator: argBothAreNumbers,
		Comparator: func(a, b Argument) bool {
			return a.ToNumber() <= b.ToNumber()
		},
	}
	OpLogicAnd = Operation{
		Token:     "&&",
		Validator: nil,
		Comparator: func(a, b Argument) bool {
			return a.ToBoolean() && b.ToBoolean()
		},
	}
	OpLogicOr = Operation{
		Token:     "||",
		Validator: nil,
		Comparator: func(a, b Argument) bool {
			return a.ToBoolean() || b.ToBoolean()
		},
	}
)

func argBothAreNumbers(a, b Argument) error {
	if !a.Kind.IsNumeric() {
		return fmt.Errorf("left side expression is not a number")
	}
	if !b.Kind.IsNumeric() {
		return fmt.Errorf("right side expression is not a number")
	}
	return nil
}
