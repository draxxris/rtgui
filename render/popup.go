package render

import (
	"github.com/draxxris/rtgui/core"
)

// DropdownPopupContent returns the skin-aware popup content area used for
// both row drawing and row hit testing. It resolves Dropdown::popup and its
// border-image for state (with normal fallback) and snaps when pixel snap
// is on. Callers must use this — never raw popup bounds — so hovered,
// pressed, and drawn rows always agree. It is safe on a nil theme, where
// it returns the normalized bounds.
func (t *Theme) DropdownPopupContent(bounds core.Rect, state core.WidgetState) core.Rect {
	return t.popupContent(core.WidgetDropdown, bounds, state)
}

// DropdownPopupRow returns one row rect inside popup content. Rows tile
// content exactly: rowHeight is content.H/count and rows start at
// content.Y, so borders and padding never overlap row content.
func DropdownPopupRow(content core.Rect, count, index int) (core.Rect, bool) {
	return popupRowRect(content, count, index)
}

// DropdownPopupIndex returns the row under pos inside popup content, or -1
// outside it. It inverts DropdownPopupRow exactly, including edge behavior:
// the bottom-right edge maps outside, matching the drawn rows.
func DropdownPopupIndex(content core.Rect, count int, pos core.Vec2) int {
	return popupRowIndex(content, count, pos)
}
