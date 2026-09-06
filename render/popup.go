package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// DropdownPopupContent returns the skin-aware popup content area used for
// both row drawing and row hit testing. It resolves Dropdown::popup and its
// border-image for state (with normal fallback) and snaps when pixel snap
// is on. Callers must use this — never raw popup bounds — so hovered,
// pressed, and drawn rows always agree. It is safe on a nil theme, where
// it returns the normalized bounds.
func (t *Theme) DropdownPopupContent(bounds core.Rect, state core.WidgetState) core.Rect {
	var background, border skin.SkinDescriptor
	if t != nil {
		background, _ = t.resolveDescriptor(core.WidgetDropdown, skin.PartPopup, state)
		border, _ = t.resolveDescriptor(core.WidgetDropdown, skin.PartPopupBorder, state)
	}
	return t.snap(ContentRect(bounds, background, border))
}

// DropdownPopupRow returns one row rect inside popup content. Rows tile
// content exactly: rowHeight is content.H/count and rows start at
// content.Y, so borders and padding never overlap row content.
func DropdownPopupRow(content core.Rect, count, index int) (core.Rect, bool) {
	if count <= 0 || index < 0 || index >= count || content.H <= 0 {
		return core.Rect{}, false
	}
	height := content.H / float32(count)
	return core.Rect{X: content.X, Y: content.Y + float32(index)*height, W: content.W, H: height}, true
}

// DropdownPopupIndex returns the row under pos inside popup content, or -1
// outside it. It inverts DropdownPopupRow exactly, including edge behavior:
// the bottom-right edge maps outside, matching the drawn rows.
func DropdownPopupIndex(content core.Rect, count int, pos core.Vec2) int {
	if count <= 0 || content.W <= 0 || content.H <= 0 {
		return -1
	}
	if pos.X < content.X || pos.X > content.X+content.W || pos.Y < content.Y || pos.Y > content.Y+content.H {
		return -1
	}
	index := int((pos.Y - content.Y) / (content.H / float32(count)))
	if index < 0 || index >= count {
		return -1
	}
	return index
}
