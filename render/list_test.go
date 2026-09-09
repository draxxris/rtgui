package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
	"github.com/draxxris/rtgui/widgets"
)

// listTestWidget returns a stable overflowing list for draw tests.
func listTestWidget() *widgets.List {
	list := widgets.NewList("cats", core.Rect{X: 10, Y: 20, W: 220, H: 100})
	list.SetItems([]widgets.ListItem{
		{ID: "fav", Label: "Favorites", Icon: "star"},
		{ID: "weapons", Label: "Weapons", Children: []widgets.ListItem{
			{ID: "swords", Label: "Swords"},
		}},
		{ID: "mats", Label: "Materials", Expanded: true, Children: []widgets.ListItem{
			{ID: "herbs", Label: "Herbs"},
			{ID: "ess", Label: "Essences"},
		}},
	})
	list.Select("herbs")
	return list
}

// TestListRowsTileContent verifies the shared row formula: rows tile
// content exactly, centers invert, scroll translates, and degenerate
// inputs miss.
func TestListRowsTileContent(t *testing.T) {
	content := core.Rect{X: 10, Y: 20, W: 200, H: 100}
	for index := 0; index < 3; index++ {
		row, ok := ListRowRect(content, 3, 28, 0, index)
		if !ok {
			t.Fatalf("row %d missing", index)
		}
		want := core.Rect{X: 10, Y: 20 + float32(index*28), W: 200, H: 28}
		if row != want {
			t.Fatalf("row %d = %+v, want %+v", index, row, want)
		}
		center := core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
		if got := ListRowAt(content, 3, 28, 0, center); got != index {
			t.Fatalf("row %d center maps to %d", index, got)
		}
	}
	// Scrolled rows shift up by the offset.
	row, _ := ListRowRect(content, 3, 28, 14, 1)
	if row.Y != 34 {
		t.Fatalf("scrolled row y = %v, want 34", row.Y)
	}
	center := core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
	if got := ListRowAt(content, 3, 28, 14, center); got != 1 {
		t.Fatalf("scrolled center maps to %d, want 1", got)
	}
	if _, ok := ListRowRect(content, 3, 28, 0, 3); ok {
		t.Fatal("out-of-range row must miss")
	}
	if _, ok := ListRowRect(content, 0, 28, 0, 0); ok {
		t.Fatal("empty list must have no rows")
	}
	if _, ok := ListRowRect(content, 3, 0, 0, 0); ok {
		t.Fatal("zero row height must miss")
	}
	for _, point := range []core.Vec2{{X: 0, Y: 0}, {X: 20, Y: 19.9}, {X: 20, Y: 120}, {X: 9.9, Y: 50}, {X: 210.1, Y: 50}} {
		if got := ListRowAt(content, 3, 28, 0, point); got != -1 {
			t.Fatalf("outside point %+v maps to %d", point, got)
		}
	}
	if got := ListRowAt(content, 0, 28, 0, core.Vec2{X: 20, Y: 50}); got != -1 {
		t.Fatalf("empty list index = %d", got)
	}
}

// TestListContentAppliesSkinInsets verifies hit testing and drawing share
// one skin-aware viewport.
func TestListContentAppliesSkinInsets(t *testing.T) {
	bounds := core.Rect{X: 10, Y: 20, W: 220, H: 100}
	var nilTheme *Theme
	if got := nilTheme.ListContent(bounds, core.StateNormal); got != bounds {
		t.Fatalf("nil theme content = %+v, want %+v", got, bounds)
	}
	theme := NewTheme(transform.New(core.Viewport{}))
	if got := theme.ListContent(bounds, core.StateNormal); got != bounds {
		t.Fatalf("unskinned content = %+v, want %+v", got, bounds)
	}
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetList, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		PaddingLeft: 4, PaddingTop: 4, PaddingRight: 4, PaddingBottom: 4,
	})
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetList, Part: skin.PartBorder, State: core.StateNormal}, skin.SkinDescriptor{
		HasNinePatch: true, NinePatch: skin.NinePatch{Left: 8, Top: 8, Right: 8, Bottom: 8},
	})
	content := theme.ListContent(bounds, core.StateNormal)
	want := core.Rect{X: 18, Y: 28, W: 204, H: 84}
	if content != want {
		t.Fatalf("skinned content = %+v, want %+v", content, want)
	}
}

// TestListScrollGeometry checks track visibility and thumb proportions.
func TestListScrollGeometry(t *testing.T) {
	content := core.Rect{X: 10, Y: 20, W: 200, H: 100}
	if _, ok := ScrollTrackRect(content, 0); ok {
		t.Fatal("fitting list must hide its track")
	}
	track, ok := ScrollTrackRect(content, 40)
	if !ok {
		t.Fatal("overflowing list must show its track")
	}
	want := core.Rect{X: 194, Y: 20, W: 16, H: 100}
	if track != want {
		t.Fatalf("track = %+v, want %+v", want, track)
	}
	top := ScrollThumbRect(track, 0, 40)
	bottom := ScrollThumbRect(track, 40, 40)
	if top.Y != track.Y || bottom.Y+bottom.H != track.Y+track.H {
		t.Fatalf("thumb travel = %+v/%+v in %+v", top, bottom, track)
	}
	if top.H < ScrollThumbMinHeight {
		t.Fatalf("thumb height = %v", top.H)
	}
	if got := ScrollThumbRect(track, 0, 0); got != (core.Rect{}) {
		t.Fatalf("still thumb = %+v", got)
	}
	first, last := ListVisibleRange(100, 5, 28, 0)
	if first != 0 || last != 3 {
		t.Fatalf("visible range = %d/%d, want 0/3", first, last)
	}
	first, last = ListVisibleRange(100, 5, 28, 40)
	if first != 1 || last != 4 {
		t.Fatalf("scrolled range = %d/%d, want 1/4", first, last)
	}
	if first, last := ListVisibleRange(0, 5, 28, 0); last >= first {
		t.Fatalf("empty viewport range = %d/%d", first, last)
	}
}

// TestListRowsContentExcludesTrack checks overflowing rows lay out left of
// the scrollbar so chevrons stay reachable: the last chevron center must
// resolve through the same content the rows draw with.
func TestListRowsContentExcludesTrack(t *testing.T) {
	content := core.Rect{X: 10, Y: 20, W: 200, H: 100}
	rows := ListRowsContent(content, 40)
	track, ok := ScrollTrackRect(content, 40)
	if !ok {
		t.Fatal("overflowing content must expose a track")
	}
	if rows.W != track.X-content.X {
		t.Fatalf("rows width = %v, want %v", rows.W, track.X-content.X)
	}
	if got := ListRowsContent(content, 0); got != content {
		t.Fatalf("fitting rows = %+v, want full content", got)
	}
	// Chevron center of the bottom visible row must hit that row, not the track.
	row, _ := ListRowRect(rows, 5, 28, 0, 3)
	chevron := core.Vec2{X: row.X + row.W - 14, Y: row.Y + row.H/2}
	if track.Contains(chevron) {
		t.Fatal("chevron center sits inside the scrollbar track")
	}
	if got := ListRowAt(rows, 5, 28, 0, chevron); got != 3 {
		t.Fatalf("chevron maps to row %d, want 3", got)
	}
}

// TestListDrawsSelectionAndChevron checks the recorded row calls: shell,
// selected highlight with gold accent, and one chevron per category.
func TestListDrawsSelectionAndChevron(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.RegisterInlineIcon("star", core.TooltipIcon{ID: 7, Width: 16, Height: 16})
	list := listTestWidget()
	recorder := newTestRecorder(t, 256)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	info := list.Snapshot(core.StateNormal)
	theme.DrawList(info, list, 0, -1, core.StateNormal)
	calls := recorder.Calls()
	hasPart := func(part skin.SkinPart) bool {
		for _, call := range calls {
			if call.Kind == core.WidgetList && call.Part == part {
				return true
			}
		}
		return false
	}
	if !hasPart(skin.PartBackground) || !hasPart(skin.PartText) {
		t.Fatal("list drew no shell or row text")
	}
	selected, arrows, icons := 0, 0, 0
	for _, call := range calls {
		if call.Kind != core.WidgetList {
			continue
		}
		if call.Part == skin.PartOverlay && call.State == core.StateSelected {
			selected++
		}
		if call.Part == skin.PartArrow {
			arrows++
		}
		if call.Part == skin.PartIcon {
			icons++
		}
	}
	// Selected highlight + accent + separator rows all log PartOverlay;
	// at least the selected fill and accent must appear.
	if selected < 2 {
		t.Fatalf("selected overlays = %d, want at least fill and accent", selected)
	}
	if arrows != 2 {
		t.Fatalf("chevrons = %d, want one per visible category", arrows)
	}
	if icons != 1 {
		t.Fatalf("icons = %d, want the single registered row icon", icons)
	}
}

// TestListDrawSkipsOffscreenRows verifies windowed drawing: scrolled-off
// rows record no text.
func TestListDrawSkipsOffscreenRows(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	list := listTestWidget()
	list.SetScrollOffset(list.MaxScroll())
	recorder := newTestRecorder(t, 256)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	theme.DrawList(list.Snapshot(core.StateNormal), list, -1, -1, core.StateNormal)
	texts := 0
	for _, call := range recorder.Calls() {
		if call.Kind == core.WidgetList && call.Part == skin.PartText {
			texts++
		}
	}
	if texts == 0 || texts >= list.VisibleRowCount() {
		t.Fatalf("visible texts = %d for %d rows", texts, list.VisibleRowCount())
	}
}
