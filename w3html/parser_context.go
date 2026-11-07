package w3html

import (
	"fmt"
	"io"
	"strings"
)

type ParserContext struct {
	Root               TreeNode
	breadcrumbs        []NodeLocation
	out                ParsedPage
	lastAddedNode      *TreeNode
	lastAddedNodeIndex uint
}

func (ctx ParserContext) String() string {
	var builder strings.Builder
	ctx.Write(&builder)
	return builder.String()
}

func (ctx ParserContext) Write(w io.Writer) {
	fmt.Fprintln(w, strings.Repeat("—", 80))
	ctx.Root.Write(w, 0)
	fmt.Fprint(w, strings.Repeat("—", 80))
}

func (ctx *ParserContext) Pivot() *TreeNode {
	if len(ctx.breadcrumbs) == 0 {
		return &ctx.Root
	} else {
		return ctx.breadcrumbs[len(ctx.breadcrumbs)-1].Pointer()
	}
}

func (ctx *ParserContext) OnEnterTagBody(mark uint) {
	ctx.breadcrumbs = append(ctx.breadcrumbs, NodeLocation{
		Parent:       ctx.Pivot(),
		ChildIndex:   ctx.lastAddedNodeIndex,
		Line:         ctx.lastAddedNode.SourceCodeLine,
		ReaderOffset: mark,
	})
}

func (ctx *ParserContext) OnLeaveTagBody(tag Tag) error {
	if len(ctx.breadcrumbs) == 0 {
		return fmt.Errorf(
			"found closing tag for %q, but no tags have been opened yet",
			tag.Kind,
		)
	}

	last := ctx.breadcrumbs[len(ctx.breadcrumbs)-1].Pointer().AsTag()
	if !last.Kind.Equal(tag.Kind) {
		return fmt.Errorf(
			"found closing tag for %q, but %q tag must be closed first!",
			tag.Kind,
			last.Kind,
		)
	}
	ctx.breadcrumbs = ctx.breadcrumbs[:len(ctx.breadcrumbs)-1]
	return nil
}

func (ctx *ParserContext) Append(hunk TreeNode) {
	ctx.append(hunk)
	pivot := ctx.Pivot()
	ctx.lastAddedNodeIndex = uint(len(pivot.Children) - 1)
	ctx.lastAddedNode = &pivot.Children[ctx.lastAddedNodeIndex]
}

func (ctx *ParserContext) append(hunk TreeNode) {
	pivot := ctx.Pivot()

	// merge [NodeHunk] if they are adjacent.
	if hunk.Kind == NodeHunk && len(pivot.Children) != 0 {
		last := &pivot.Children[len(pivot.Children)-1]
		if last.Kind == NodeHunk {
			last.Data = append(last.AsHunk(), hunk.AsHunk()...)
			return
		}
	}

	// for everything else
	pivot.Children = append(pivot.Children, hunk)
}
