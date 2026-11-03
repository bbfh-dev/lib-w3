package w3html

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
)

type SlotLocation struct {
	Parent *Hunk
	Index  uint
}

type ParsedPage struct {
	Slots map[string]SlotLocation
	Root  Hunk
	// temporary, stores current hunk parents during parsing
	breadcrumbs []*Hunk
	last        *Hunk
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
	page.breadcrumbs = append(page.breadcrumbs, page.last)
}

func (page *ParsedPage) OnLeaveTagBody(tag Tag) error {
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

func (page *ParsedPage) AppendAndRemember(hunk Hunk) {
	page.Append(hunk)
	page.last = &page.Pivot().Children[len(page.Pivot().Children)-1]
}

func (page *ParsedPage) Place(slot string, replacement ...Hunk) error {
	location, ok := page.Slots[slot]
	if !ok {
		return fmt.Errorf(
			"trying to inject into an undefined slot named %q. Defined slots are: [%s]",
			slot,
			strings.Join(slices.Collect(maps.Keys(page.Slots)), ", "),
		)
	}

	// we subtract 1 from capacity, because one of the children is the placeholder.
	children := make([]Hunk, 0, len(location.Parent.Children)+len(replacement)-1)
	children = append(children, location.Parent.Children[:location.Index]...)
	children = append(children, replacement...)
	children = append(children, location.Parent.Children[location.Index+1:]...)
	location.Parent.Children = children
	// TODO: Add merge logic from [ParsedPage.Append]
	return nil
}

func (page *ParsedPage) String() string {
	var builder strings.Builder
	page.Write(&builder)
	return builder.String()
}

func (page *ParsedPage) Write(w io.Writer) {
	fmt.Fprintln(w, strings.Repeat("—", 80))
	page.Root.Write(w, 0)
	fmt.Fprint(w, strings.Repeat("—", 80))
}
