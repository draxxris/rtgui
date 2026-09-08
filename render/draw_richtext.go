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
// ::highlight backdrop behind that segment's fragments. It stays for external
// tests; the UI should Update a RichLayoutCache once and call
// DrawRichTextLayout per frame instead.
func (t *Theme) DrawRichText(info core.WidgetInfo, segments []core.RichSegment, hoveredSeg int) {
	if t == nil || len(segments) == 0 {
		return
	}
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	spans := t.LayoutRichSpans(info.Bounds, segments, info.State)
	t.PushClip(t.RichContent(info.Bounds, info.State))
	defer t.PopClip()
	for _, span := range spans {
		if span.Linked() && span.Segment == hoveredSeg {
			t.drawRichHighlight(info, span.Bounds)
		}
		t.drawRichCachedFragment(info, span)
	}
}

// DrawRichTextLayout renders cached fragments without measuring or wrapping.
// The cache must already reflect info.Bounds and info.State via Update or
// UpdateSegments; this call itself never layouts and allocates nothing after
// recorder warmup. hoveredSeg highlights one linked segment, or -1 for none.
func (t *Theme) DrawRichTextLayout(info core.WidgetInfo, cache *RichLayoutCache, hoveredSeg int) {
	if t == nil || cache == nil || cache.Len() == 0 {
		return
	}
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	t.PushClip(cache.Content())
	defer t.PopClip()
	for _, span := range cache.Spans() {
		if span.Linked() && span.Segment == hoveredSeg {
			t.drawRichHighlight(info, span.Bounds)
		}
		t.drawRichCachedFragment(info, span)
	}
}

// drawRichCachedFragment records and draws one cached fragment using its
// stored color and link. Storing style in the span keeps the cached draw path
// free of segment lookups and index panics. Disabled state dims the tint so
// rich text matches single-line widget text contrast.
func (t *Theme) drawRichCachedFragment(info core.WidgetInfo, span RichSpanLayout) {
	tint := richFragmentTint(span)
	if info.State == core.StateDisabled {
		tint = dimRichTint(tint)
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
	if span.Link.Kind == core.LinkNone {
		return
	}
	underline := t.snap(core.Rect{X: row.X, Y: row.Y + row.H - 4, W: row.W, H: 1})
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateNormal, row, underline, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(underline), tint)
	}
}

// richFragmentTint resolves body, explicit, and link-blue tints for fragments.
func richFragmentTint(span RichSpanLayout) color.RGBA {
	if span.HasColor {
		return span.Color.RGBA()
	}
	if span.Link.Kind != core.LinkNone {
		return richLinkText
	}
	return richBodyText
}

// dimRichTint halves a fragment tint for disabled state, mirroring
// single-line widget text dimming while preserving alpha.
func dimRichTint(tint color.RGBA) color.RGBA {
	tint.R /= 2
	tint.G /= 2
	tint.B /= 2
	return tint
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
		rl.DrawRectangleRec(toRaylibRect(row), tint)
	}
}
