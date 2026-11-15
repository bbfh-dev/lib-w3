package w3html

import (
	"fmt"

	"codeberg.org/bbfh/lib-w3/w3data"
)

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
		{Name: "join", Kind: ArgString, IsOptional: true},
	},
	Handler: func(parser ParserApi, node *TreeNode, tag Tag) error {
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
	RendererFactory: func(tag Tag) Renderable {
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
	values := w3data.Walk(ctx.Data, step.Path)
	for i, value := range values {
		if i != 0 {
			ctx.Writer.Write([]byte(step.JoinString))
		}
		fmt.Fprint(ctx.Writer, value.Interface())
	}
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
			Handler: func(parser ParserApi, node *TreeNode, tag Tag) error {
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
				node.Children = nil
				for _, child := range nestedPage.Nodes {
					node.Children = append(node.Children, TreeNode{
						Node:           child,
						SourceCodeLine: 1,
						Children:       nil,
					})
				}
				return nil
			},
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
			Handler: func(parser ParserApi, node *TreeNode, tag Tag) error {
				nestedPage, err := parser.ParseNested(tag, SlotsMap{})
				if err != nil {
					return err
				}
				for _, child := range nestedPage.Nodes {
					parser.Append(TreeNode{
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
			Handler: func(parser ParserApi, node *TreeNode, tag Tag) error {
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
