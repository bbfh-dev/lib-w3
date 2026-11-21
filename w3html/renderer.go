package w3html

import (
	"io"
	"reflect"

	"codeberg.org/bbfh/lib-w3/w3data"
	"codeberg.org/bbfh/lib-w3/w3utils"
)

type Renderer []Renderable

func (renderer Renderer) Render(ctx RenderContext) (int, error) {
	written := 0
	for _, step := range renderer {
		n, err := step.Render(ctx)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

type Renderable interface {
	Render(ctx RenderContext) (written int, err error)
}

type RenderContext struct {
	Writer     io.Writer
	GlobalData reflect.Value
	LocalData  reflect.Value
	Scope      w3data.Path
}

func (context RenderContext) Query(path w3data.Path) []reflect.Value {
	if len(context.Scope) == 0 {
		return w3data.Walk(context.GlobalData, path)
	}
	return w3data.Walk(context.LocalData, path.TrimScope(context.Scope))
}

type HunkRenderStep []byte

func (step HunkRenderStep) Render(ctx RenderContext) (int, error) {
	return ctx.Writer.Write(step)
}

func Bake(page []Node) Renderable {
	pipeline := Renderer{}
	for _, node := range page {
		switch node.Kind {

		case NodeHunk:
			if len(pipeline) != 0 {
				if step, ok := pipeline[len(pipeline)-1].(HunkRenderStep); ok {
					pipeline[len(pipeline)-1] = append(step, node.AsHunk()...)
					continue
				}
			}
			pipeline = append(pipeline, HunkRenderStep(node.AsHunk()))

		case NodeTag:
			tag := node.AsTag()
			if tag.Kind.RendererFactory == nil {
				continue
			}
			renderable := tag.Kind.RendererFactory(&node, tag)
			if renderable == nil {
				continue
			}
			pipeline = append(pipeline, renderable)

		default:
			w3utils.Logger.Warn(
				"Skipping %q because it is not renderable: %#v",
				node.Kind,
				node.Data,
			)
		}
	}

	if len(pipeline) == 1 {
		// no value in boxing the value
		return pipeline[0]
	}
	return pipeline
}
