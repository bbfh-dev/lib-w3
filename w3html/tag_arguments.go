package w3html

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strings"

	"codeberg.org/bbfh/lib-w3/w3data"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

// read-only
type ArgumentEnum uint8

const (
	ArgString ArgumentEnum = iota
	ArgFormatString
	ArgOperation
	ArgLogicOperation
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
	case ArgLogicOperation:
		return "LogicOperation"
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
	case ArgInt, ArgFloat, ArgBool, ArgPath:
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

func (arg Argument) Compute(ctx RenderContext) any {
	switch arg.Kind {
	case ArgPath:
		return ctx.Query(arg.AsPath())[0].Interface()
	default:
		return arg.Data
	}
}

type Stringable interface {
	String() string
}

func ToBoolean(data any) bool {
	switch data := data.(type) {
	case string:
		return len(data) != 0
	case FormatString:
		return len(data) != 0
	case Stringable:
		return len(data.String()) != 0
	case int8, int16, int32, int64, int, uint, uint16, uint32, uint64:
		return data != 0
	case float32, float64:
		return data != 0.0
	case bool:
		return data
	default:
		return false
	}
}

func ToNumber(data any) float64 {
	switch data := data.(type) {
	case float64:
		return data
	case int:
		return float64(data)
	case uint:
		return float64(data)
	case bool:
		if data {
			return 1.0
		}
		return 0.0
	default:
		return 0.0
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

func (arg Argument) AsLogicOperation() LogicOperation {
	return arg.Data.(LogicOperation)
}

func (arg Argument) Equal(other Argument) bool {
	if arg.Kind != other.Kind {
		return false
	}
	switch arg.Kind {
	case ArgFormatString:
		// TODO: Use EqualFunc()
		return slices.Equal(arg.AsFormatString(), other.AsFormatString())
	case ArgOperation:
		return arg.AsOperation().Token == other.AsOperation().Token
	case ArgLogicOperation:
		return arg.AsLogicOperation().Token == other.AsLogicOperation().Token
	case ArgPath:
		return slices.EqualFunc(arg.AsPath(), other.AsPath(), w3data.Segment.Equal)
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

type Operation struct {
	Token      string
	Comparator func(a, b Argument) Predicate
	Validator  func(a, b Argument) error
}

func (op Operation) String() string {
	return op.Token
}

var (
	OpEquals = Operation{
		Token:     "==",
		Validator: nil,
		Comparator: func(a, b Argument) Predicate {
			return func(ctx RenderContext) bool {
				return w3utils.LooseEqual(
					a.Compute(ctx),
					b.Compute(ctx),
				)
			}
		},
	}
	OpNotEquals = Operation{
		Token:     "!=",
		Validator: nil,
		Comparator: func(a, b Argument) Predicate {
			return func(ctx RenderContext) bool {
				return !w3utils.LooseEqual(
					a.Compute(ctx),
					b.Compute(ctx),
				)
			}
		},
	}
	OpMoreThanStrict = Operation{
		Token:     ">",
		Validator: argBothAreNumbers,
		Comparator: func(a, b Argument) Predicate {
			return func(ctx RenderContext) bool {
				return ToNumber(a.Compute(ctx)) > ToNumber(b.Compute(ctx))
			}
		},
	}
	OpMoreThan = Operation{
		Token:     ">=",
		Validator: argBothAreNumbers,
		Comparator: func(a, b Argument) Predicate {
			return func(ctx RenderContext) bool {
				return ToNumber(a.Compute(ctx)) >= ToNumber(b.Compute(ctx))
			}
		},
	}
	OpLessThanStrict = Operation{
		Token:     "<",
		Validator: argBothAreNumbers,
		Comparator: func(a, b Argument) Predicate {
			return func(ctx RenderContext) bool {
				return ToNumber(a.Compute(ctx)) < ToNumber(b.Compute(ctx))
			}
		},
	}
	OpLessThan = Operation{
		Token:     "<=",
		Validator: argBothAreNumbers,
		Comparator: func(a, b Argument) Predicate {
			return func(ctx RenderContext) bool {
				return ToNumber(a.Compute(ctx)) <= ToNumber(b.Compute(ctx))
			}
		},
	}
)

// ————————————————————————————————

type LogicOperation struct {
	Token   string
	Compose func(a, b Predicate) Predicate
}

func (op LogicOperation) String() string {
	return op.Token
}

var (
	OpLogicAnd = LogicOperation{
		Token: "&&",
		Compose: func(a, b Predicate) Predicate {
			return func(ctx RenderContext) bool {
				return a(ctx) && b(ctx)
			}
		},
	}
	OpLogicOr = LogicOperation{
		Token: "||",
		Compose: func(a, b Predicate) Predicate {
			return func(ctx RenderContext) bool {
				return a(ctx) || b(ctx)
			}
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
