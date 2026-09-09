package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// listTestItems returns a stable tree: two leaves around one collapsed
// category and one expanded category with two leaves.
func listTestItems() []widgets.ListItem {
	return []widgets.ListItem{
		{ID: "fav", Label: "Favorites"},
		{ID: "weapons", Label: "Weapons", Children: []widgets.ListItem{
			{ID: "swords", Label: "Swords"},
		}},
		{ID: "mats", Label: "Materials", Expanded: true, Children: []widgets.ListItem{
			{ID: "herbs", Label: "Herbs"},
			{ID: "ess", Label: "Essences"},
		}},
	}
}

// newListUI returns a UI holding one populated list.
func newListUI(t *testing.T, bounds core.Rect) (*UI, *widgets.List) {
	t.Helper()
	u := New(400, 400)
	list := widgets.NewList("cats", bounds)
	list.SetItems(listTestItems())
	mustAdd(t, u, list)
	return u, list
}

// listRowCenter resolves the skin-aware center of one visible row. Rows lay
// out in the track-excluded area, matching draw and hit-test geometry.
func listRowCenter(u *UI, list *widgets.List, index int) core.Vec2 {
	content := u.listContentRect(list)
	rows := render.ListRowsContent(content, list.MaxScroll())
	row, _ := render.ListRowRect(rows, list.VisibleRowCount(), list.RowHeight(), list.ScrollOffset(), index)
	return core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
}

// listChevronCenter resolves the expander chevron center of one category row.
func listChevronCenter(u *UI, list *widgets.List, index int) core.Vec2 {
	content := u.listContentRect(list)
	rows := render.ListRowsContent(content, list.MaxScroll())
	row, _ := render.ListRowRect(rows, list.VisibleRowCount(), list.RowHeight(), list.ScrollOffset(), index)
	return core.Vec2{X: row.X + row.W - widgets.ListChevronWidth/2, Y: row.Y + row.H/2}
}

// TestListTogglesCategoryThroughPressRelease checks physical expansion.
func TestListTogglesCategoryThroughPressRelease(t *testing.T) {
	u, list := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 300})
	var toggles []string
	clicks := 0
	u.OnListToggle("cats", func(id string, expanded bool) { toggles = append(toggles, id) })
	u.OnClick("cats", func() { clicks++ })
	before := list.VisibleRowCount()
	clickAt(u, listRowCenter(u, list, 1)) // weapons: collapsed category
	if !list.IsExpanded("weapons") || list.VisibleRowCount() != before+1 {
		t.Fatal("category click did not expand")
	}
	if len(toggles) != 1 || clicks != 1 {
		t.Fatalf("callbacks = %v/%d", toggles, clicks)
	}
	if _, ok := list.Selected(); ok {
		t.Fatal("category toggle must never select")
	}
	clickAt(u, listRowCenter(u, list, 1))
	if list.IsExpanded("weapons") || len(toggles) != 2 || clicks != 2 {
		t.Fatalf("second click = %v/%v/%d", list.IsExpanded("weapons"), toggles, clicks)
	}
}

// TestListSelectsLeafThroughPressRelease checks physical selection.
func TestListSelectsLeafThroughPressRelease(t *testing.T) {
	u, list := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 300})
	var selected []string
	clicks := 0
	u.OnListSelect("cats", func(id string) { selected = append(selected, id) })
	u.OnClick("cats", func() { clicks++ })
	clickAt(u, listRowCenter(u, list, 4)) // ess leaf
	id, _ := list.Selected()
	if id != "ess" || len(selected) != 1 || clicks != 1 {
		t.Fatalf("leaf click = %q/%v/%d", id, selected, clicks)
	}
	clickAt(u, listRowCenter(u, list, 4))
	if len(selected) != 1 || clicks != 2 {
		t.Fatalf("same-leaf click fired select: %v/%d", selected, clicks)
	}
	if list.IsExpanded("ess") {
		t.Fatal("leaf must never report expanded")
	}
}

// TestListSemanticsBypassPointer checks SelectListItem and SetListExpanded.
func TestListSemanticsBypassPointer(t *testing.T) {
	u, list := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 300})
	var selected []string
	var toggles []string
	u.OnListSelect("cats", func(id string) { selected = append(selected, id) })
	u.OnListToggle("cats", func(id string, expanded bool) { toggles = append(toggles, id) })
	if !u.SelectListItem("cats", "herbs") || len(selected) != 1 {
		sel, _ := list.Selected()
		t.Fatalf("semantic select = %q/%v", sel, selected)
	}
	if u.SelectListItem("cats", "mats") || u.SelectListItem("cats", "missing") ||
		u.SelectListItem("cats", "swords") || u.SelectListItem("missing", "herbs") {
		t.Fatal("category, unknown, hidden, or missing select accepted")
	}
	if !u.SetListExpanded("cats", "weapons", true) || len(toggles) != 1 {
		t.Fatal("semantic expand rejected")
	}
	if u.SetListExpanded("cats", "herbs", true) || u.SetListExpanded("cats", "missing", true) {
		t.Fatal("leaf or unknown expand accepted")
	}
	if !u.Activate("cats") {
		t.Fatal("activate with selection rejected")
	}
	list.ClearSelection()
	if u.Activate("cats") {
		t.Fatal("activate without selection accepted")
	}
	list.SetEnabled(false)
	if u.SelectListItem("cats", "herbs") || u.SetListExpanded("cats", "weapons", false) {
		t.Fatal("disabled list accepted semantics")
	}
}

// TestListBlocksWhenDisabled checks pointer gestures honour availability.
func TestListBlocksWhenDisabled(t *testing.T) {
	u, list := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 300})
	toggles := 0
	selects := 0
	u.OnListToggle("cats", func(string, bool) { toggles++ })
	u.OnListSelect("cats", func(string) { selects++ })
	list.SetEnabled(false)
	clickAt(u, listRowCenter(u, list, 1))
	clickAt(u, listRowCenter(u, list, 4))
	if toggles != 0 || selects != 0 || list.IsExpanded("weapons") {
		t.Fatal("disabled list mutated")
	}
	if _, ok := list.Selected(); ok {
		t.Fatal("disabled list selected")
	}
}

// TestListWheelScrollsOverflow checks wheel ownership on long lists.
func TestListWheelScrollsOverflow(t *testing.T) {
	u, list := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 100})
	if list.MaxScroll() <= 0 {
		t.Fatal("fixture must overflow its viewport")
	}
	mid := core.Vec2{X: 30, Y: 60}
	if !u.HandleMouse(MouseEvent{Pos: mid, Wheel: -1}) || list.ScrollOffset() <= 0 {
		t.Fatalf("wheel did not scroll: %v", list.ScrollOffset())
	}
	before := list.ScrollOffset()
	if !u.HandleMouse(MouseEvent{Pos: mid, Wheel: 1}) || list.ScrollOffset() >= before {
		t.Fatal("reverse wheel did not scroll back")
	}
}

// TestListScrollbarDragsThumb checks track jump and thumb drag gestures.
func TestListScrollbarDragsThumb(t *testing.T) {
	u, list := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 100})
	content := u.listContentRect(list)
	track, ok := render.ScrollTrackRect(content, list.MaxScroll())
	if !ok {
		t.Fatal("overflowing list must expose a track")
	}
	thumb := render.ScrollThumbRect(track, list.ScrollOffset(), list.MaxScroll())
	// Track jump below the thumb scrolls down without selecting.
	jump := core.Vec2{X: track.X + track.W/2, Y: track.Y + track.H - 4}
	if !u.HandleMouse(MouseEvent{Pos: jump, Pressed: true}) {
		t.Fatal("track press not consumed")
	}
	if list.ScrollOffset() <= 0 {
		t.Fatal("track press did not jump")
	}
	if _, ok := list.Selected(); ok {
		t.Fatal("scrollbar press must never select")
	}
	u.HandleMouse(MouseEvent{Pos: jump, Released: true})
	// Thumb drag advances the offset and releases cleanly.
	list.SetScrollOffset(0)
	thumb = render.ScrollThumbRect(track, list.ScrollOffset(), list.MaxScroll())
	center := core.Vec2{X: thumb.X + thumb.W/2, Y: thumb.Y + thumb.H/2}
	if !u.HandleMouse(MouseEvent{Pos: center, Pressed: true}) || u.scrollThumbDragging != widgets.Widget(list) {
		t.Fatal("thumb press did not arm a drag")
	}
	dragged := core.Vec2{X: center.X, Y: center.Y + 20}
	if !u.HandleMouse(MouseEvent{Pos: dragged, Down: true}) || list.ScrollOffset() <= 0 {
		t.Fatal("thumb drag did not scroll")
	}
	if !u.HandleMouse(MouseEvent{Pos: dragged, Released: true}) || u.scrollThumbDragging != nil {
		t.Fatal("thumb release did not settle")
	}
	// The pointer rests on the thumb, so hover feedback remains but the
	// press must be gone.
	if state, err := u.ScrollThumbState("cats"); err != nil || state != core.StateHovered {
		t.Fatalf("thumb state = %v/%v", state, err)
	}
}

// TestListThumbStateReports checks shared scrollbar thumb states.
func TestListThumbStateReports(t *testing.T) {
	u, list := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 100})
	content := u.listContentRect(list)
	track, _ := render.ScrollTrackRect(content, list.MaxScroll())
	thumb := render.ScrollThumbRect(track, list.ScrollOffset(), list.MaxScroll())
	center := core.Vec2{X: thumb.X + thumb.W/2, Y: thumb.Y + thumb.H/2}
	u.HandleMouse(MouseEvent{Pos: center})
	if state, _ := u.ScrollThumbState("cats"); state != core.StateHovered {
		t.Fatalf("thumb hover state = %v", state)
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 390, Y: 390}})
	if state, _ := u.ScrollThumbState("cats"); state != core.StateNormal {
		t.Fatalf("thumb state after leave = %v", state)
	}
	if _, err := u.ScrollThumbState("missing"); err == nil {
		t.Fatal("missing thumb state accepted")
	}
	mustAdd(t, u, widgets.NewLabel("plain", core.Rect{}, "x"))
	if _, err := u.ScrollThumbState("plain"); err == nil {
		t.Fatal("non-scrollable thumb state accepted")
	}
}

// TestListChevronReachableWhenOverflowing checks the chevron regression:
// on an overflowing list the expander must sit left of the scrollbar track
// so a physical click on it toggles instead of missing.
func TestListChevronReachableWhenOverflowing(t *testing.T) {
	u, list := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 100})
	if list.MaxScroll() <= 0 {
		t.Fatal("fixture must overflow its viewport")
	}
	// weapons is the first visible category row.
	chevron := listChevronCenter(u, list, 1)
	content := u.listContentRect(list)
	track, ok := render.ScrollTrackRect(content, list.MaxScroll())
	if !ok || track.Contains(chevron) {
		t.Fatal("chevron must sit outside the scrollbar track")
	}
	toggles := 0
	u.OnListToggle("cats", func(string, bool) { toggles++ })
	clickAt(u, chevron)
	if !list.IsExpanded("weapons") || toggles != 1 {
		t.Fatal("chevron click did not toggle the category")
	}
}

// TestListDrawsKindCalls checks the draw path emits list shell and rows.
func TestListDrawsKindCalls(t *testing.T) {
	u, _ := newListUI(t, core.Rect{X: 20, Y: 20, W: 220, H: 300})
	recorder := attachDrawRecorder(t, u)
	u.Draw()
	calls := recorder.Calls()
	if !hasKindPart(calls, core.WidgetList, skin.PartBackground) {
		t.Fatal("list drew no shell")
	}
	if !hasKindPart(calls, core.WidgetList, skin.PartText) {
		t.Fatal("list drew no row text")
	}
	if !hasKindPart(calls, core.WidgetList, skin.PartArrow) {
		t.Fatal("list drew no chevrons")
	}
}
