package render

import (
	"image/color"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Rich-text draw colors for v1. Body text matches single-line widget text;
// links default to blue with a geometric underline unless the segment
// carries its own color.
var (
	richBodyText = color.RGBA{R: 20, G: 20, B: 20, A: 255}
	richLinkText = color.RGBA{R: 60, G: 120, B: 220, A: 255}
)

// DrawRichText renders wrapped message segments with per-fragment link
// underlines. hoveredSeg is the hovered segment index, or -1, and draws the
// ::highlight backdrop behind that segment's fragments.
func (t *Theme) DrawRichText(info core.WidgetInfo, segments []core.RichSegment, hoveredSeg int) {
	if t == nil || len(segments) == 0 {
		return
	}
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	spans := t.LayoutRichSpans(info.Bounds, segments, info.State)
	for _, span := range spans {
		if span.Linked() && span.Segment == hoveredSeg {
			t.drawRichHighlight(info, span.Bounds)
		}
		t.drawRichFragment(info, span, segments[span.Segment])
	}
}

// drawRichFragment records and draws one laid-out text fragment with an
// optional link underline in the fragment's text color.
func (t *Theme) drawRichFragment(info core.WidgetInfo, span RichSpanLayout, segment core.RichSegment) {
	tint := richBodyText
	if segment.HasColor {
		tint = segment.Color.RGBA()
	} else if segment.Link.Kind != core.LinkNone {
		tint = richLinkText
	}
	row := span.Bounds
	t.logDrawCall(info.Kind, skin.PartText, info.State, row, row, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		if t.HasFont() {
			rl.DrawTextEx(t.FontForSize(RichFontSize), span.Text, rl.NewVector2(row.X, row.Y), RichFontSize, richTextSpacing, tint)
		} else {
			rl.DrawText(span.Text, int32(row.X), int32(row.Y), RichFontSize, tint)
		}
	}
	if segment.Link.Kind == core.LinkNone {
		return
	}
	underline := t.snap(core.Rect{X: row.X, Y: row.Y + row.H - 4, W: row.W, H: 1})
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateNormal, row, underline, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		drawFallbackPart(underline, tint)
	}
}

// drawRichHighlight records and draws one hovered link-fragment backdrop.
// A textured RichText::highlight replaces the fixed fill; without one the
// fixed fill draws so unskinned messages keep their hover feedback.
func (t *Theme) drawRichHighlight(info core.WidgetInfo, row core.Rect) {
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartOverlay, core.StateHovered)
	if hasTexture(descriptor, fallback) {
		destination := t.snap(row)
		tint := effectiveTint(descriptor, false)
		t.logDrawCall(info.Kind, skin.PartOverlay, core.StateHovered, row, destination, descriptor, tint, false)
		if rl.IsWindowReady() {
			drawTexturedPart(descriptor, destination, tint)
		}
		return
	}
	tint := color.RGBA{R: 67, G: 97, B: 139, A: 55}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateHovered, row, row, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		drawFallbackPart(row, tint)
	}
}
