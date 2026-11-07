package w3html

import (
	"fmt"
	"slices"
	"strings"
)

// TagEnum can be used to define custom tags by appending to [RegisteredTags].
type TagEnum struct {
	// Allowed chars: /[a-z_0-9]+/.
	// Namespace is empty for built-in w3 tags
	Namespace string
	// Id is a unique /[a-z_]+/ identifier
	Id string
	// IsPaired indicates whether tag has a closing counterpart
	IsPaired bool
	// Handler gets called by the [Parser] when the tag is encountered.
	//
	// For [TagEnum.IsPaired] it is always called on the closing tag.
	Handler func(parser *Parser, tag Tag) error
}

func (enum TagEnum) String() string {
	if enum.Namespace == "" {
		return enum.Id
	}
	return enum.Namespace + ":" + enum.Id
}

func (enum TagEnum) Equal(other TagEnum) bool {
	return enum.Namespace == other.Namespace && enum.Id == other.Id
}

var IdSlot = TagEnum{IsPaired: true, Id: "slot", Handler: nil}

var RegisteredTags = []TagEnum{
	{IsPaired: true, Id: "extend", Handler: nil},
	{IsPaired: true, Id: "if", Handler: nil},
	IdSlot,
	{IsPaired: true, Id: "range", Handler: nil},
	{IsPaired: false, Id: "include", Handler: nil},
	{IsPaired: false, Id: "place", Handler: nil},
	{IsPaired: false, Id: "", Handler: nil},
}

func ResolveTag(namespace, id string) (TagEnum, bool) {
	query := TagEnum{Namespace: namespace, Id: id}
	for _, enum := range RegisteredTags {
		if enum.Equal(query) {
			return enum, true
		}
	}
	return query, false
}

type Tag struct {
	Kind      TagEnum
	Arguments []Argument
}

func (tag Tag) String() string {
	var builder strings.Builder
	builder.WriteString("{{")
	if len(tag.Kind.Id) != 0 {
		builder.WriteRune(' ')
		builder.WriteString(tag.Kind.String())
	}
	for _, arg := range tag.Arguments {
		builder.WriteRune(' ')
		arg.Write(&builder)
	}
	builder.WriteString(" }}")
	return builder.String()
}

func (tag Tag) FindArg(name string, index uint) (Argument, bool) {
	if len(tag.Arguments) == 0 {
		return Argument{}, false
	}

	for _, arg := range tag.Arguments {
		if arg.Key == name {
			return arg, true
		}
	}

	lastIndex := uint(len(tag.Arguments) - 1)
	return tag.Arguments[min(index, lastIndex)], true
}

func (tag Tag) FindArgStrict(kind ArgumentEnum, name string, index uint) (Argument, error) {
	arg, ok := tag.FindArg(name, index)
	if !ok {
		return arg, fmt.Errorf(
			"%q tag requires %q argument of type %q",
			tag.Kind.String(),
			name,
			kind,
		)
	}
	if arg.Kind != kind {
		return arg, fmt.Errorf(
			"%q tag requires %q argument to be %q, but got %q instead",
			tag.Kind.String(),
			name,
			kind,
			arg.Kind,
		)
	}
	return arg, nil
}

func (tag Tag) ToParams(names ...string) map[string]string {
	out := map[string]string{}
	for _, arg := range tag.Arguments {
		if arg.Key != "" && !slices.Contains(names, arg.Key) {
			out[arg.Key] = arg.ToString()
		}
	}
	return out
}
