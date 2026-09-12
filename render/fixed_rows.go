package render

import (
	"image/color"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// ListSelectedAccent is the shared fixed-row selection marker width.
	ListSelectedAccent = float32(3)
	// TableSelectedAccent preserves the table name for the shared marker width.
	TableSelectedAccent = ListSelectedAccent
)

// fixedRowsContent resolves a fixed-row widget's skin-aware shell content.
func (t *Theme) fixedRowsContent(kind core.WidgetKind, bounds core.Rect, state core.WidgetState, class ...string) core.Rect {
	var background, border skin.SkinDescriptor
	if t != nil {
		background, _ = t.resolveDescriptor(kind, skin.PartBackground, state, class...)
		border, _ = t.resolveDescriptor(kind, skin.PartBorder, state, class...)
	}
	return t.snap(ContentRect(bounds, background, border))
}

// fixedRowsViewport excludes the shared scrollbar track when rows overflow.
func fixedRowsViewport(content core.Rect, maxScroll float32) core.Rect {
	if maxScroll <= 0 || !finiteFloat32(maxScroll) {
		return content
	}
	track, ok := ScrollTrackRect(content, maxScroll)
	if !ok {
		return content
	}
	content.W = track.X - content.X
	if content.W < 0 {
		content.W = 0
	}
	return content
}

// drawFixedRowHighlight paints selected, hovered, or pressed row feedback.
func (t *Theme) drawFixedRowHighlight(info core.WidgetInfo, row core.Rect, state core.WidgetState) {
	if state == core.StateNormal || state == core.StateDisabled {
		return
	}
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartOverlay, state, info.Class)
	destination := t.snap(row)
	if !fallback && hasVisualBackground(descriptor) {
		tint := effectiveTint(descriptor, false)
		t.logDrawCall(info.Kind, skin.PartOverlay, state, row, destination, descriptor, tint, false)
		if rl.IsWindowReady() {
			drawDescriptorBackground(descriptor, destination, tint)
		}
		return
	}
	tint := fixedRowFallbackTint(state)
	t.logDrawCall(info.Kind, skin.PartOverlay, state, row, destination, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(destination), tint)
	}
}

// fixedRowFallbackTint selects the shared fill when no highlight is authored.
func fixedRowFallbackTint(state core.WidgetState) color.RGBA {
	switch state {
	case core.StateSelected:
		return color.RGBA{R: 74, G: 58, B: 32, A: 255}
	case core.StatePressed:
		return color.RGBA{R: 44, G: 34, B: 20, A: 255}
	default:
		return color.RGBA{R: 44, G: 58, B: 78, A: 255}
	}
}

// drawFixedRowAccent paints the shared fixed-width selected-row marker.
func (t *Theme) drawFixedRowAccent(info core.WidgetInfo, row core.Rect, width float32) {
	if width > row.W {
		width = row.W
	}
	if width <= 0 || row.H <= 0 {
		return
	}
	accent := core.Rect{X: row.X, Y: row.Y, W: width, H: row.H}
	tint := color.RGBA{R: 232, G: 200, B: 118, A: 255}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateSelected, row, t.snap(accent), skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(t.snap(accent)), tint)
	}
}

// drawFixedRowSeparator paints the shared one-pixel row divider.
func (t *Theme) drawFixedRowSeparator(info core.WidgetInfo, row core.Rect) {
	if row.W <= 0 || row.H <= 0 {
		return
	}
	line := core.Rect{X: row.X, Y: row.Y + row.H - 1, W: row.W, H: 1}
	tint := color.RGBA{R: 255, G: 255, B: 255, A: 16}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateNormal, row, t.snap(line), skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(t.snap(line)), tint)
	}
}
