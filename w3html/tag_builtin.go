package w3html

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"

	"codeberg.org/bbfh/lib-w3/w3data"
)

var IdSlot = TagEnum{
	IsPaired:     true,
	KeepChildren: false,
	Id:           "slot",
	Arguments: []ArgumentDefinition{
		{Name: ArgNameSlotId, Kind: ArgString, IsOptional: false},
	},
	Handler: nil,
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

func (step VariableRenderStep) Render(ctx RenderContext) (int, error) {
	written := 0
	values := ctx.Query(step.Path)
	for i, value := range values {
		if i != 0 {
			n, err := ctx.Writer.Write([]byte(step.JoinString))
			written += n
			if err != nil {
				return written, err
			}
		}
		n, err := fmt.Fprint(ctx.Writer, value.Interface())
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
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

func (step RangeRenderStep) Render(ctx RenderContext) (int, error) {
	written := 0
	values := ctx.Query(step.Path)
	for _, value := range values {
		ctx.LocalData = value
		ctx.Scope = step.Path
		n, err := step.Contents.Render(ctx)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

var IdIf = TagEnum{
	IsPaired:     true,
	KeepChildren: true,
	Id:           "if",
	Arguments:    SkipArgumentCheck,
	Handler: func(parser ParserApi, node *Node, tag Tag) error {
		_, err := ParsePredicate(tag.Arguments)
		if err != nil {
			return err
		}
		return nil
	},
	RendererFactory: func(node *Node, tag Tag) Renderable {
		predicate, _ := ParsePredicate(tag.Arguments)
		return IfRenderStep{
			Predicate: predicate,
			Contents:  Bake(node.Children),
		}
	},
}

type IfRenderStep struct {
	Predicate Predicate
	Contents  Renderable
}

func (step IfRenderStep) Render(ctx RenderContext) (int, error) {
	written := 0
	if step.Predicate(ctx) {
		n, err := step.Contents.Render(ctx)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
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
		{
			IsPaired:     false,
			KeepChildren: false,
			Id:           "static",
			Handler: func(parser ParserApi, node *Node, tag Tag) error {
				arg, err := tag.FindArgNamed(ArgNamePath)
				if err != nil {
					return err
				}
				path := arg.AsString()

				fsys := parser.StaticFS()
				if fsys == nil {
					return fmt.Errorf(
						"static filesystem is not configured. Make sure the EngineBuilder.WithStaticConfig() is called before EngineBuilder.LoadPage()",
					)
				}

				file, err := fsys.Open(path)
				if err != nil {
					return err
				}

				data, err := io.ReadAll(file)
				if err != nil {
					return err
				}

				sum := md5.Sum(data)
				checksum := hex.EncodeToString(sum[:])
				parser.AppendHunk([]byte("/static/" + path + "?checksum=" + checksum))
				return nil
			},
			Arguments: []ArgumentDefinition{
				{
					Name:       ArgNamePath,
					Kind:       ArgString,
					IsOptional: false,
				},
			},
			RendererFactory: nil,
		},
	}
}
