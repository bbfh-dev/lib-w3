package w3html

import (
	"fmt"
	"io"
	"strings"
)

type ArgumentEnum uint8

const (
	ArgString ArgumentEnum = iota
	ArgOperation
	ArgInt
	ArgFloat
	ArgPath
)

func (enum ArgumentEnum) String() string {
	switch enum {
	case ArgString:
		return "String"
	case ArgOperation:
		return "Operation"
	case ArgInt:
		return "Int"
	case ArgFloat:
		return "Float"
	case ArgPath:
		return "Path"
	default:
		return fmt.Sprintf("ArgumentEnum(%d)", enum)
	}
}

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
	case ArgPath:
		fmt.Fprintf(w, ".%s", arg.Data)
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

func (arg Argument) AsString() string {
	return arg.Data.(string)
}

func (arg Argument) AsInt() int {
	return arg.Data.(int)
}

func (arg Argument) AsFloat() float64 {
	return arg.Data.(float64)
}

type StringPath []string

func (path StringPath) String() string {
	return strings.Join(path, ".")
}

func (arg Argument) AsPath() StringPath {
	return arg.Data.(StringPath)
}

type Operation string

const (
	OpEquals         Operation = "=="
	OpNotEquals      Operation = "!="
	OpMoreThanStrict Operation = ">"
	OpMoreThan       Operation = ">="
	OpLessThanStrict Operation = "<"
	OpLessThan       Operation = "<="
	OpAnd            Operation = "&&"
	OpOr             Operation = "||"
)

func (arg Argument) AsOperation() Operation {
	return arg.Data.(Operation)
}
