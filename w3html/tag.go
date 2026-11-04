package w3html

import (
	"slices"
	"strconv"
	"strings"
)

type Tag struct {
	Name string
	Args []Argument
}

type Argument struct {
	Key, Value string
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
		tag.Args[0].Key == "" &&
		strings.HasPrefix(tag.Args[0].Value, ".")
}

func (tag Tag) FindNamedArg(name string, i int) string {
	if len(tag.Args) == 0 {
		return ""
	}

	for _, arg := range tag.Args {
		if arg.Key == name {
			return arg.Value
		}
	}

	iLast := len(tag.Args) - 1
	return tag.Args[min(i, iLast)].Value
}

func (tag Tag) ArgsExcluding(names ...string) map[string]string {
	out := map[string]string{}
	for _, arg := range tag.Args {
		if !slices.Contains(names, arg.Key) {
			out[arg.Key] = arg.Value
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
		if arg.Key != "" {
			builder.WriteString(arg.Key)
			builder.WriteRune('=')
		}
		builder.WriteRune('"')
		quoted := strconv.Quote(arg.Value)
		quoted = quoted[1 : len(quoted)-1]
		builder.WriteString(quoted)
		builder.WriteRune('"')
	}
	builder.WriteString(" }}")
	return builder.String()
}
