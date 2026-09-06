package render

import (
	"github.com/draxxris/rtgui/core"
)

// Menu metrics are fixed in v1 so hit testing and drawing always agree
// without font measurement. Width and row height are logical pixels.
const (
	// MenuWidth is the fixed outer width of an ephemeral context menu.
	MenuWidth = 220
	// MenuRowHeight is the fixed outer row height, separators included.
	MenuRowHeight = 30
)

// MenuContent returns the skin-aware menu content area used for both row
// drawing and row hit testing. It resolves Menu::popup and its border for
// state (with normal fallback) and snaps when pixel snap is on. Callers
// must use this — never raw menu bounds — so hovered, pressed, and drawn
// rows always agree. It is safe on a nil theme, where it returns bounds.
func (t *Theme) MenuContent(bounds core.Rect, state core.WidgetState) core.Rect {
	return t.popupContent(core.WidgetMenu, bounds, state)
}

// MenuRowRect returns one row rect inside menu content. Rows tile content
// exactly: row height is content.H/count starting at content.Y, so borders
// and padding never overlap row content.
func MenuRowRect(content core.Rect, count, index int) (core.Rect, bool) {
	return popupRowRect(content, count, index)
}

// MenuIndexAt returns the row under pos inside menu content, or -1 outside
// it. It inverts MenuRowRect exactly, including edge behavior: the
// bottom-right edge maps outside, matching the drawn rows.
func MenuIndexAt(content core.Rect, count int, pos core.Vec2) int {
	return popupRowIndex(content, count, pos)
}

// MenuOuterBounds returns the fixed outer bounds for count rows anchored at
// pos, clamped into logical so the menu never leaves the viewport.
func MenuOuterBounds(pos, logical core.Vec2, count int) core.Rect {
	if count <= 0 {
		return core.Rect{}
	}
	bounds := core.Rect{X: pos.X, Y: pos.Y, W: MenuWidth, H: float32(count) * MenuRowHeight}
	if logical.X > 0 && bounds.X+bounds.W > logical.X {
		bounds.X = logical.X - bounds.W
	}
	if logical.Y > 0 && bounds.Y+bounds.H > logical.Y {
		bounds.Y = logical.Y - bounds.H
	}
	if bounds.X < 0 {
		bounds.X = 0
	}
	if bounds.Y < 0 {
		bounds.Y = 0
	}
	return bounds
}
