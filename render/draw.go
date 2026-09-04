package render

import (
	"errors"
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	"rtgui/core"
	"rtgui/skin"
)

func effectiveTint(descriptor skin.SkinDescriptor, fallback bool) color.RGBA {
	if fallback {
		return color.RGBA{R: 200, G: 200, B: 200, A: 255}
	}
	base := normalizedTint(descriptor.Tint)
	finalAlpha := float32(base.A) * normalizedAlpha(descriptor.Alpha)
	if finalAlpha < 0 {
		finalAlpha = 0
	}
	if finalAlpha > 255 {
		finalAlpha = 255
	}
	base.A = uint8(finalAlpha + 0.5)
	return base
}

func normalizedAlpha(alpha float32) float32 {
	if alpha == 0 {
		return 1
	}
	if alpha < 0 {
		return 0
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}

func normalizedTint(tint core.Color) color.RGBA {
	if tint.R == 0 && tint.G == 0 && tint.B == 0 && tint.A == 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return color.RGBA{R: tint.R, G: tint.G, B: tint.B, A: tint.A}
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

func (t *Theme) logDrawCall(kind core.WidgetKind, part skin.SkinPart, state core.WidgetState, bounds, dest core.Rect, descriptor skin.SkinDescriptor, tint color.RGBA, fallback bool) {
	alpha := descriptor.Alpha
	if alpha == 0 {
		alpha = 1
	}
	t.drawLog = append(t.drawLog, DrawCall{
		Kind:     kind,
		Part:     part,
		State:    state,
		Bounds:   bounds,
		Dest:     dest,
		Src:      descriptor.AtlasRegion,
		Tint:     core.Color{R: tint.R, G: tint.G, B: tint.B, A: tint.A},
		Alpha:    alpha,
		Fallback: fallback,
	})
}

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

func drawNinePatch(texture rl.Texture2D, source core.Rect, descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA) {
	sourceRects := NinePatchSourceRects(source, descriptor.NinePatch)
	destinationRects := NinePatchRects(source, NinePatchConfig{
		Source: source,
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
	x := int32(math.Round(float64(content.X + 6)))
	y := int32(math.Round(float64(content.Y + (content.H-float32(fontSize))/2)))
	t.logDrawCall(kind, skin.PartText, state, content, content, skin.SkinDescriptor{}, textColor, false)
	if rl.IsWindowReady() {
		rl.DrawText(value, x, y, fontSize, textColor)
	}
}

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
func (t *Theme) DrawWidgetPart(kind core.WidgetKind, part skin.SkinPart, bounds core.Rect, state core.WidgetState) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	t.drawPart(kind, part, bounds, state)
	return nil
}

// DrawWidget renders a widget using the theme's registry and transform.
func (t *Theme) DrawWidget(info core.WidgetInfo, value string, amount float32, checked bool) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	background, _ := t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
	t.lastWidgetInfo = info
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
	return nil
}

func (t *Theme) drawCheckbox(info core.WidgetInfo, value string, content core.Rect, checked bool) {
	if checked {
		t.drawCheckmark(info, content)
	}
	t.drawTextInContent(info.Kind, value, content, info.State)
}

func (t *Theme) drawCheckmark(info core.WidgetInfo, content core.Rect) {
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartCheckmark, info.State)
	if hasTexture(descriptor, fallback) {
		t.drawTexturedCheckmark(info, content, descriptor)
		return
	}
	t.drawGeometryCheckmark(info, fallback)
}

func hasTexture(descriptor skin.SkinDescriptor, fallback bool) bool {
	return !fallback && descriptor.HasTexture && descriptor.Texture.ID != 0
}

func (t *Theme) drawTexturedCheckmark(info core.WidgetInfo, content core.Rect, descriptor skin.SkinDescriptor) {
	tint := effectiveTint(descriptor, false)
	width, height := checkmarkSize(descriptor, content)
	destination := t.snap(core.Rect{
		X: content.X + (content.W-width)/2,
		Y: content.Y + (content.H-height)/2,
		W: width,
		H: height,
	})
	t.logDrawCall(info.Kind, skin.PartCheckmark, info.State, destination, destination, descriptor, tint, false)
	if !rl.IsWindowReady() {
		return
	}
	texture := toRaylibTexture(descriptor.Texture)
	source := atlasRegion(descriptor, texture)
	drawSingleTexture(texture, source, destination, tint)
}

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

func (t *Theme) drawGeometryCheckmark(info core.WidgetInfo, missingSkin bool) {
	destination := t.snap(info.Bounds)
	if rl.IsWindowReady() {
		checkColor := color.RGBA{R: 20, G: 120, B: 60, A: 255}
		if missingSkin {
			checkColor = color.RGBA{A: 255}
		}
		x, y, width, height := destination.X, destination.Y, destination.W, destination.H
		p1 := rl.NewVector2(x+width*0.25, y+height*0.55)
		p2 := rl.NewVector2(x+width*0.40, y+height*0.70)
		p3 := rl.NewVector2(x+width*0.75, y+height*0.30)
		rl.DrawLineEx(p1, p2, 2.5, checkColor)
		rl.DrawLineEx(p2, p3, 2.5, checkColor)
	}
	t.logDrawCall(info.Kind, skin.PartCheckmark, info.State, destination, destination, skin.SkinDescriptor{}, color.RGBA{}, true)
}

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

func DebugBounds(call DrawCall) DebugInfo {
	return DebugInfo{
		Bounds:   call.Bounds,
		Fallback: call.Fallback,
		SkinKey:  skin.SkinKey{Widget: call.Kind, Part: call.Part, State: call.State},
	}
}

func EstimateDrawCalls(calls []DrawCall) (drawCalls int, textureSwitches int) {
	if len(calls) == 0 {
		return 0, 0
	}
	drawCalls = len(calls)
	lastSource := calls[0].Src
	for _, call := range calls[1:] {
		if call.Src != lastSource {
			textureSwitches++
			lastSource = call.Src
		}
	}
	return drawCalls, textureSwitches
}
