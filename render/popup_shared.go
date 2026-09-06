package render

import (
	"image/color"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// popupContent returns the skin-aware popup content area shared by dropdown
// and menu popups. It resolves kind's ::popup shell and border for state
// (with normal fallback) and snaps when pixel snap is on. Callers must use
// this — never raw popup bounds — so hovered, pressed, and drawn rows always
// agree. It is safe on a nil theme, where it returns the normalized bounds.
func (t *Theme) popupContent(kind core.WidgetKind, bounds core.Rect, state core.WidgetState) core.Rect {
	var background, border skin.SkinDescriptor
	if t != nil {
		background, _ = t.resolveDescriptor(kind, skin.PartPopup, state)
		border, _ = t.resolveDescriptor(kind, skin.PartPopupBorder, state)
	}
	return t.snap(ContentRect(bounds, background, border))
}

// popupRowRect returns one row rect inside popup content. Rows tile content
// exactly: row height is content.H/count starting at content.Y, so borders
// and padding never overlap row content.
func popupRowRect(content core.Rect, count, index int) (core.Rect, bool) {
	if count <= 0 || index < 0 || index >= count || content.H <= 0 {
		return core.Rect{}, false
	}
	height := content.H / float32(count)
	return core.Rect{X: content.X, Y: content.Y + float32(index)*height, W: content.W, H: height}, true
}

// popupRowIndex returns the row under pos inside popup content, or -1
// outside it. It inverts popupRowRect exactly, including edge behavior: the
// bottom-right edge maps outside, matching the drawn rows.
func popupRowIndex(content core.Rect, count int, pos core.Vec2) int {
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

// drawPopupShell renders a popup shell and records the widget snapshot.
func (t *Theme) drawPopupShell(info core.WidgetInfo) {
	t.DrawWidgetPart(info.Kind, skin.PartPopup, info.Bounds, info.State)
	t.DrawWidgetPart(info.Kind, skin.PartPopupBorder, info.Bounds, info.State)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
}

// drawPopupRowHighlight records and draws one hovered popup row background.
// A textured ::highlight replaces the fixed fill; without one the fixed fill
// draws so unskinned popups keep their hover feedback.
func (t *Theme) drawPopupRowHighlight(info core.WidgetInfo, row core.Rect) {
	destination := t.snap(core.Rect{X: row.X + 4, Y: row.Y + 3, W: row.W - 8, H: row.H - 6})
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartOverlay, core.StateHovered)
	if hasTexture(descriptor, fallback) {
		tint := effectiveTint(descriptor, false)
		t.logDrawCall(info.Kind, skin.PartOverlay, core.StateHovered, row, destination, descriptor, tint, false)
		if rl.IsWindowReady() {
			drawTexturedPart(descriptor, destination, tint)
		}
		return
	}
	tint := color.RGBA{R: 67, G: 97, B: 139, A: 255}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateHovered, row, destination, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		drawFallbackPart(destination, tint)
	}
}
