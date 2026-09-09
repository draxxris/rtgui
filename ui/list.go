package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// listContentRect returns the skin-aware viewport inside the list shell.
func (u *UI) listContentRect(list *widgets.List) core.Rect {
	if list == nil {
		return core.Rect{}
	}
	if u == nil || u.theme == nil {
		return list.Bounds()
	}
	return u.theme.ListContent(list.Bounds(), u.visualState(list), list.Class())
}

// reconcileListBounds syncs the derived scroll limit with the true content
// height before hit testing, scrolling, and drawing agree on one viewport.
func (u *UI) reconcileListBounds(list *widgets.List) core.Rect {
	content := u.listContentRect(list)
	if list != nil {
		list.EnsureScrollBounds(content.H)
	}
	return content
}

// listIndexAt resolves the visible row under pos, or -1. Rows lay out in the
// track-excluded area, so scrollbar presses naturally miss.
func (u *UI) listIndexAt(list *widgets.List, pos core.Vec2) int {
	if list == nil {
		return -1
	}
	content := u.reconcileListBounds(list)
	rows := render.ListRowsContent(content, list.MaxScroll())
	return render.ListRowAt(rows, list.VisibleRowCount(), list.RowHeight(), list.ScrollOffset(), pos)
}

// listHoverIndex resolves the pointer row for drawing, or -1.
func (u *UI) listHoverIndex(list *widgets.List) int {
	if list == nil || u.hovered != list {
		return -1
	}
	return u.listIndexAt(list, u.pointer)
}

// listPressedIndex resolves the armed row for drawing, or -1.
func (u *UI) listPressedIndex(list *widgets.List) int {
	if list == nil || u.pressed != list {
		return -1
	}
	return u.listIndexAt(list, u.pointer)
}

// drawList renders one collapsible list with pointer-aware row states.
func (u *UI) drawList(list *widgets.List, state core.WidgetState) {
	info := list.Snapshot(state)
	u.reconcileListBounds(list)
	u.theme.DrawList(info, list, u.listHoverIndex(list), u.listPressedIndex(list), u.scrollThumbStateFor(list))
	if needsBorder(list.Kind()) {
		u.theme.DrawWidgetPart(list.Kind(), skin.PartBorder, list.Bounds(), state, list.Class())
	}
}

// releaseList commits the released row and closes the gesture. Category
// rows toggle expansion; leaf rows select. Releasing outside any row
// consumes without mutation.
func (u *UI) releaseList(list *widgets.List, pos core.Vec2) bool {
	index := u.listIndexAt(list, pos)
	if index < 0 {
		return true
	}
	row, ok := list.VisibleRowAt(index)
	if !ok {
		return true
	}
	if row.HasChildren {
		u.commitListToggle(list, row.ID, !row.Expanded)
		return true
	}
	u.commitListSelection(list, row.ID)
	return true
}

// commitListSelection records a leaf selection and fires callbacks.
// OnListSelect runs only after a real ID change; OnClick always runs.
func (u *UI) commitListSelection(list *widgets.List, id string) {
	if list.Select(id) {
		u.fireOnListSelect(list.Name(), id)
	}
	if u.Lookup(list.Name()) == list {
		u.fireOnClick(list.Name())
	}
}

// commitListToggle records a category expansion and fires callbacks.
// OnListToggle runs only after a real flag change; OnClick always runs.
func (u *UI) commitListToggle(list *widgets.List, id string, expanded bool) {
	if list.SetExpanded(id, expanded) {
		u.fireOnListToggle(list.Name(), id, expanded)
	}
	if u.Lookup(list.Name()) == list {
		u.fireOnClick(list.Name())
	}
}

// lookupListRow resolves a registered available list and one visible row.
// Hidden rows (inside collapsed categories) miss like pointer hit testing.
func (u *UI) lookupListRow(op, name, id string) (*widgets.List, widgets.ListRow, bool) {
	var zero widgets.ListRow
	target := u.Lookup(name)
	if target == nil {
		u.diagnose("ui.%s: widget %q not found", op, name)
		return nil, zero, false
	}
	list, ok := target.(*widgets.List)
	if !ok {
		u.diagnose("ui.%s: widget %q is %v, expected WidgetList", op, name, target.Kind())
		return nil, zero, false
	}
	if !u.available(list) {
		u.diagnose("ui.%s: widget %q is disabled", op, name)
		return nil, zero, false
	}
	row, ok := list.VisibleRowAt(list.IndexOfRow(id))
	if !ok {
		u.diagnose("ui.%s: id %q not visible in %q", op, id, name)
		return nil, zero, false
	}
	return list, row, true
}

// SelectListItem changes a list selection through the shared callback path
// without manufacturing pointer or hover state.
func (u *UI) SelectListItem(name, id string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	list, row, ok := u.lookupListRow("SelectListItem", name, id)
	if !ok {
		return false
	}
	if row.HasChildren {
		u.diagnose("ui.SelectListItem: id %q in %q is a category", id, name)
		return false
	}
	u.commitListSelection(list, id)
	return true
}

// SetListExpanded changes a category through the shared callback path
// without manufacturing pointer or hover state.
func (u *UI) SetListExpanded(name, id string, expanded bool) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	list, row, ok := u.lookupListRow("SetListExpanded", name, id)
	if !ok {
		return false
	}
	if !row.HasChildren {
		u.diagnose("ui.SetListExpanded: id %q in %q is a leaf", id, name)
		return false
	}
	u.commitListToggle(list, id, expanded)
	return true
}
