package w3html

import (
	"fmt"
	"slices"
	"strings"
)

type Tag struct {
	Name string
	Args []Argument
}

func (tag Tag) IsPaired() bool {
	switch tag.Name {
	case "extend", "if", "slot":
		return true
	default:
		return false
	}
}

func (tag Tag) IsPath() bool {
	return tag.Name == "" &&
		len(tag.Args) == 1 &&
		tag.Args[0].Kind == ArgPath
}

func (tag Tag) FindNamedArg(name string, fallbackIndex int) *Argument {
	if len(tag.Args) == 0 {
		return nil
	}

	for _, arg := range tag.Args {
		if arg.Key == name {
			return &arg
		}
	}

	lastIndex := len(tag.Args) - 1
	return &tag.Args[min(fallbackIndex, lastIndex)]
}

func (tag Tag) RequireNamedArg(
	requiredKind ArgumentEnum,
	name string,
	fallbackIndex int,
) (*Argument, error) {
	arg := tag.FindNamedArg("path", 0)
	if arg == nil {
		return nil, fmt.Errorf(
			"%q tag requires a %q argument of type %q",
			tag.Name,
			name,
			requiredKind,
		)
	}
	if arg.Kind != requiredKind {
		return nil, fmt.Errorf(
			"%q tag requires a %q argument to be a %q, but got %q instead",
			tag.Name,
			name,
			requiredKind,
			arg.Kind,
		)
	}
	return arg, nil
}

func (tag Tag) StringArgsExcluding(names ...string) map[string]string {
	out := map[string]string{}
	for _, arg := range tag.Args {
		if !slices.Contains(names, arg.Key) {
			out[arg.Key] = arg.ToString()
		}
	}
	return out
}

func (tag Tag) String() string {
	var builder strings.Builder
	builder.WriteString("{{")
	if len(tag.Name) != 0 {
		builder.WriteRune(' ')
		builder.WriteString(tag.Name)
	}
	for _, arg := range tag.Args {
		builder.WriteRune(' ')
		arg.Write(&builder)
	}
	builder.WriteString(" }}")
	return builder.String()
}
