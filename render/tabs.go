package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// TabContent returns the skin-aware tab strip content area used for both
// tab drawing and tab hit testing. It resolves TabBar background and border
// for state (with normal fallback) and snaps when pixel snap is on. Callers
// must use this — never raw bar bounds — so hovered, pressed, and drawn
// tabs always agree. It is safe on a nil theme, where it returns bounds.
func (t *Theme) TabContent(bounds core.Rect, state core.WidgetState, class ...string) core.Rect {
	var background, border skin.SkinDescriptor
	if t != nil {
		background, _ = t.resolveDescriptor(core.WidgetTabBar, skin.PartBackground, state, class...)
		border, _ = t.resolveDescriptor(core.WidgetTabBar, skin.PartBorder, state, class...)
	}
	return t.snap(ContentRect(bounds, background, border))
}

// TabTabRect returns one equal-width tab cell inside tab content. Cells tile
// content exactly: cell width is content.W/count starting at content.X.
func TabTabRect(content core.Rect, count, index int) (core.Rect, bool) {
	if count <= 0 || index < 0 || index >= count || content.W <= 0 {
		return core.Rect{}, false
	}
	width := content.W / float32(count)
	return core.Rect{X: content.X + float32(index)*width, Y: content.Y, W: width, H: content.H}, true
}

// TabIndexAt returns the tab cell under pos inside tab content, or -1
// outside it. It inverts TabTabRect exactly, including edge behavior.
func TabIndexAt(content core.Rect, count int, pos core.Vec2) int {
	if count <= 0 || content.W <= 0 || content.H <= 0 {
		return -1
	}
	if pos.X < content.X || pos.X > content.X+content.W || pos.Y < content.Y || pos.Y > content.Y+content.H {
		return -1
	}
	index := int((pos.X - content.X) / (content.W / float32(count)))
	if index < 0 || index >= count {
		return -1
	}
	return index
}
