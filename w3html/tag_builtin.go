package w3html

import (
	"fmt"

	"codeberg.org/bbfh/lib-w3/w3data"
)

var IdSlot = TagEnum{
	IsPaired:     true,
	KeepChildren: false,
	Id:           "slot",
	Arguments:    NoArguments,
	Handler:      nil,
}

var IdVariable = TagEnum{
	IsPaired:     false,
	KeepChildren: false,
	Id:           "",
	Arguments: []ArgumentDefinition{
		{Name: ArgNamePath, Kind: ArgPath, IsOptional: false},
		{Name: "join", Kind: ArgString, IsOptional: true},
	},
	Handler: func(parser ParserApi, node *Node, tag Tag) error {
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
			parser.AppendHunk([]byte(value))
		} else {
			parser.AppendTag(tag)
		}
		return nil
	},
	RendererFactory: func(node *Node, tag Tag) Renderable {
		arg, err := tag.FindArgNamed(ArgNamePath)
		if err != nil {
			return nil
		}
		path := arg.AsPath()

		var join = ", "
		arg, err = tag.FindArgNamed("join")
		if err == nil {
			join = arg.AsString()
		}
		return VariableRenderStep{
			Path:       path,
			JoinString: join,
		}
	},
}

type VariableRenderStep struct {
	Path       w3data.Path
	JoinString string
}

func (step VariableRenderStep) Render(ctx RenderContext) {
	values := ctx.Query(step.Path)
	for i, value := range values {
		if i != 0 {
			ctx.Writer.Write([]byte(step.JoinString))
		}
		fmt.Fprint(ctx.Writer, value.Interface())
	}
}

var IdRange = TagEnum{
	IsPaired:     true,
	KeepChildren: true,
	Id:           "range",
	Arguments: []ArgumentDefinition{
		{Name: ArgNameScope, Kind: ArgPath, IsOptional: false},
	},
	Handler: func(parser ParserApi, node *Node, tag Tag) error {
		path := tag.Arguments[0].AsPath()
		if !path[len(path)-1].Iterate {
			return fmt.Errorf("'range' tag requires that the 'path' argument ends with '[]'")
		}
		return nil
	},
	RendererFactory: func(node *Node, tag Tag) Renderable {
		arg, _ := tag.FindArg(0)
		path := arg.AsPath()
		return RangeRenderStep{
			Path:     path,
			Contents: Bake(node.Children),
		}
	},
}

type RangeRenderStep struct {
	Path     w3data.Path
	Contents Renderable
}

func (step RangeRenderStep) Render(ctx RenderContext) {
	values := ctx.Query(step.Path)
	for _, value := range values {
		ctx.LocalData = value
		ctx.Scope = step.Path
		step.Contents.Render(ctx)
	}
}

var IdIf = TagEnum{
	IsPaired:     true,
	KeepChildren: true,
	Id:           "if",
	Arguments:    SkipArgumentCheck,
	Handler:      nil,
	RendererFactory: func(node *Node, tag Tag) Renderable {
		return IfRenderStep{
			Predicate: ParsePredicate(tag.Arguments),
			Contents:  Bake(node.Children),
		}
	},
}

type IfRenderStep struct {
	Predicate Predicate
	Contents  Renderable
}

func (step IfRenderStep) Render(ctx RenderContext) {
	if step.Predicate(ctx) {
		step.Contents.Render(ctx)
	}
}

var RegisteredTags = []TagEnum{}

func init() {
	RegisteredTags = []TagEnum{
		IdSlot,
		IdRange,
		IdIf,
		{
			IsPaired:     true,
			KeepChildren: false,
			Id:           "extend",
			Arguments: []ArgumentDefinition{
				{Name: ArgNamePath, Kind: ArgString, IsOptional: false},
				{Name: ArgNameScope, Kind: ArgPath, IsOptional: true},
			},
			Handler: func(parser ParserApi, node *Node, tag Tag) error {
				slots, err := node.ExtractInjectSlots()
				if err != nil {
					return err
				}

				nestedPage, err := parser.ParseNested(tag, slots)
				if err != nil {
					return err
				}

				node.Kind = NodeRoot
				node.Data = nil
				node.Children = nestedPage.Nodes
				return nil
			},
		},
		{
			IsPaired:     false,
			KeepChildren: false,
			Id:           "include",
			Arguments: []ArgumentDefinition{
				{Name: ArgNamePath, Kind: ArgString, IsOptional: false},
				{Name: ArgNameScope, Kind: ArgPath, IsOptional: true},
			},
			Handler: func(parser ParserApi, node *Node, tag Tag) error {
				nestedPage, err := parser.ParseNested(tag, SlotsMap{})
				if err != nil {
					return err
				}
				for _, child := range nestedPage.Nodes {
					parser.Append(child)
				}
				return nil
			},
		},
		{
			IsPaired:     false,
			KeepChildren: false,
			Id:           "place",
			Arguments: []ArgumentDefinition{
				{Name: ArgNameSlotId, Kind: ArgString, IsOptional: false},
			},
			Handler: func(parser ParserApi, node *Node, tag Tag) error {
				slotIdArg, err := tag.FindArgNamed(ArgNameSlotId)
				if err != nil {
					return err
				}
				slotId := slotIdArg.AsString()
				parser.AppendPlaceholder()
				parser.SaveSlot(slotId, parser.LastAddedNode())
				return nil
			},
		},
		IdVariable,
	}
}
