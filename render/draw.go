package render

import (
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
	if t == nil || t.recorder == nil {
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

// drawTexturedPart selects simple, nine-patch, or fallback drawing for descriptor.
func drawTexturedPart(descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA) {
	if !descriptor.HasTexture || descriptor.Texture.ID == 0 {
		drawFallbackPart(dest, tint)
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
	rl.DrawRectangleRec(toRaylibRect(dest), tint)
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

// drawTextInContent lays out and draws text, recording its text operation when enabled.
func (t *Theme) drawTextInContent(kind core.WidgetKind, value string, content core.Rect, state core.WidgetState) {
	if value == "" || content.W <= 0 || content.H <= 0 {
		return
	}
	fontSize := int32(16)
	if content.H < 20 {
		fontSize = 10
	} else if content.H < 28 {
		fontSize = 14
	}
	textColor := color.RGBA{R: 20, G: 20, B: 20, A: 255}
	if state == core.StateDisabled {
		textColor = color.RGBA{R: 130, G: 130, B: 130, A: 255}
	} else if state == core.StatePressed {
		textColor = color.RGBA{R: 30, G: 30, B: 30, A: 255}
	}
	x := float32(math.Round(float64(content.X + 6)))
	y := float32(math.Round(float64(content.Y + (content.H-float32(fontSize))/2)))
	t.logDrawCall(kind, skin.PartText, state, content, content, skin.SkinDescriptor{}, textColor, false)
	if rl.IsWindowReady() {
		if t != nil && t.HasFont() {
			rl.DrawTextEx(t.FontForSize(float32(fontSize)), value, rl.NewVector2(x, y), float32(fontSize), float32(fontSize)/10, textColor)
		} else {
			rl.DrawText(value, int32(x), int32(y), fontSize, textColor)
		}
	}
}

// drawPart resolves, records, and optionally draws one widget skin part.
func (t *Theme) drawPart(kind core.WidgetKind, part skin.SkinPart, bounds core.Rect, state core.WidgetState) (skin.SkinDescriptor, bool) {
	descriptor, fallback := t.resolveDescriptor(kind, part, state)
	tint := effectiveTint(descriptor, fallback)
	destination := t.snap(bounds)
	t.logDrawCall(kind, part, state, bounds, destination, descriptor, tint, fallback)
	if rl.IsWindowReady() {
		drawTexturedPart(descriptor, destination, tint)
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
func (t *Theme) DrawWidget(info core.WidgetInfo, value string, amount float32, checked bool) {
	if t == nil {
		return
	}
	background, _ := t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	content := t.snap(ContentRect(info.Bounds, background))

	switch info.Kind {
	case core.WidgetCheckbox:
		t.drawCheckbox(info, value, content, checked)
	case core.WidgetSlider:
		t.drawSlider(info, content, amount)
	case core.WidgetProgressBar:
		t.drawProgressBar(info, content, amount)
	default:
		t.drawTextInContent(info.Kind, value, content, info.State)
	}
}

// DrawDropdownPopup renders a dropdown list with the gallery's popup spacing,
// hover highlight, and themed text. info.Bounds is the complete popup bounds.
func (t *Theme) DrawDropdownPopup(info core.WidgetInfo, items []string, hovered int) {
	if t == nil || len(items) == 0 || info.Bounds.H <= 0 {
		return
	}
	t.DrawWidget(info, "", 0, false)
	t.DrawWidgetPart(info.Kind, skin.PartBorder, info.Bounds, info.State)
	rowHeight := info.Bounds.H / float32(len(items))
	for index, item := range items {
		row := core.Rect{X: info.Bounds.X, Y: info.Bounds.Y + float32(index)*rowHeight, W: info.Bounds.W, H: rowHeight}
		if index == hovered {
			t.drawDropdownHighlight(info, row)
		}
		t.drawDropdownText(info, row, item)
	}
}

// drawDropdownHighlight records and draws the fixed popup row highlight.
func (t *Theme) drawDropdownHighlight(info core.WidgetInfo, row core.Rect) {
	destination := core.Rect{X: row.X + 4, Y: row.Y + 3, W: row.W - 8, H: row.H - 6}
	tint := color.RGBA{R: 67, G: 97, B: 139, A: 255}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateHovered, row, destination, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		drawFallbackPart(destination, tint)
	}
}

// drawDropdownText records and draws one popup label with the gallery colors.
func (t *Theme) drawDropdownText(info core.WidgetInfo, row core.Rect, value string) {
	tint := color.RGBA{R: 230, G: 240, B: 255, A: 255}
	t.logDrawCall(info.Kind, skin.PartText, info.State, row, row, skin.SkinDescriptor{}, tint, false)
	if !rl.IsWindowReady() {
		return
	}
	if t.HasFont() {
		rl.DrawTextEx(t.FontForSize(16), value, rl.NewVector2(row.X+16, row.Y+8), 16, 1.6, tint)
		return
	}
	rl.DrawText(value, int32(row.X+16), int32(row.Y+8), 16, tint)
}

// drawCheckbox renders its icon state followed by its optional text.
func (t *Theme) drawCheckbox(info core.WidgetInfo, value string, content core.Rect, checked bool) {
	if checked {
		t.drawCheckmark(info, content)
	} else {
		t.drawUncheckedBox(info, content)
	}
	t.drawTextInContent(info.Kind, value, content, info.State)
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

// drawGeometryCheckmark renders the fallback checkmark and records its color.
func (t *Theme) drawGeometryCheckmark(info core.WidgetInfo, missingSkin bool) {
	destination := t.snap(info.Bounds)
	checkColor := color.RGBA{R: 20, G: 120, B: 60, A: 255}
	if missingSkin {
		checkColor = color.RGBA{A: 255}
	}
	if rl.IsWindowReady() {
		x, y, width, height := destination.X, destination.Y, destination.W, destination.H
		p1 := rl.NewVector2(x+width*0.25, y+height*0.55)
		p2 := rl.NewVector2(x+width*0.40, y+height*0.70)
		p3 := rl.NewVector2(x+width*0.75, y+height*0.30)
		rl.DrawLineEx(p1, p2, 2.5, checkColor)
		rl.DrawLineEx(p2, p3, 2.5, checkColor)
	}
	t.logDrawCall(info.Kind, skin.PartCheckmark, info.State, destination, destination, skin.SkinDescriptor{}, checkColor, true)
}

// drawSlider renders the track and thumb at amount's clamped position.
func (t *Theme) drawSlider(info core.WidgetInfo, content core.Rect, amount float32) {
	track, fallback := t.resolveDescriptor(info.Kind, skin.PartTrack, info.State)
	trackTint := effectiveTint(track, fallback)
	trackRect := sliderTrackRect(t, info, content, track, fallback)
	t.logDrawCall(info.Kind, skin.PartTrack, info.State, trackRect, trackRect, track, trackTint, fallback)
	if rl.IsWindowReady() {
		drawTexturedPart(track, trackRect, trackTint)
	}

	thumb, thumbFallback := t.resolveDescriptor(info.Kind, skin.PartThumb, info.State)
	thumbTint := effectiveTint(thumb, thumbFallback)
	thumbRect := sliderThumbRect(t, trackRect, thumb, thumbFallback, amount)
	t.logDrawCall(info.Kind, skin.PartThumb, info.State, thumbRect, thumbRect, thumb, thumbTint, thumbFallback)
	if rl.IsWindowReady() {
		drawTexturedPart(thumb, thumbRect, thumbTint)
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

// drawProgressBar renders a track and a proportional overlay.
func (t *Theme) drawProgressBar(info core.WidgetInfo, content core.Rect, amount float32) {
	track, fallback := t.resolveDescriptor(info.Kind, skin.PartTrack, info.State)
	trackTint := effectiveTint(track, fallback)
	trackRect := content
	if trackRect.W == 0 || trackRect.H == 0 {
		trackRect = t.snap(info.Bounds)
	}
	t.logDrawCall(info.Kind, skin.PartTrack, info.State, trackRect, trackRect, track, trackTint, fallback)
	if rl.IsWindowReady() {
		drawTexturedPart(track, trackRect, trackTint)
	}

	amount = clamp01(amount)
	fill, fillFallback := t.resolveDescriptor(info.Kind, skin.PartOverlay, info.State)
	fillTint := effectiveTint(fill, fillFallback)
	fillWidth := trackRect.W * amount
	if fillWidth <= 0 {
		return
	}
	fillRect := t.snap(core.Rect{X: trackRect.X, Y: trackRect.Y, W: fillWidth, H: trackRect.H})
	t.logDrawCall(info.Kind, skin.PartOverlay, info.State, fillRect, fillRect, fill, fillTint, fillFallback)
	if rl.IsWindowReady() {
		drawTexturedPart(fill, fillRect, fillTint)
	}
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
