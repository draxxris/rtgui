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
// exact RGBA tint or solid background color when no texture is present.
func effectiveTint(descriptor skin.SkinDescriptor, fallback bool) color.RGBA {
	if fallback {
		return color.RGBA{R: 200, G: 200, B: 200, A: 255}
	}
	if !descriptor.HasTexture && descriptor.HasBackgroundColor {
		return descriptor.BackgroundColor.RGBA()
	}
	return descriptor.Tint.RGBA()
}

func (t *Theme) snap(rect core.Rect) core.Rect {
	if t == nil || !t.GetPixelSnap() {
		return rect
	}
	return t.transform.SnapRect(rect)
}

func (t *Theme) resolveDescriptor(kind core.WidgetKind, part skin.SkinPart, state core.WidgetState, class ...string) (skin.SkinDescriptor, bool) {
	descriptor, ok := t.Lookup(kind, part, state, class...)
	return descriptor, !ok
}

// logDrawCall records one operation only when diagnostics are explicitly enabled.
func (t *Theme) logDrawCall(kind core.WidgetKind, part skin.SkinPart, state core.WidgetState, bounds, dest core.Rect, descriptor skin.SkinDescriptor, tint color.RGBA, fallback bool) {
	if t == nil {
		return
	}
	if fallback && t.diagnosticHandler != nil {
		t.warnMissing(skin.SkinKey{Widget: kind, Part: part, State: state})
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

// warnMissing reports each missing skin key once, retaining at most 128 keys.
func (t *Theme) warnMissing(key skin.SkinKey) {
	for _, old := range t.warned {
		if old == key {
			return
		}
	}
	if len(t.warned) >= 128 {
		return
	}
	t.warned = append(t.warned, key)
	t.diagnosticHandler(fmt.Sprintf("render: missing skin for widget=%v part=%v state=%v", key.Widget, key.Part, key.State))
}

// drawTexturedPart selects simple, 3-patch, or nine-patch drawing for descriptor.
// Descriptors without a texture draw nothing; missing skins stay invisible
// (text still draws via drawTextInContent) and are reported via Fallback logs.
func drawTexturedPart(descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA) {
	if !descriptor.HasTexture || descriptor.Texture.ID == 0 {
		return
	}
	texture := toRaylibTexture(descriptor.Texture)
	source := atlasRegion(descriptor, texture)
	if descriptor.HasThreePatch {
		drawThreePatch(texture, source, descriptor, dest, tint)
		return
	}
	if !hasNinePatchBorders(descriptor) {
		drawSingleTexture(texture, source, dest, tint)
		return
	}
	drawNinePatch(texture, source, descriptor, dest, tint)
}

// drawThreePatch renders source across the three vertical destination rectangles (top, middle, bottom).
func drawThreePatch(texture rl.Texture2D, source core.Rect, descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA) {
	sourceRects := ThreePatchSourceRects(source, descriptor.ThreePatch)
	destRects := ThreePatchRects(ThreePatchConfig{
		Top:    float32(descriptor.ThreePatch.Top),
		Bottom: float32(descriptor.ThreePatch.Bottom),
	}, dest)
	for i := range destRects {
		destination := destRects[i]
		sourceRect := sourceRects[i]
		if destination.W > 0 && destination.H > 0 && sourceRect.W > 0 && sourceRect.H > 0 {
			drawSingleTexture(texture, sourceRect, destination, tint)
		}
	}
}

func drawFallbackPart(dest core.Rect, tint color.RGBA) {
	if !rl.IsWindowReady() {
		return
	}
	rl.DrawRectangleRec(toRaylibRect(dest), tint)
	rl.DrawRectangleLinesEx(toRaylibRect(dest), 1, color.RGBA{R: 255, G: 0, B: 255, A: 255})
}

// drawDescriptorBackground renders the solid color, gradient stack, and
// texture layers of descriptor into dest. Layers composite back-to-front so
// multiple directions and centers blend like the reference tooltip. A
// declared border-radius clips the background layers to a rounded rectangle
// while leaving the border texture square.
func drawDescriptorBackground(descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA) {
	if radius := effectiveBackgroundRadius(descriptor, dest); radius > 0 {
		drawRoundedBackground(descriptor, dest, tint, radius)
		return
	}
	if descriptor.HasBackgroundColor && descriptor.BackgroundColor.A > 0 {
		rl.DrawRectangleRec(toRaylibRect(dest), descriptor.BackgroundColor.RGBA())
	}
	for i := descriptor.GradientCount - 1; i >= 0; i-- {
		drawGradient(descriptor.Gradients[i], dest)
	}
	if descriptor.HasTexture && descriptor.Texture.ID != 0 {
		drawTexturedPart(descriptor, dest, tint)
	}
}

// hasVisualBackground reports whether descriptor holds a drawable layer.
func hasVisualBackground(descriptor skin.SkinDescriptor) bool {
	return (descriptor.HasBackgroundColor && descriptor.BackgroundColor.A > 0) ||
		descriptor.HasGradient() ||
		(descriptor.HasTexture && descriptor.Texture.ID != 0)
}

// drawGradient dispatches one gradient layer to its linear or radial path.
func drawGradient(grad skin.Gradient, dest core.Rect) {
	if dest.W <= 0 || dest.H <= 0 || grad.StopCount < 2 {
		return
	}
	if !rl.IsWindowReady() {
		return
	}
	if grad.Kind == skin.GradientRadial {
		drawRadialGradient(grad, dest)
		return
	}
	if grad.Kind == skin.GradientInner {
		drawInnerGradient(grad, dest)
		return
	}
	drawLinearGradient(grad, dest)
}

// drawLinearGradient renders a linear gradient with stop positions honored.
// Two-stop edge-to-edge fills use one quad; cardinal multi-stops slice into
// exact strips; diagonal and angled multi-stops use a banded mesh.
func drawLinearGradient(grad skin.Gradient, dest core.Rect) {
	if isSingleQuadGradient(grad) {
		tl, bl, br, tr := linearQuadCorners(grad)
		rl.DrawRectangleGradientEx(toRaylibRect(dest), tl, bl, br, tr)
		return
	}
	if isCardinalGradient(grad) {
		drawCardinalStrips(grad, dest)
		return
	}
	drawLinearBandedMesh(grad, dest)
}

// isSingleQuadGradient reports whether one quad reproduces the gradient.
// Two edge-to-edge stops are linear in (u, v) for any direction or angle.
func isSingleQuadGradient(grad skin.Gradient) bool {
	return grad.StopCount == 2 && grad.Stops[0].Position <= 0 && grad.Stops[1].Position >= 1
}

// linearQuadCorners samples the four destination corners of a linear fill.
func linearQuadCorners(grad skin.Gradient) (topLeft, bottomLeft, bottomRight, topRight color.RGBA) {
	return sampleLinearRGBA(grad, 0, 0),
		sampleLinearRGBA(grad, 0, 1),
		sampleLinearRGBA(grad, 1, 1),
		sampleLinearRGBA(grad, 1, 0)
}

// sampleLinearRGBA samples a linear gradient at normalized (u, v).
func sampleLinearRGBA(grad skin.Gradient, u, v float32) color.RGBA {
	return skin.SampleLinearAt(grad, u, v).RGBA()
}

// isCardinalGradient reports a horizontal or vertical fill without an angle.
func isCardinalGradient(grad skin.Gradient) bool {
	if grad.UseAngle {
		return false
	}
	switch grad.Direction {
	case skin.GradientToBottom, skin.GradientToTop, skin.GradientToRight, skin.GradientToLeft:
		return true
	default:
		return false
	}
}

// isVerticalGradient reports a bottom or top fill without an angle.
func isVerticalGradient(grad skin.Gradient) bool {
	return !grad.UseAngle && (grad.Direction == skin.GradientToBottom || grad.Direction == skin.GradientToTop)
}

// drawCardinalStrips slices a multi-stop cardinal fill into exact strips.
func drawCardinalStrips(grad skin.Gradient, dest core.Rect) {
	if isVerticalGradient(grad) {
		drawVerticalStrips(grad, dest)
		return
	}
	drawHorizontalStrips(grad, dest)
}

// drawVerticalStrips renders a vertical multi-stop fill as horizontal bands.
// Boundaries come from stop positions mapped to y and sorted ascending; each
// band samples its edge colors so hard stops and mid fades stay exact.
func drawVerticalStrips(grad skin.Gradient, dest core.Rect) {
	var bounds [skin.MaxGradientStops + 2]float32
	n := verticalBounds(grad, dest, &bounds)
	for i := 0; i+1 < n; i++ {
		y0, y1 := bounds[i], bounds[i+1]
		if y1 <= y0 {
			continue
		}
		v0 := (y0 - dest.Y) / dest.H
		v1 := (y1 - dest.Y) / dest.H
		top := sampleLinearRGBA(grad, 0.5, v0)
		bottom := sampleLinearRGBA(grad, 0.5, v1)
		band := core.Rect{X: dest.X, Y: y0, W: dest.W, H: y1 - y0}
		rl.DrawRectangleGradientEx(toRaylibRect(band), top, bottom, bottom, top)
	}
}

// verticalBounds collects sorted y boundaries from stop positions.
func verticalBounds(grad skin.Gradient, dest core.Rect, bounds *[skin.MaxGradientStops + 2]float32) int {
	n := 0
	bounds[n], n = dest.Y, n+1
	for i := 0; i < grad.StopCount; i++ {
		bounds[n], n = stopY(grad, dest, i), n+1
	}
	bounds[n], n = dest.Y+dest.H, n+1
	sortAscending(bounds, n)
	return n
}

// stopY maps one stop position to its y for the gradient direction.
func stopY(grad skin.Gradient, dest core.Rect, index int) float32 {
	p := grad.Stops[index].Position
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	if grad.Direction == skin.GradientToTop && !grad.UseAngle {
		return dest.Y + (1-p)*dest.H
	}
	return dest.Y + p*dest.H
}

// drawHorizontalStrips renders a horizontal multi-stop fill as vertical bands.
func drawHorizontalStrips(grad skin.Gradient, dest core.Rect) {
	var bounds [skin.MaxGradientStops + 2]float32
	n := horizontalBounds(grad, dest, &bounds)
	for i := 0; i+1 < n; i++ {
		x0, x1 := bounds[i], bounds[i+1]
		if x1 <= x0 {
			continue
		}
		u0 := (x0 - dest.X) / dest.W
		u1 := (x1 - dest.X) / dest.W
		left := sampleLinearRGBA(grad, u0, 0.5)
		right := sampleLinearRGBA(grad, u1, 0.5)
		band := core.Rect{X: x0, Y: dest.Y, W: x1 - x0, H: dest.H}
		rl.DrawRectangleGradientEx(toRaylibRect(band), left, left, right, right)
	}
}

// horizontalBounds collects sorted x boundaries from stop positions.
func horizontalBounds(grad skin.Gradient, dest core.Rect, bounds *[skin.MaxGradientStops + 2]float32) int {
	n := 0
	bounds[n], n = dest.X, n+1
	for i := 0; i < grad.StopCount; i++ {
		bounds[n], n = stopX(grad, dest, i), n+1
	}
	bounds[n], n = dest.X+dest.W, n+1
	sortAscending(bounds, n)
	return n
}

// stopX maps one stop position to its x for the gradient direction.
func stopX(grad skin.Gradient, dest core.Rect, index int) float32 {
	p := grad.Stops[index].Position
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	if grad.Direction == skin.GradientToLeft && !grad.UseAngle {
		return dest.X + (1-p)*dest.W
	}
	return dest.X + p*dest.W
}

// sortAscending bubble-sorts the used prefix of bounds in place.
func sortAscending(bounds *[skin.MaxGradientStops + 2]float32, n int) {
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if bounds[j] < bounds[i] {
				bounds[i], bounds[j] = bounds[j], bounds[i]
			}
		}
	}
}

// linearBandMaxRows caps the tessellation of angled multi-stop fills.
// Rows stay at one pixel for tooltip-scale widgets and coarsen for tall
// panels, keeping vertex emission bounded per frame.
const linearBandMaxRows = 128

// drawLinearBandedMesh renders diagonal and angled multi-stop fills.
// Capped rows batch into a single triangle mesh with per-vertex colors,
// so kinks between stops stay smooth without one draw call per row.
func drawLinearBandedMesh(grad skin.Gradient, dest core.Rect) {
	u0, v0, u1, v1, ready := beginGradientMesh()
	if !ready {
		return
	}
	rows := int(dest.H + 0.5)
	if rows < 1 {
		rows = 1
	}
	if rows > linearBandMaxRows {
		rows = linearBandMaxRows
	}
	for i := 0; i < rows; i++ {
		y0 := dest.Y + float32(i)*dest.H/float32(rows)
		y1 := dest.Y + float32(i+1)*dest.H/float32(rows)
		emitLinearBand(grad, dest, y0, y1, u0, v0, u1, v1)
	}
	endGradientMesh()
}

// emitLinearBand emits one band quad of a diagonal fill with sampled corners.
func emitLinearBand(grad skin.Gradient, dest core.Rect, y0, y1, u0, v0, u1, v1 float32) {
	w0 := (y0 - dest.Y) / dest.H
	w1 := (y1 - dest.Y) / dest.H
	emitGradientQuad(dest.X, y0, dest.X+dest.W, y1,
		sampleLinearRGBA(grad, 0, w0), sampleLinearRGBA(grad, 0, w1),
		sampleLinearRGBA(grad, 1, w1), sampleLinearRGBA(grad, 1, w0),
		u0, v0, u1, v1)
}

// radialGridDivisions bounds the tessellation of radial fills.
const radialGridDivisions = 12

// drawRadialGradient renders a radial fill as a batched color mesh.
// The center maps to position 0 and the farthest unit-square corner to 1,
// so stop percentages behave like CSS radial stop lists.
func drawRadialGradient(grad skin.Gradient, dest core.Rect) {
	u0, v0, u1, v1, ready := beginGradientMesh()
	if !ready {
		return
	}
	maxDist := skin.RadialMaxDist(grad.CenterX, grad.CenterY)
	steps := radialGridDivisions
	for j := 0; j < steps; j++ {
		for i := 0; i < steps; i++ {
			emitRadialCell(grad, dest, i, j, steps, maxDist, u0, v0, u1, v1)
		}
	}
	endGradientMesh()
}

// drawInnerGradient renders an inner fill as a batched color mesh. Every
// border maps to stop position 0 and the center to 1, so translucent edge
// colors glow inward from all four sides at once.
func drawInnerGradient(grad skin.Gradient, dest core.Rect) {
	u0, v0, u1, v1, ready := beginGradientMesh()
	if !ready {
		return
	}
	steps := radialGridDivisions
	for j := 0; j < steps; j++ {
		for i := 0; i < steps; i++ {
			emitInnerCell(grad, dest, i, j, steps, u0, v0, u1, v1)
		}
	}
	endGradientMesh()
}

// emitInnerCell emits one grid cell of an inner fill with sampled corners.
func emitInnerCell(grad skin.Gradient, dest core.Rect, i, j, steps int, u0, v0, u1, v1 float32) {
	nu0 := float32(i) / float32(steps)
	nu1 := float32(i+1) / float32(steps)
	nv0 := float32(j) / float32(steps)
	nv1 := float32(j+1) / float32(steps)
	emitGradientQuad(dest.X+nu0*dest.W, dest.Y+nv0*dest.H, dest.X+nu1*dest.W, dest.Y+nv1*dest.H,
		sampleInnerRGBA(grad, nu0, nv0), sampleInnerRGBA(grad, nu0, nv1),
		sampleInnerRGBA(grad, nu1, nv1), sampleInnerRGBA(grad, nu1, nv0),
		u0, v0, u1, v1)
}

// sampleInnerRGBA samples an inner gradient at normalized (u, v).
func sampleInnerRGBA(grad skin.Gradient, u, v float32) color.RGBA {
	return skin.SampleInnerAt(grad, u, v).RGBA()
}

// emitRadialCell emits one grid cell of a radial fill with sampled corners.
func emitRadialCell(grad skin.Gradient, dest core.Rect, i, j, steps int, maxDist, u0, v0, u1, v1 float32) {
	nu0 := float32(i) / float32(steps)
	nu1 := float32(i+1) / float32(steps)
	nv0 := float32(j) / float32(steps)
	nv1 := float32(j+1) / float32(steps)
	emitGradientQuad(dest.X+nu0*dest.W, dest.Y+nv0*dest.H, dest.X+nu1*dest.W, dest.Y+nv1*dest.H,
		sampleRadialRGBA(grad, nu0, nv0, maxDist), sampleRadialRGBA(grad, nu0, nv1, maxDist),
		sampleRadialRGBA(grad, nu1, nv1, maxDist), sampleRadialRGBA(grad, nu1, nv0, maxDist),
		u0, v0, u1, v1)
}

// sampleRadialRGBA samples a radial gradient with a precomputed max distance.
func sampleRadialRGBA(grad skin.Gradient, u, v, maxDist float32) color.RGBA {
	return skin.SampleRadialAt(grad, u, v, maxDist).RGBA()
}

// beginGradientMesh binds the white shapes pixel and opens a triangle batch.
// It reports ready=false without touching GL headlessly, matching the fill
// mesh used below the line graph widget.
func beginGradientMesh() (u0, v0, u1, v1 float32, ready bool) {
	if !rl.IsWindowReady() {
		return 0, 0, 0, 0, false
	}
	shapeTex := rl.GetShapesTexture()
	shapeRec := rl.GetShapesTextureRectangle()
	if shapeTex.Width > 0 && shapeTex.Height > 0 {
		u0 = shapeRec.X / float32(shapeTex.Width)
		v0 = shapeRec.Y / float32(shapeTex.Height)
		u1 = (shapeRec.X + shapeRec.Width) / float32(shapeTex.Width)
		v1 = (shapeRec.Y + shapeRec.Height) / float32(shapeTex.Height)
	}
	rl.SetTexture(shapeTex.ID)
	rl.Begin(rl.Triangles)
	return u0, v0, u1, v1, true
}

// endGradientMesh closes the batch opened by beginGradientMesh.
func endGradientMesh() {
	rl.End()
	rl.SetTexture(0)
}

// emitGradientQuad emits two counter-clockwise triangles for an axis quad.
// Corner colors run top-left, bottom-left, bottom-right, top-right, matching
// the winding raylib shapes use while culling stays enabled.
func emitGradientQuad(x0, y0, x1, y1 float32, tl, bl, br, tr color.RGBA, u0, v0, u1, v1 float32) {
	rl.TexCoord2f(u1, v1)
	rl.Color4ub(br.R, br.G, br.B, br.A)
	rl.Vertex2f(x1, y1)
	rl.TexCoord2f(u1, v0)
	rl.Color4ub(tr.R, tr.G, tr.B, tr.A)
	rl.Vertex2f(x1, y0)
	rl.TexCoord2f(u0, v0)
	rl.Color4ub(tl.R, tl.G, tl.B, tl.A)
	rl.Vertex2f(x0, y0)
	rl.TexCoord2f(u0, v1)
	rl.Color4ub(bl.R, bl.G, bl.B, bl.A)
	rl.Vertex2f(x0, y1)
	rl.TexCoord2f(u1, v1)
	rl.Color4ub(br.R, br.G, br.B, br.A)
	rl.Vertex2f(x1, y1)
	rl.TexCoord2f(u0, v0)
	rl.Color4ub(tl.R, tl.G, tl.B, tl.A)
	rl.Vertex2f(x0, y0)
}

// renderDescriptorOrFallback renders descriptor visuals or a debug fallback box.
func (t *Theme) renderDescriptorOrFallback(descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA, fallback bool, fallbackAlpha uint8) {
	if !rl.IsWindowReady() {
		return
	}
	if hasVisualBackground(descriptor) {
		drawDescriptorBackground(descriptor, dest, tint)
	} else if fallback && t != nil && t.debugMode {
		drawFallbackPart(dest, color.RGBA{R: 255, G: 0, B: 255, A: fallbackAlpha})
	}
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
	if fontSize <= 0 && t != nil {
		if desc, ok := t.Lookup(info.Kind, skin.PartBackground, info.State, info.Class); ok && desc.HasFontSize && desc.FontSize > 0 {
			fontSize = desc.FontSize
		}
	}
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
	desc, _ := t.Lookup(info.Kind, skin.PartBackground, state, info.Class)
	textColor := t.resolveWidgetTextColor(info, state, desc)
	t.logDrawCall(info.Kind, skin.PartText, state, content, content, skin.SkinDescriptor{}, textColor, false)
	if desc.HasFont && t != nil && t.HasNamedFont(desc.Font) && rl.IsWindowReady() {
		face := t.namedFonts[desc.Font]
		rl.DrawTextEx(face.forSize(fontSize), value, rl.NewVector2(x, y), fontSize, fontSize/10, textColor)
		return
	}
	t.DrawText(value, x, y, fontSize, info.Italic, textColor)
}

// resolveWidgetTextColor resolves the text tint from explicit widget color, CSS color, or state default.
func (t *Theme) resolveWidgetTextColor(info core.WidgetInfo, state core.WidgetState, desc skin.SkinDescriptor) color.RGBA {
	if info.HasTextColor {
		base := info.TextColor.RGBA()
		if state == core.StateDisabled {
			return color.RGBA{R: base.R / 2, G: base.G / 2, B: base.B / 2, A: base.A}
		}
		return base
	}
	if desc.HasTextColor {
		base := desc.TextColor.RGBA()
		if state == core.StateDisabled {
			return color.RGBA{R: base.R / 2, G: base.G / 2, B: base.B / 2, A: base.A}
		}
		return base
	}
	return defaultWidgetTextColor(info, state)
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
func (t *Theme) drawPart(kind core.WidgetKind, part skin.SkinPart, bounds core.Rect, state core.WidgetState, class ...string) (skin.SkinDescriptor, bool) {
	descriptor, fallback := t.resolveDescriptor(kind, part, state, class...)
	tint := effectiveTint(descriptor, fallback)
	destination := t.snap(bounds)
	t.logDrawCall(kind, part, state, bounds, destination, descriptor, tint, fallback)
	t.renderDescriptorOrFallback(descriptor, destination, tint, fallback, 100)
	return descriptor, fallback
}

// DrawWidgetPart draws one part of a widget.
func (t *Theme) DrawWidgetPart(kind core.WidgetKind, part skin.SkinPart, bounds core.Rect, state core.WidgetState, class ...string) {
	if t != nil {
		t.drawPart(kind, part, bounds, state, class...)
	}
}

// DrawWidget renders a widget using the theme's registry and transform.
// It covers the text-only case; DrawControl adds single-line rich runs.
func (t *Theme) DrawWidget(info core.WidgetInfo, value string, amount float32, checked bool) {
	t.DrawControl(info, value, nil, amount, checked)
}

// DrawControl renders one control with plain text or single-line rich runs.
// A non-empty segments slice draws rich runs with the control alignment;
// links render in registered colors without underlines or activation.
// Otherwise value draws as plain text. Checkbox has no background by design
// (::box is PartIcon, ::checkmark is PartCheckmark); its content is the full
// bounds so missing art stays invisible.
func (t *Theme) DrawControl(info core.WidgetInfo, value string, segments []core.RichSegment, amount float32, checked bool) {
	if t == nil {
		return
	}
	content := t.controlContentRect(info.Kind, info.Bounds, info.State, info.Class)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}

	switch info.Kind {
	case core.WidgetCheckbox:
		t.drawControlCheckbox(info, value, segments, content, checked)
	case core.WidgetSlider:
		t.drawSlider(info, value, content, amount)
	case core.WidgetProgressBar:
		t.drawProgressBar(info, value, content, amount)
	default:
		if len(segments) > 0 {
			t.DrawRichSingleLine(info, content, segments, info.Align)
		} else {
			t.drawTextInContent(info, value, content, info.State)
		}
	}
}

// drawControlCheckbox renders the box with a plain or rich single-line label.
// An empty value without segments draws the lone box, matching drawCheckbox.
func (t *Theme) drawControlCheckbox(info core.WidgetInfo, value string, segments []core.RichSegment, content core.Rect, checked bool) {
	if value == "" && len(segments) == 0 {
		if checked {
			t.drawCheckmark(info, content)
		} else {
			t.drawUncheckedBox(info, content)
		}
		return
	}
	iconRect, textRect := checkboxLayout(content)
	if checked {
		t.drawCheckmark(info, iconRect)
	} else {
		t.drawUncheckedBox(info, iconRect)
	}
	if len(segments) > 0 {
		if t.recorder != nil {
			t.recorder.setLastWidgetInfo(info)
		}
		t.DrawRichSingleLine(info, textRect, segments, info.Align)
		return
	}
	t.drawTextInContent(info, value, textRect, info.State)
}

// DrawDropdownPopup renders a dropdown list with popup skin parts and
// themed text. info.Bounds is the complete popup bounds. The popup fill is
// Dropdown::popup and its ring is the same selector's border-image; rows lay
// out inside their merged insets so the ring never overlaps row content.
func (t *Theme) DrawDropdownPopup(info core.WidgetInfo, items []string, hovered int) {
	t.DrawRichDropdownPopup(info, items, nil, hovered)
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
	iconRect, textRect := checkboxLayout(content)
	if checked {
		t.drawCheckmark(info, iconRect)
	} else {
		t.drawUncheckedBox(info, iconRect)
	}
	t.drawTextInContent(info, value, textRect, info.State)
}

// drawUncheckedBox renders the empty box icon when a textured PartIcon is
// registered for checkboxes. Without one it draws nothing, preserving the
// old no-skin behavior of an empty unchecked box.
func (t *Theme) drawUncheckedBox(info core.WidgetInfo, content core.Rect) {
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartIcon, info.State, info.Class)
	if !hasTexture(descriptor, fallback) {
		return
	}
	t.drawCenteredIcon(info, content, skin.PartIcon, descriptor)
}

func (t *Theme) drawCheckmark(info core.WidgetInfo, content core.Rect) {
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartCheckmark, info.State, info.Class)
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
	track, fallback := t.resolveDescriptor(info.Kind, skin.PartTrack, info.State, info.Class)
	trackTint := effectiveTint(track, fallback)
	trackRect := sliderTrackRect(t, info, content, track, fallback)
	t.logDrawCall(info.Kind, skin.PartTrack, info.State, trackRect, trackRect, track, trackTint, fallback)
	t.renderDescriptorOrFallback(track, trackRect, trackTint, fallback, 100)

	thumb, thumbFallback := t.resolveDescriptor(info.Kind, skin.PartThumb, info.State, info.Class)
	thumbTint := effectiveTint(thumb, thumbFallback)
	thumbRect := sliderThumbRect(t, trackRect, thumb, thumbFallback, amount)
	t.logDrawCall(info.Kind, skin.PartThumb, info.State, thumbRect, thumbRect, thumb, thumbTint, thumbFallback)
	t.renderDescriptorOrFallback(thumb, thumbRect, thumbTint, thumbFallback, 150)
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
	track, fallback := t.resolveDescriptor(info.Kind, skin.PartTrack, info.State, info.Class)
	trackTint := effectiveTint(track, fallback)
	trackRect := content
	if trackRect.W == 0 || trackRect.H == 0 {
		trackRect = t.snap(info.Bounds)
	}
	t.logDrawCall(info.Kind, skin.PartTrack, info.State, trackRect, trackRect, track, trackTint, fallback)
	t.renderDescriptorOrFallback(track, trackRect, trackTint, fallback, 100)

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
	fill, fillFallback := t.resolveDescriptor(info.Kind, skin.PartOverlay, info.State, info.Class)
	fillTint := effectiveTint(fill, fillFallback)
	fillWidth := trackRect.W * amount
	if fillWidth <= 0 {
		return 0
	}
	fillRect := t.snap(core.Rect{X: trackRect.X, Y: trackRect.Y, W: fillWidth, H: trackRect.H})
	t.logDrawCall(info.Kind, skin.PartOverlay, info.State, fillRect, fillRect, fill, fillTint, fillFallback)
	t.renderDescriptorOrFallback(fill, fillRect, fillTint, fillFallback, 150)
	return fillWidth
}

// drawProgressSpark renders the ::spark marker centered on the fill edge.
// Without a spark descriptor it draws nothing, preserving unskinned bars.
func (t *Theme) drawProgressSpark(info core.WidgetInfo, track core.Rect, fillWidth float32) {
	spark, fallback := t.resolveDescriptor(info.Kind, skin.PartSpark, info.State, info.Class)
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
