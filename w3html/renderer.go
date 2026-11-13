package w3html

import (
	"io"
	"reflect"
)

type Renderer []Renderable

func (renderer Renderer) Render(ctx RenderContext) {
	for _, step := range renderer {
		step.Render(ctx)
	}
}

type Renderable interface {
	Render(ctx RenderContext)
}

type RenderContext struct {
	Writer io.Writer
	Data   reflect.Value
}

type HunkRenderStep []byte

func (step HunkRenderStep) Render(ctx RenderContext) {
	ctx.Writer.Write(step)
}

func Bake(page ParsedPage) Renderable {
	pipeline := Renderer{}
	for _, node := range page.Nodes {
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

		default:
		}
	}

	if len(pipeline) == 1 {
		// no value in boxing the value
		return pipeline[0]
	}
	return pipeline
}
