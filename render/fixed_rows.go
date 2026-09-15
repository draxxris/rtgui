package render

import (
	"image/color"
	"math"

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

// drawFixedRowSeparator paints the shared one-pixel row divider. The divider
// covers exactly one physical pixel (see fixedRowDividerRect) so it stays
// visible at every window scale instead of fading on bad subpixel phases.
func (t *Theme) drawFixedRowSeparator(info core.WidgetInfo, row core.Rect) {
	if row.W <= 0 || row.H <= 0 {
		return
	}
	line := t.fixedRowDividerRect(row)
	tint := color.RGBA{R: 255, G: 255, B: 255, A: 16}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateNormal, row, t.snap(line), skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(t.snap(line)), tint)
	}
}

// fixedRowDividerRect quantizes a row-bottom divider to exactly one physical
// pixel anchored inside the row: at downscaled sizes a 1-logical-px line
// would straddle two physical rows and fade, with the worst phase dropping
// below visibility as the window resizes. Anchoring to the row bottom keeps
// the following row's opaque state backgrounds from ever covering it.
// Without a usable transform it falls back to the 1-logical-px line.
func (t *Theme) fixedRowDividerRect(row core.Rect) core.Rect {
	fallback := core.Rect{X: row.X, Y: row.Y + row.H - 1, W: row.W, H: 1}
	if t == nil || t.transform == nil {
		return fallback
	}
	_, sy := t.transform.Scale()
	if math.IsNaN(float64(sy)) || math.IsInf(float64(sy), 0) || sy <= 0 {
		return fallback
	}
	bottomPhys := float64(t.transform.ViewportToPhysical(core.Vec2{Y: row.Y + row.H}).Y)
	topPhys := math.Round(bottomPhys) - 1
	if topPhys < 0 {
		topPhys = 0
	}
	top := t.transform.PhysicalToViewport(core.Vec2{Y: float32(topPhys)}).Y
	x, w := row.X, row.W
	if t.GetPixelSnap() {
		x, w = t.transform.Snap(x), t.transform.Snap(w)
	}
	return core.Rect{X: x, Y: top, W: w, H: 1 / sy}
}
