package w3html

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// TagEnum can be used to define custom tags by appending to [RegisteredTags].
type TagEnum struct {
	// Allowed chars: /[a-z_0-9]+/.
	// Namespace is empty for built-in w3 tags
	Namespace string
	// Id is a unique /[a-z_0-9]+/ identifier
	Id string
	// IsPaired indicates whether tag has a closing counterpart
	IsPaired bool
	// Handler gets called by the [Parser] when the tag is encountered.
	//
	// For [TagEnum.IsPaired] it is always called on the closing tag.
	Handler func(parser *Parser, tag Tag) error
	// Arguments defines [Argument] that will be validated.
	//
	// Set to 'nil' (or [SkipArgumentCheck]) to skip argument validation.
	//
	// If this [Tag] defines a new [AccessPath] variable scope, provide an argument named [ArgNameScope].
	Arguments []ArgumentDefinition
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

func (enum TagEnum) DefinesArg(name string) bool {
	for _, arg := range enum.Arguments {
		if arg.Name == name {
			return true
		}
	}
	return false
}

var IdSlot = TagEnum{
	IsPaired:  true,
	Id:        "slot",
	Arguments: NoArguments,
	Handler:   nil,
}

var IdVariable = TagEnum{
	IsPaired: false,
	Id:       "",
	Arguments: []ArgumentDefinition{
		{Name: ArgNamePath, Kind: ArgPath, IsOptional: false},
	},
	Handler: func(parser *Parser, tag Tag) error {
		arg, err := tag.FindArgNamed(ArgNamePath)
		if err != nil {
			return err
		}
		path := arg.AsPath()
		if path.IsCompileTime() {
			value, err := parser.Param(path.Flatten())
			if err != nil {
				return err
			}
			parser.appendHunk([]byte(value))
		} else {
			parser.appendTag(tag)
		}
		return nil
	},
}

var RegisteredTags = []TagEnum{}

func init() {
	RegisteredTags = []TagEnum{
		IdSlot,
		{
			IsPaired: true,
			Id:       "extend",
			Arguments: []ArgumentDefinition{
				{Name: ArgNamePath, Kind: ArgString, IsOptional: false},
				{Name: ArgNameScope, Kind: ArgPath, IsOptional: true},
			},
			Handler: nil,
		},
		{
			IsPaired:  true,
			Id:        "if",
			Arguments: SkipArgumentCheck,
			Handler:   nil,
		},
		{
			IsPaired: true,
			Id:       "range",
			Arguments: []ArgumentDefinition{
				{Name: ArgNameScope, Kind: ArgPath, IsOptional: false},
			},
			Handler: nil,
		},
		{
			IsPaired: false,
			Id:       "include",
			Arguments: []ArgumentDefinition{
				{Name: ArgNamePath, Kind: ArgString, IsOptional: false},
				{Name: ArgNameScope, Kind: ArgPath, IsOptional: true},
			},
			Handler: func(parser *Parser, tag Tag) error {
				nestedPage, err := parser.ParseNested(tag)
				if err != nil {
					return err
				}
				for _, child := range nestedPage.Nodes {
					parser.ctx.Append(TreeNode{
						Node:           child,
						SourceCodeLine: 1,
						Children:       nil,
					})
				}
				return nil
			},
		},
		{
			IsPaired: false,
			Id:       "place",
			Arguments: []ArgumentDefinition{
				{Name: ArgNameSlotId, Kind: ArgString, IsOptional: false},
			},
			Handler: nil,
		},
		IdVariable,
	}
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

func (tag Tag) FindArg(index uint) (Argument, bool) {
	if len(tag.Arguments) == 0 {
		return Argument{}, false
	}
	required := tag.Kind.Arguments[index]

	for _, arg := range tag.Arguments {
		if arg.Key == required.Name {
			return arg, true
		}
	}

	lastIndex := uint(len(tag.Arguments) - 1)
	if lastIndex < index {
		return Argument{}, false
	}
	return tag.Arguments[index], true
}

func (tag Tag) FindArgNamed(name string) (Argument, error) {
	var index uint = math.MaxUint8
	for i, arg := range tag.Kind.Arguments {
		if arg.Name == name {
			index = uint(i)
			break
		}
	}
	if index == math.MaxUint8 {
		return Argument{}, fmt.Errorf(
			"couldn't find argument %q in definition of tag %q",
			name,
			tag.Kind,
		)
	}
	return tag.FindArgStrict(index)
}

func (tag Tag) FindArgStrict(index uint) (Argument, error) {
	required := tag.Kind.Arguments[index]
	arg, ok := tag.FindArg(index)
	if !ok {
		return arg, fmt.Errorf(
			"%q tag requires %q argument of type %q",
			tag.Kind.String(),
			required.Name,
			required.Kind,
		)
	}
	var predicate bool
	switch required.Kind {
	case ArgMatchAny:
		predicate = true
	case ArgMatchAnyString:
		predicate = arg.Kind == ArgString || arg.Kind == ArgFormatString
	default:
		predicate = arg.Kind == required.Kind
	}
	if !predicate {
		return arg, fmt.Errorf(
			"%q tag requires %q argument to be %q, but got %q instead",
			tag.Kind.String(),
			required.Name,
			required.Kind,
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
