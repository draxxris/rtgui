package render

import (
	"image/color"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Rich-text draw colors. Body text matches single-line widget text;
// links default to blue with a geometric underline unless the segment
// carries its own color or the parent registered a per-kind tint.
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
// stored color and link. Icon fragments draw the whitelisted graphic
// centered in the row; text fragments draw the word run. Storing style in
// the span keeps the cached draw path free of segment lookups and index
// panics. Disabled state dims the tint so rich text matches single-line
// widget text contrast.
func (t *Theme) drawRichCachedFragment(info core.WidgetInfo, span RichSpanLayout) {
	t.drawRichFragment(info, span, true)
}

// drawRichFragment records and draws one text fragment with an optional link
// underline. Multi-line messages underline links; single-line controls pass
// false so links render in color without activation affordance.
func (t *Theme) drawRichFragment(info core.WidgetInfo, span RichSpanLayout, underline bool) {
	if span.IsIcon {
		t.drawRichIconFragment(info, span)
		return
	}
	tint := t.richFragmentTint(span)
	if info.State == core.StateDisabled {
		tint = dimRichTint(tint)
	}
	row := span.Bounds
	size := span.Size
	if size <= 0 {
		size = RichFontSize
	}
	t.logDrawCall(info.Kind, skin.PartText, info.State, row, row, skin.SkinDescriptor{}, tint, false)
	t.drawRichWord(span.Text, size, span.Font, span.Bold, row.X, row.Y, tint)
	if !underline || span.Link.Kind == core.LinkNone {
		return
	}
	line := t.snap(core.Rect{X: row.X, Y: row.Y + row.H - 4, W: row.W, H: 1})
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateNormal, row, line, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(line), tint)
	}
}

// richFragmentTint resolves body, explicit, registered, and default tints.
// Explicit segment colors win, then parent-registered per-kind colors,
// then link blue, then body text.
func (t *Theme) richFragmentTint(span RichSpanLayout) color.RGBA {
	if span.HasColor {
		return span.Color.RGBA()
	}
	if span.Link.Kind != core.LinkNone {
		return t.richLinkTint(false, core.Color{}, span.Link.Kind, richLinkText)
	}
	return richBodyText
}

// drawRichIconFragment records and draws one inline icon placeholder.
// Missing whitelist entries log a placeholder box without drawing pixels;
// linked icons skip the text underline but still earn hover highlights.
func (t *Theme) drawRichIconFragment(info core.WidgetInfo, span RichSpanLayout) {
	t.drawRichIconBox(info, span.Icon, span.Bounds)
}

// drawRichIconBox records and draws one whitelisted icon centered in row.
// Tooltip bodies share this path with absolute rows; message fragments pass
// their own bounds. A missing entry logs geometry without drawing pixels.
func (t *Theme) drawRichIconBox(info core.WidgetInfo, name string, row core.Rect) {
	size := float32(RichIconSize)
	if size > row.H {
		size = row.H
	}
	if size > row.W {
		size = row.W
	}
	dest := core.Rect{X: row.X, Y: row.Y + (row.H-size)/2, W: size, H: size}
	icon, ok := t.LookupInlineIcon(name)
	tint := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if ok && icon.HasTint {
		tint = icon.Tint.RGBA()
	}
	if info.State == core.StateDisabled {
		tint = dimRichTint(tint)
	}
	descriptor := skin.SkinDescriptor{Tint: core.ToColor(tint), HasTexture: ok && icon.ID != 0}
	if ok && icon.ID != 0 {
		descriptor.Texture = skin.Texture{ID: icon.ID, Width: icon.Width, Height: icon.Height}
		descriptor.AtlasRegion = icon.SourceRect()
	}
	t.logDrawCall(info.Kind, skin.PartIcon, info.State, row, t.snap(dest), descriptor, tint, !ok || icon.ID == 0)
	if !rl.IsWindowReady() || !ok || icon.ID == 0 {
		return
	}
	texture := toRaylibTexture(descriptor.Texture)
	drawSingleTexture(texture, atlasRegion(descriptor, texture), t.snap(dest), tint)
}

// drawRichWord draws one styled word run without logging.
// Callers log the text call first. Bold without a bold asset double-draws
// with a one-pixel offset.
func (t *Theme) drawRichWord(text string, size float32, font string, bold bool, x, y float32, tint color.RGBA) {
	if text == "" || !rl.IsWindowReady() {
		return
	}
	size = clampRichFontSize(size)
	if t != nil && (t.HasFont() || t.HasNamedFont(font)) {
		rl.DrawTextEx(t.richFont(size, font), text, rl.NewVector2(x, y), size, richTextSpacing, tint)
		if bold {
			rl.DrawTextEx(t.richFont(size, font), text, rl.NewVector2(x+1, y), size, richTextSpacing, tint)
		}
		return
	}
	rl.DrawText(text, int32(x), int32(y), int32(size), tint)
	if bold {
		rl.DrawText(text, int32(x)+1, int32(y), int32(size), tint)
	}
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
// An authored RichText::highlight replaces the fixed fill; without one the
// fixed fill draws so unskinned messages keep their hover feedback.
func (t *Theme) drawRichHighlight(info core.WidgetInfo, row core.Rect) {
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartOverlay, core.StateHovered)
	if !fallback && hasVisualBackground(descriptor) {
		destination := t.snap(row)
		tint := effectiveTint(descriptor, false)
		t.logDrawCall(info.Kind, skin.PartOverlay, core.StateHovered, row, destination, descriptor, tint, false)
		if rl.IsWindowReady() {
			drawDescriptorBackground(descriptor, destination, tint)
		}
		return
	}
	tint := color.RGBA{R: 67, G: 97, B: 139, A: 55}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateHovered, row, row, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(row), tint)
	}
}
