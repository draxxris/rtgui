package render

import (
	"fmt"
	"image/color"
	"math"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// effectiveTint returns either the fixed fallback tint or the descriptor's
// exact RGBA tint. Zero descriptor tint intentionally remains transparent black.
func effectiveTint(descriptor skin.SkinDescriptor, fallback bool) color.RGBA {
	if fallback {
		return color.RGBA{R: 200, G: 200, B: 200, A: 255}
	}
	return descriptor.Tint.RGBA()
}

func (t *Theme) snap(rect core.Rect) core.Rect {
	if t == nil || !t.GetPixelSnap() {
		return rect
	}
	return t.transform.SnapRect(rect)
}

func (t *Theme) resolveDescriptor(kind core.WidgetKind, part skin.SkinPart, state core.WidgetState) (skin.SkinDescriptor, bool) {
	descriptor, ok := t.Lookup(kind, part, state)
	return descriptor, !ok
}

// logDrawCall records one operation only when diagnostics are explicitly enabled.
func (t *Theme) logDrawCall(kind core.WidgetKind, part skin.SkinPart, state core.WidgetState, bounds, dest core.Rect, descriptor skin.SkinDescriptor, tint color.RGBA, fallback bool) {
	if t == nil {
		return
	}
	if fallback && t.diagnosticHandler != nil {
		t.diagnosticHandler(fmt.Sprintf("render: missing skin for widget=%v part=%v state=%v", kind, part, state))
	}
	if t.recorder == nil {
		return
	}
	t.recorder.record(DrawCall{
		Kind:     kind,
		Part:     part,
		State:    state,
		Bounds:   bounds,
		Dest:     dest,
		Src:      descriptor.AtlasRegion,
		Tint:     core.Color{R: tint.R, G: tint.G, B: tint.B, A: tint.A},
		Fallback: fallback,
	})
}

// drawTexturedPart selects simple or nine-patch drawing for descriptor.
// Descriptors without a texture draw nothing; missing skins stay invisible
// (text still draws via drawTextInContent) and are reported via Fallback logs.
func drawTexturedPart(descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA) {
	if !descriptor.HasTexture || descriptor.Texture.ID == 0 {
		return
	}
	texture := toRaylibTexture(descriptor.Texture)
	source := atlasRegion(descriptor, texture)
	if !hasNinePatchBorders(descriptor) {
		drawSingleTexture(texture, source, dest, tint)
		return
	}
	drawNinePatch(texture, source, descriptor, dest, tint)
}

func drawFallbackPart(dest core.Rect, tint color.RGBA) {
	if !rl.IsWindowReady() {
		return
	}
	rl.DrawRectangleRec(toRaylibRect(dest), tint)
	rl.DrawRectangleLinesEx(toRaylibRect(dest), 1, color.RGBA{R: 255, G: 0, B: 255, A: 255})
}

func atlasRegion(descriptor skin.SkinDescriptor, texture rl.Texture2D) core.Rect {
	if descriptor.AtlasRegion.W == 0 && descriptor.AtlasRegion.H == 0 {
		return core.Rect{W: float32(texture.Width), H: float32(texture.Height)}
	}
	return descriptor.AtlasRegion
}

func hasNinePatchBorders(descriptor skin.SkinDescriptor) bool {
	if !descriptor.HasNinePatch {
		return false
	}
	patch := descriptor.NinePatch
	return patch.Left != 0 || patch.Top != 0 || patch.Right != 0 || patch.Bottom != 0
}

func drawSingleTexture(texture rl.Texture2D, source, dest core.Rect, tint color.RGBA) {
	rl.DrawTexturePro(texture, toRaylibRect(source), toRaylibRect(dest), rl.NewVector2(0, 0), 0, tint)
}

// drawNinePatch renders source across the deterministic nine destination rectangles.
func drawNinePatch(texture rl.Texture2D, source core.Rect, descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA) {
	sourceRects := NinePatchSourceRects(source, descriptor.NinePatch)
	destinationRects := NinePatchRects(NinePatchConfig{
		Left:   float32(descriptor.NinePatch.Left),
		Top:    float32(descriptor.NinePatch.Top),
		Right:  float32(descriptor.NinePatch.Right),
		Bottom: float32(descriptor.NinePatch.Bottom),
	}, dest)
	for i := range destinationRects {
		destination := destinationRects[i]
		sourceRect := sourceRects[i]
		if !validPatch(destination, sourceRect) {
			continue
		}
		if skipCenter(descriptor.CenterFill, i) {
			continue
		}
		drawSingleTexture(texture, sourceRect, destination, tint)
	}
}

func validPatch(destination, source core.Rect) bool {
	return destination.W > 0 && destination.H > 0 && source.W > 0 && source.H > 0
}

func skipCenter(centerFill bool, index int) bool { return !centerFill && index == 4 }

// widgetTextOrigin returns the font size and rounded text origin shared by
// single-line widget text layout, including textbox selection and caret.
// Sizes are requested ~1.5x the nominal body size: raylib interprets TTF
// sizes as pixel height (ascent+descent) rather than EM units, so common
// faces render much smaller than requested (see the in-house renderer
// roadmap entry). Sizes below target a true ~13-15px EM for body text.
func widgetTextOrigin(content core.Rect) (int32, float32, float32) {
	fontSize := int32(22)
	if content.H < 20 {
		fontSize = 14
	} else if content.H < 28 {
		fontSize = 20
	}
	x := float32(math.Round(float64(content.X + 6)))
	y := float32(math.Round(float64(content.Y + (content.H-float32(fontSize))/2)))
	return fontSize, x, y
}

// textLayout computes the font size and origin for widget text considering explicit size and alignment.
func (t *Theme) textLayout(info core.WidgetInfo, value string, content core.Rect) (float32, float32, float32) {
	fontSize := info.FontSize
	if fontSize <= 0 {
		autoSize, defX, defY := widgetTextOrigin(content)
		if info.Align == core.AlignLeft {
			return float32(autoSize), defX, defY
		}
		fontSize = float32(autoSize)
	}
	y := float32(math.Round(float64(content.Y + (content.H-fontSize)/2)))
	if info.Kind == core.WidgetLabel {
		switch info.Align {
		case core.AlignCenter:
			w := t.MeasureText(value, fontSize, info.Italic)
			x := float32(math.Round(float64(content.X + (content.W-w)/2)))
			return fontSize, x, y
		case core.AlignRight:
			w := t.MeasureText(value, fontSize, info.Italic)
			x := float32(math.Round(float64(content.X + content.W - w)))
			return fontSize, x, y
		default:
			return fontSize, content.X, y
		}
	}
	switch info.Align {
	case core.AlignCenter:
		w := t.MeasureText(value, fontSize, info.Italic)
		x := float32(math.Round(float64(content.X + (content.W-w)/2)))
		return fontSize, x, y
	case core.AlignRight:
		w := t.MeasureText(value, fontSize, info.Italic)
		x := float32(math.Round(float64(content.X + content.W - w - 6)))
		return fontSize, x, y
	default:
		x := float32(math.Round(float64(content.X + 6)))
		return fontSize, x, y
	}
}

// drawTextInContent lays out and draws text, recording its text operation when enabled.
func (t *Theme) drawTextInContent(info core.WidgetInfo, value string, content core.Rect, state core.WidgetState) {
	if value == "" || content.W <= 0 || content.H <= 0 {
		return
	}
	fontSize, x, y := t.textLayout(info, value, content)
	textColor := defaultWidgetTextColor(info, state)
	t.logDrawCall(info.Kind, skin.PartText, state, content, content, skin.SkinDescriptor{}, textColor, false)
	t.DrawText(value, x, y, fontSize, info.Italic, textColor)
}

// defaultWidgetTextColor resolves the text tint from explicit widget color or state default.
func defaultWidgetTextColor(info core.WidgetInfo, state core.WidgetState) color.RGBA {
	if info.HasTextColor {
		base := info.TextColor.RGBA()
		if state == core.StateDisabled {
			return color.RGBA{R: base.R / 2, G: base.G / 2, B: base.B / 2, A: base.A}
		}
		return base
	}
	if state == core.StateDisabled {
		return color.RGBA{R: 130, G: 130, B: 130, A: 255}
	} else if state == core.StatePressed {
		return color.RGBA{R: 30, G: 30, B: 30, A: 255}
	}
	return color.RGBA{R: 20, G: 20, B: 20, A: 255}
}

// drawPart resolves, records, and optionally draws one widget skin part.
func (t *Theme) drawPart(kind core.WidgetKind, part skin.SkinPart, bounds core.Rect, state core.WidgetState) (skin.SkinDescriptor, bool) {
	descriptor, fallback := t.resolveDescriptor(kind, part, state)
	tint := effectiveTint(descriptor, fallback)
	destination := t.snap(bounds)
	t.logDrawCall(kind, part, state, bounds, destination, descriptor, tint, fallback)
	if rl.IsWindowReady() {
		if descriptor.HasTexture && descriptor.Texture.ID != 0 {
			drawTexturedPart(descriptor, destination, tint)
		} else if fallback && t != nil && t.debugMode {
			drawFallbackPart(destination, color.RGBA{R: 255, G: 0, B: 255, A: 100})
		}
	}
	return descriptor, fallback
}

// DrawWidgetPart draws one part of a widget.
func (t *Theme) DrawWidgetPart(kind core.WidgetKind, part skin.SkinPart, bounds core.Rect, state core.WidgetState) {
	if t != nil {
		t.drawPart(kind, part, bounds, state)
	}
}

// DrawWidget renders a widget using the theme's registry and transform.
// Checkbox has no background by design (::box is PartIcon, ::checkmark is
// PartCheckmark); its content is the full bounds so missing art stays invisible.
func (t *Theme) DrawWidget(info core.WidgetInfo, value string, amount float32, checked bool) {
	if t == nil {
		return
	}
	var content core.Rect
	if info.Kind == core.WidgetCheckbox {
		content = t.snap(info.Bounds)
	} else {
		background, _ := t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
		border, _ := t.resolveDescriptor(info.Kind, skin.PartBorder, info.State)
		content = t.snap(ContentRect(info.Bounds, background, border))
	}
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}

	switch info.Kind {
	case core.WidgetCheckbox:
		t.drawCheckbox(info, value, content, checked)
	case core.WidgetSlider:
		t.drawSlider(info, value, content, amount)
	case core.WidgetProgressBar:
		t.drawProgressBar(info, value, content, amount)
	default:
		t.drawTextInContent(info, value, content, info.State)
	}
}

// DrawDropdownPopup renders a dropdown list with popup skin parts and
// themed text. info.Bounds is the complete popup bounds. The popup fill is
// Dropdown::popup and its ring is the same selector's border-image; rows lay
// out inside their merged insets so the ring never overlaps row content.
func (t *Theme) DrawDropdownPopup(info core.WidgetInfo, items []string, hovered int) {
	if t == nil || len(items) == 0 || info.Bounds.H <= 0 {
		return
	}
	t.drawPopupShell(info)
	content := t.DropdownPopupContent(info.Bounds, info.State)
	for index, item := range items {
		row, ok := DropdownPopupRow(content, len(items), index)
		if !ok {
			continue
		}
		if index == hovered {
			t.drawPopupRowHighlight(info, row)
		}
		t.drawDropdownText(info, row, item)
	}
}

// drawDropdownText records and draws one popup label with the gallery colors.
// The label keeps the popup row's +16 horizontal indent but centers
// vertically with widgetTextOrigin so row text matches other widgets.
func (t *Theme) drawDropdownText(info core.WidgetInfo, row core.Rect, value string) {
	tint := color.RGBA{R: 230, G: 240, B: 255, A: 255}
	t.logDrawCall(info.Kind, skin.PartText, info.State, row, row, skin.SkinDescriptor{}, tint, false)
	if !rl.IsWindowReady() {
		return
	}
	fontSize, _, y := widgetTextOrigin(row)
	x := float32(math.Round(float64(row.X + 16)))
	if t.HasFont() {
		rl.DrawTextEx(t.FontForSize(float32(fontSize)), value, rl.NewVector2(x, y), float32(fontSize), float32(fontSize)/10, tint)
		return
	}
	rl.DrawText(value, int32(x), int32(y), fontSize, tint)
}

// drawCheckbox renders its icon state followed by its optional text.
func (t *Theme) drawCheckbox(info core.WidgetInfo, value string, content core.Rect, checked bool) {
	if value == "" {
		if checked {
			t.drawCheckmark(info, content)
		} else {
			t.drawUncheckedBox(info, content)
		}
		return
	}
	boxSize := float32(20)
	if boxSize > content.H {
		boxSize = content.H
	}
	iconRect := core.Rect{
		X: content.X + 4,
		Y: content.Y + (content.H-boxSize)/2,
		W: boxSize,
		H: boxSize,
	}
	if checked {
		t.drawCheckmark(info, iconRect)
	} else {
		t.drawUncheckedBox(info, iconRect)
	}
	textRect := core.Rect{
		X: content.X + boxSize + 10,
		Y: content.Y,
		W: content.W - (boxSize + 10),
		H: content.H,
	}
	t.drawTextInContent(info, value, textRect, info.State)
}

// drawUncheckedBox renders the empty box icon when a textured PartIcon is
// registered for checkboxes. Without one it draws nothing, preserving the
// old no-skin behavior of an empty unchecked box.
func (t *Theme) drawUncheckedBox(info core.WidgetInfo, content core.Rect) {
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartIcon, info.State)
	if !hasTexture(descriptor, fallback) {
		return
	}
	t.drawCenteredIcon(info, content, skin.PartIcon, descriptor)
}

func (t *Theme) drawCheckmark(info core.WidgetInfo, content core.Rect) {
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartCheckmark, info.State)
	if hasTexture(descriptor, fallback) {
		t.drawCenteredIcon(info, content, skin.PartCheckmark, descriptor)
		return
	}
	t.drawGeometryCheckmark(info, fallback)
}

func hasTexture(descriptor skin.SkinDescriptor, fallback bool) bool {
	return !fallback && descriptor.HasTexture && descriptor.Texture.ID != 0
}

// drawCenteredIcon renders a small centered part (checkmark or unchecked box)
// from a textured descriptor, capping it at 60% of the content box.
func (t *Theme) drawCenteredIcon(info core.WidgetInfo, content core.Rect, part skin.SkinPart, descriptor skin.SkinDescriptor) {
	tint := effectiveTint(descriptor, false)
	width, height := checkmarkSize(descriptor, content)
	destination := t.snap(core.Rect{
		X: content.X + (content.W-width)/2,
		Y: content.Y + (content.H-height)/2,
		W: width,
		H: height,
	})
	t.logDrawCall(info.Kind, part, info.State, destination, destination, descriptor, tint, false)
	if !rl.IsWindowReady() {
		return
	}
	texture := toRaylibTexture(descriptor.Texture)
	source := atlasRegion(descriptor, texture)
	drawSingleTexture(texture, source, destination, tint)
}

// checkmarkSize derives an icon size from its atlas region and content box.
func checkmarkSize(descriptor skin.SkinDescriptor, content core.Rect) (float32, float32) {
	width, height := descriptor.AtlasRegion.W, descriptor.AtlasRegion.H
	if width == 0 || height == 0 {
		width, height = 16, 16
	}
	if width > content.W {
		width = content.W * 0.6
	}
	if height > content.H {
		height = content.H * 0.6
	}
	return width, height
}

// drawGeometryCheckmark records a missing checkmark without drawing pixels.
// Untextured checkmarks stay invisible by design; callers log Fallback=true.
func (t *Theme) drawGeometryCheckmark(info core.WidgetInfo, missingSkin bool) {
	destination := t.snap(info.Bounds)
	checkColor := color.RGBA{R: 20, G: 120, B: 60, A: 255}
	if missingSkin {
		checkColor = color.RGBA{A: 255}
	}
	t.logDrawCall(info.Kind, skin.PartCheckmark, info.State, destination, destination, skin.SkinDescriptor{}, checkColor, true)
}

// drawSlider renders the track and thumb at amount's clamped position.
func (t *Theme) drawSlider(info core.WidgetInfo, value string, content core.Rect, amount float32) {
	track, fallback := t.resolveDescriptor(info.Kind, skin.PartTrack, info.State)
	trackTint := effectiveTint(track, fallback)
	trackRect := sliderTrackRect(t, info, content, track, fallback)
	t.logDrawCall(info.Kind, skin.PartTrack, info.State, trackRect, trackRect, track, trackTint, fallback)
	if rl.IsWindowReady() {
		if track.HasTexture && track.Texture.ID != 0 {
			drawTexturedPart(track, trackRect, trackTint)
		} else if fallback && t != nil && t.debugMode {
			drawFallbackPart(trackRect, color.RGBA{R: 255, G: 0, B: 255, A: 100})
		}
	}

	thumb, thumbFallback := t.resolveDescriptor(info.Kind, skin.PartThumb, info.State)
	thumbTint := effectiveTint(thumb, thumbFallback)
	thumbRect := sliderThumbRect(t, trackRect, thumb, thumbFallback, amount)
	t.logDrawCall(info.Kind, skin.PartThumb, info.State, thumbRect, thumbRect, thumb, thumbTint, thumbFallback)
	if rl.IsWindowReady() {
		if thumb.HasTexture && thumb.Texture.ID != 0 {
			drawTexturedPart(thumb, thumbRect, thumbTint)
		} else if thumbFallback && t != nil && t.debugMode {
			drawFallbackPart(thumbRect, color.RGBA{R: 255, G: 0, B: 255, A: 150})
		}
	}
	if value != "" {
		t.drawTextInContent(info, value, content, info.State)
	}
}

// sliderTrackRect chooses the content-aligned track rectangle for a slider.
func sliderTrackRect(t *Theme, info core.WidgetInfo, content core.Rect, descriptor skin.SkinDescriptor, fallback bool) core.Rect {
	trackHeight := sliderTrackHeight(content.H, descriptor, fallback)
	trackY := content.Y + (content.H-trackHeight)/2
	if content.H == 0 {
		trackY = info.Bounds.Y + (info.Bounds.H-trackHeight)/2
	}
	trackWidth := content.W
	if trackWidth == 0 {
		trackWidth = info.Bounds.W
	}
	trackRect := t.snap(core.Rect{X: content.X, Y: trackY, W: trackWidth, H: trackHeight})
	if trackRect.W == 0 && trackRect.H == 0 {
		return t.snap(core.Rect{X: info.Bounds.X, Y: info.Bounds.Y + (info.Bounds.H-trackHeight)/2, W: info.Bounds.W, H: trackHeight})
	}
	return trackRect
}

// sliderTrackHeight chooses a textured or fallback track height within content.
func sliderTrackHeight(contentHeight float32, descriptor skin.SkinDescriptor, fallback bool) float32 {
	trackHeight := float32(8)
	if !fallback && descriptor.AtlasRegion.H > 0 {
		trackHeight = descriptor.AtlasRegion.H
	}
	if trackHeight > contentHeight && contentHeight > 0 {
		return contentHeight * 0.5
	}
	return trackHeight
}

// sliderThumbRect positions the thumb in track for the clamped amount.
func sliderThumbRect(t *Theme, track core.Rect, descriptor skin.SkinDescriptor, fallback bool, amount float32) core.Rect {
	thumbWidth, thumbHeight := float32(14), float32(20)
	if !fallback && descriptor.AtlasRegion.W > 0 {
		thumbWidth = descriptor.AtlasRegion.W
	}
	if !fallback && descriptor.AtlasRegion.H > 0 {
		thumbHeight = descriptor.AtlasRegion.H
	}
	amount = clamp01(amount)
	return t.snap(core.Rect{
		X: track.X + amount*(track.W-thumbWidth),
		Y: track.Y + (track.H-thumbHeight)/2,
		W: thumbWidth,
		H: thumbHeight,
	})
}

// drawProgressBar renders a track, a proportional fill, and its spark.
// The fill is ProgressBar::fill, which shares the PartOverlay key with
// Dropdown::highlight; registry keys are widget-scoped so they never meet.
func (t *Theme) drawProgressBar(info core.WidgetInfo, value string, content core.Rect, amount float32) {
	track, fallback := t.resolveDescriptor(info.Kind, skin.PartTrack, info.State)
	trackTint := effectiveTint(track, fallback)
	trackRect := content
	if trackRect.W == 0 || trackRect.H == 0 {
		trackRect = t.snap(info.Bounds)
	}
	t.logDrawCall(info.Kind, skin.PartTrack, info.State, trackRect, trackRect, track, trackTint, fallback)
	if rl.IsWindowReady() {
		if track.HasTexture && track.Texture.ID != 0 {
			drawTexturedPart(track, trackRect, trackTint)
		} else if fallback && t != nil && t.debugMode {
			drawFallbackPart(trackRect, color.RGBA{R: 255, G: 0, B: 255, A: 100})
		}
	}

	fillWidth := t.drawProgressBarFill(info, trackRect, amount)
	if fillWidth > 0 {
		t.drawProgressSpark(info, trackRect, fillWidth)
	}
	if value != "" {
		t.drawTextInContent(info, value, trackRect, info.State)
	}
}

// drawProgressBarFill renders the proportional fill overlay for a progress bar.
func (t *Theme) drawProgressBarFill(info core.WidgetInfo, trackRect core.Rect, amount float32) float32 {
	amount = clamp01(amount)
	fill, fillFallback := t.resolveDescriptor(info.Kind, skin.PartOverlay, info.State)
	fillTint := effectiveTint(fill, fillFallback)
	fillWidth := trackRect.W * amount
	if fillWidth <= 0 {
		return 0
	}
	fillRect := t.snap(core.Rect{X: trackRect.X, Y: trackRect.Y, W: fillWidth, H: trackRect.H})
	t.logDrawCall(info.Kind, skin.PartOverlay, info.State, fillRect, fillRect, fill, fillTint, fillFallback)
	if rl.IsWindowReady() {
		if fill.HasTexture && fill.Texture.ID != 0 {
			drawTexturedPart(fill, fillRect, fillTint)
		} else if fillFallback && t != nil && t.debugMode {
			drawFallbackPart(fillRect, color.RGBA{R: 255, G: 0, B: 255, A: 150})
		}
	}
	return fillWidth
}

// drawProgressSpark renders the ::spark marker centered on the fill edge.
// Without a spark descriptor it draws nothing, preserving unskinned bars.
func (t *Theme) drawProgressSpark(info core.WidgetInfo, track core.Rect, fillWidth float32) {
	spark, fallback := t.resolveDescriptor(info.Kind, skin.PartSpark, info.State)
	if !hasTexture(spark, fallback) {
		return
	}
	amount := fillWidth / track.W
	if track.W <= 0 || amount <= 0 || amount >= 1 {
		return
	}
	width, height := sparkSize(spark, track)
	x := track.X + fillWidth - width/2
	if x < track.X {
		x = track.X
	}
	if x+width > track.X+track.W {
		x = track.X + track.W - width
	}
	destination := t.snap(core.Rect{X: x, Y: track.Y + (track.H-height)/2, W: width, H: height})
	tint := effectiveTint(spark, false)
	t.logDrawCall(info.Kind, skin.PartSpark, info.State, destination, destination, spark, tint, false)
	if rl.IsWindowReady() {
		texture := toRaylibTexture(spark.Texture)
		drawSingleTexture(texture, atlasRegion(spark, texture), destination, tint)
	}
}

// sparkSize derives a marker size from its atlas region, defaulting to a
// thin full-height bar when the region carries no size.
func sparkSize(descriptor skin.SkinDescriptor, track core.Rect) (float32, float32) {
	width, height := descriptor.AtlasRegion.W, descriptor.AtlasRegion.H
	if width <= 0 {
		width = 4
	}
	if height <= 0 {
		height = track.H
	}
	if height > track.H && track.H > 0 {
		height = track.H
	}
	return width, height
}

// clamp01 confines value to the unit interval.
func clamp01(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func toRaylibRect(rect core.Rect) rl.Rectangle {
	return rl.Rectangle{X: rect.X, Y: rect.Y, Width: rect.W, Height: rect.H}
}

func toRaylibTexture(texture skin.Texture) rl.Texture2D {
	return rl.NewTexture2D(texture.ID, texture.Width, texture.Height, texture.Mipmaps, rl.PixelFormat(texture.Format))
}
