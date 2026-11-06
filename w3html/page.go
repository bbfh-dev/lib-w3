package w3html

import (
	"fmt"
	"io"
	"strings"
)

type HunkLocation struct {
	Parent     *Hunk
	ChildIndex uint
	Line       uint
	ByteOffset uint
}

func (location HunkLocation) IsNull() bool {
	return location.Parent == nil
}

func (location HunkLocation) Hunk() *Hunk {
	return &location.Parent.Children[location.ChildIndex]
}

type ParsedPage struct {
	MentionedPaths []StringPath
	Slots          map[string]HunkLocation
	Root           Hunk
	// temporary, stores current hunk parents during parsing
	breadcrumbs []*Hunk
	lastHunk    *Hunk
	lastIndex   uint
}

func (page ParsedPage) Hunks() []InlineHunk {
	hunks := make([]InlineHunk, 0, len(page.Root.Children))
	var visit func(Hunk)
	visit = func(hunk Hunk) {
		hunks = append(hunks, InlineHunk{
			Kind: hunk.Kind,
			Data: hunk.Data,
		})

		for _, child := range hunk.Children {
			visit(child)
		}
	}
	for _, hunk := range page.Root.Children {
		visit(hunk)
	}
	return hunks
}

// Pivot returns the current [Hunk] that should be appended to.
func (page *ParsedPage) Pivot() *Hunk {
	if len(page.breadcrumbs) == 0 {
		return &page.Root
	} else {
		return page.breadcrumbs[len(page.breadcrumbs)-1]
	}
}

func (page *ParsedPage) OnEnterTagBody() {
	page.breadcrumbs = append(page.breadcrumbs, page.lastHunk)
}

func (page *ParsedPage) OnLeaveTagBody(tag Tag) error {
	if len(page.breadcrumbs) == 0 {
		return fmt.Errorf("found closing tag for %q, but no tags have been opened yet", tag.Name)
	}

	last := page.breadcrumbs[len(page.breadcrumbs)-1].AsTag()
	if last.Name != tag.Name {
		return fmt.Errorf(
			"found closing tag for %q, but %q tag must be closed first!",
			tag.Name,
			last.Name,
		)
	}
	page.breadcrumbs = page.breadcrumbs[:len(page.breadcrumbs)-1]
	return nil
}

func (page *ParsedPage) Append(hunk Hunk) {
	page.append(hunk)
	pivot := page.Pivot()
	page.lastIndex = uint(len(pivot.Children) - 1)
	page.lastHunk = &pivot.Children[page.lastIndex]
}

func (page *ParsedPage) append(hunk Hunk) {
	pivot := page.Pivot()

	// merge [HunkRaw] if they are adjacent.
	if hunk.Kind == HunkRaw && len(pivot.Children) != 0 {
		last := &pivot.Children[len(pivot.Children)-1]
		if last.Kind == HunkRaw {
			last.Data = append(last.AsRaw(), hunk.AsRaw()...)
			return
		}
	}

	// for everything else
	pivot.Children = append(pivot.Children, hunk)
}

func (page ParsedPage) String() string {
	var builder strings.Builder
	page.Write(&builder)
	return builder.String()
}

func (page ParsedPage) Write(w io.Writer) {
	fmt.Fprintln(w, strings.Repeat("—", 80))
	page.Root.Write(w, 0)
	fmt.Fprint(w, strings.Repeat("—", 80))
}
