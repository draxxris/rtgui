package render

import (
	"image/color"
	"math"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Rounded background clipping keeps square background layers from peeking out
// from behind rounded border textures. A uniform border-radius clips the
// color, gradient, and single-texture layers to a rounded rectangle by tiling
// them into exact 1px corner rows plus one middle band; the rows partition
// dest with no gaps or overlaps, so rounded output matches the sharp layers
// exactly. Nine-patch and three-patch texture layers cannot tile this way and
// draw sharp (see drawRoundedBackground). Radius zero keeps the single-quad
// fast path, so unauthored skins pay nothing.

// effectiveBackgroundRadius returns the drawable corner radius for dest, or 0
// for square output. Values clamp to half the smaller side per CSS rules.
func effectiveBackgroundRadius(descriptor skin.SkinDescriptor, dest core.Rect) float32 {
	if !descriptor.HasRadius || descriptor.Radius <= 0 {
		return 0
	}
	if dest.W <= 0 || dest.H <= 0 {
		return 0
	}
	radius := float32(math.Floor(float64(descriptor.Radius)))
	limit := float32(math.Floor(float64(minFloat(dest.W, dest.H)) / 2))
	if radius > limit {
		radius = limit
	}
	return radius
}

// roundedBandCount returns the band tally for an integer radius: one 1px row
// per corner pixel plus the single full-width middle band.
func roundedBandCount(radius float32) int {
	return int(radius)*2 + 1
}

// roundedBand returns band index of count for dest clipped to radius. Bands
// below radius are top corner rows, the middle band keeps full width, and the
// rest mirror as bottom corner rows.
func roundedBand(dest core.Rect, radius float32, index, count int) core.Rect {
	middle := count / 2
	if index == middle {
		return core.Rect{X: dest.X, Y: dest.Y + radius, W: dest.W, H: dest.H - 2*radius}
	}
	row := index
	y := dest.Y + float32(index)
	if index > middle {
		row = count - 1 - index
		y = dest.Y + dest.H - float32(row) - 1
	}
	inset := roundedRowInset(radius, row)
	return core.Rect{X: dest.X + inset, Y: y, W: dest.W - 2*inset, H: 1}
}

// roundedRowInset returns the horizontal cut for a corner-zone row sampled at
// the row middle, so the staircase hugs the true arc from the inside.
func roundedRowInset(radius float32, row int) float32 {
	dy := radius - float32(row) - 0.5
	chord := float32(math.Sqrt(float64(maxFloat(0, radius*radius-dy*dy))))
	return radius - chord
}

// drawRoundedBackground renders descriptor layers clipped to radius. Color,
// the gradient stack, and single textures tile into bands; patched textures
// fall back to sharp drawing since their tiles cannot follow the arc.
func drawRoundedBackground(descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA, radius float32) {
	if descriptor.HasBackgroundColor && descriptor.BackgroundColor.A > 0 {
		drawRoundedColor(descriptor.BackgroundColor.RGBA(), dest, radius)
	}
	for i := descriptor.GradientCount - 1; i >= 0; i-- {
		drawRoundedGradient(descriptor.Gradients[i], dest, radius)
	}
	if descriptor.HasTexture && descriptor.Texture.ID != 0 {
		if hasNinePatchBorders(descriptor) || descriptor.HasThreePatch {
			drawTexturedPart(descriptor, dest, tint)
			return
		}
		texture := toRaylibTexture(descriptor.Texture)
		drawRoundedSingleTexture(texture, atlasRegion(descriptor, texture), dest, tint, radius)
	}
}

// drawRoundedColor tiles a solid fill into rounded bands.
func drawRoundedColor(fill color.RGBA, dest core.Rect, radius float32) {
	count := roundedBandCount(radius)
	for i := range count {
		band := roundedBand(dest, radius, i, count)
		if band.W > 0 && band.H > 0 {
			rl.DrawRectangleRec(toRaylibRect(band), fill)
		}
	}
}

// drawRoundedGradient tiles one gradient layer into rounded bands.
// Linear bands sample their own corners so two-stop fills match the sharp
// path exactly; radial bands subdivide into a batched mesh so the center
// highlight survives rounded clipping.
func drawRoundedGradient(grad skin.Gradient, dest core.Rect, radius float32) {
	if grad.StopCount < 2 || dest.W <= 0 || dest.H <= 0 {
		return
	}
	if grad.Kind == skin.GradientRadial {
		drawRoundedRadial(grad, dest, radius)
		return
	}
	if grad.Kind == skin.GradientInner {
		drawRoundedInner(grad, dest, radius)
		return
	}
	drawRoundedLinear(grad, dest, radius)
}

// drawRoundedLinear tiles a linear gradient into rounded bands.
// Corner rows draw as single quads; the tall middle band of a cardinal fill
// subdivides at stop boundaries so multi-stop kinks match the sharp path.
func drawRoundedLinear(grad skin.Gradient, dest core.Rect, radius float32) {
	count := roundedBandCount(radius)
	for i := range count {
		band := roundedBand(dest, radius, i, count)
		if band.W <= 0 || band.H <= 0 {
			continue
		}
		if i == count/2 && isCardinalGradient(grad) {
			emitRoundedMiddleStrips(grad, dest, band)
			continue
		}
		emitRoundedLinearBand(grad, dest, band)
	}
}

// emitRoundedMiddleStrips subdivides the tall middle band at stop edges.
// Each sub-strip samples its own corners, reproducing sharp-path kinks.
func emitRoundedMiddleStrips(grad skin.Gradient, dest, band core.Rect) {
	if isVerticalGradient(grad) {
		emitRoundedMiddleRows(grad, dest, band)
		return
	}
	emitRoundedMiddleColumns(grad, dest, band)
}

// emitRoundedMiddleRows slices a vertical middle band into horizontal strips.
func emitRoundedMiddleRows(grad skin.Gradient, dest, band core.Rect) {
	var bounds [skin.MaxGradientStops + 2]float32
	n := verticalBounds(grad, dest, &bounds)
	for i := 0; i+1 < n; i++ {
		y0 := maxFloat(bounds[i], band.Y)
		y1 := minFloat(bounds[i+1], band.Y+band.H)
		if y1 <= y0 {
			continue
		}
		emitRoundedLinearBand(grad, dest, core.Rect{X: band.X, Y: y0, W: band.W, H: y1 - y0})
	}
}

// emitRoundedMiddleColumns slices a horizontal middle band into vertical strips.
func emitRoundedMiddleColumns(grad skin.Gradient, dest, band core.Rect) {
	var bounds [skin.MaxGradientStops + 2]float32
	n := horizontalBounds(grad, dest, &bounds)
	for i := 0; i+1 < n; i++ {
		x0 := maxFloat(bounds[i], band.X)
		x1 := minFloat(bounds[i+1], band.X+band.W)
		if x1 <= x0 {
			continue
		}
		emitRoundedLinearBand(grad, dest, core.Rect{X: x0, Y: band.Y, W: x1 - x0, H: band.H})
	}
}

// emitRoundedLinearBand draws one rounded band with sampled corners.
func emitRoundedLinearBand(grad skin.Gradient, dest, band core.Rect) {
	u0 := (band.X - dest.X) / dest.W
	u1 := (band.X + band.W - dest.X) / dest.W
	v0 := (band.Y - dest.Y) / dest.H
	v1 := (band.Y + band.H - dest.Y) / dest.H
	rl.DrawRectangleGradientEx(toRaylibRect(band),
		sampleLinearRGBA(grad, u0, v0),
		sampleLinearRGBA(grad, u0, v1),
		sampleLinearRGBA(grad, u1, v1),
		sampleLinearRGBA(grad, u1, v0))
}

// roundedRadialColumns bounds the horizontal tessellation of radial bands.
const roundedRadialColumns = 12

// roundedRadialRows bounds the vertical slices of the tall middle band.
const roundedRadialRows = 12

// drawRoundedRadial tiles a radial gradient into a batched rounded mesh.
// Corner rows emit one strip of column cells each; the tall middle band
// splits into rows so vertical falloff stays smooth in a single batch.
func drawRoundedRadial(grad skin.Gradient, dest core.Rect, radius float32) {
	u0, v0, u1, v1, ready := beginGradientMesh()
	if !ready {
		return
	}
	maxDist := skin.RadialMaxDist(grad.CenterX, grad.CenterY)
	count := roundedBandCount(radius)
	for i := range count {
		band := roundedBand(dest, radius, i, count)
		if band.W <= 0 || band.H <= 0 {
			continue
		}
		if i == count/2 {
			emitRoundedRadialMiddle(grad, dest, band, maxDist, u0, v0, u1, v1)
			continue
		}
		emitRoundedRadialStrip(grad, dest, band, maxDist, u0, v0, u1, v1)
	}
	endGradientMesh()
}

// emitRoundedRadialMiddle splits the tall middle band into row slices.
func emitRoundedRadialMiddle(grad skin.Gradient, dest, band core.Rect, maxDist, u0, v0, u1, v1 float32) {
	for r := range roundedRadialRows {
		y0 := band.Y + float32(r)*band.H/float32(roundedRadialRows)
		y1 := band.Y + float32(r+1)*band.H/float32(roundedRadialRows)
		sub := core.Rect{X: band.X, Y: y0, W: band.W, H: y1 - y0}
		if sub.H <= 0 {
			continue
		}
		emitRoundedRadialStrip(grad, dest, sub, maxDist, u0, v0, u1, v1)
	}
}

// emitRoundedRadialStrip emits one band row as column cells with sampled corners.
func emitRoundedRadialStrip(grad skin.Gradient, dest, band core.Rect, maxDist, u0, v0, u1, v1 float32) {
	for c := range roundedRadialColumns {
		x0 := band.X + float32(c)*band.W/float32(roundedRadialColumns)
		x1 := band.X + float32(c+1)*band.W/float32(roundedRadialColumns)
		if x1 <= x0 {
			continue
		}
		nu0 := (x0 - dest.X) / dest.W
		nu1 := (x1 - dest.X) / dest.W
		nv0 := (band.Y - dest.Y) / dest.H
		nv1 := (band.Y + band.H - dest.Y) / dest.H
		emitGradientQuad(x0, band.Y, x1, band.Y+band.H,
			sampleRadialRGBA(grad, nu0, nv0, maxDist), sampleRadialRGBA(grad, nu0, nv1, maxDist),
			sampleRadialRGBA(grad, nu1, nv1, maxDist), sampleRadialRGBA(grad, nu1, nv0, maxDist),
			u0, v0, u1, v1)
	}
}

// drawRoundedInner tiles an inner gradient into a batched rounded mesh.
// Corner rows emit one strip of column cells each; the tall middle band
// splits into rows so the vertical falloff stays smooth in one batch.
func drawRoundedInner(grad skin.Gradient, dest core.Rect, radius float32) {
	u0, v0, u1, v1, ready := beginGradientMesh()
	if !ready {
		return
	}
	count := roundedBandCount(radius)
	for i := range count {
		band := roundedBand(dest, radius, i, count)
		if band.W <= 0 || band.H <= 0 {
			continue
		}
		if i == count/2 {
			emitRoundedInnerMiddle(grad, dest, band, u0, v0, u1, v1)
			continue
		}
		emitRoundedInnerStrip(grad, dest, band, u0, v0, u1, v1)
	}
	endGradientMesh()
}

// emitRoundedInnerMiddle splits the tall middle band into row slices.
func emitRoundedInnerMiddle(grad skin.Gradient, dest, band core.Rect, u0, v0, u1, v1 float32) {
	for r := range roundedRadialRows {
		y0 := band.Y + float32(r)*band.H/float32(roundedRadialRows)
		y1 := band.Y + float32(r+1)*band.H/float32(roundedRadialRows)
		sub := core.Rect{X: band.X, Y: y0, W: band.W, H: y1 - y0}
		if sub.H <= 0 {
			continue
		}
		emitRoundedInnerStrip(grad, dest, sub, u0, v0, u1, v1)
	}
}

// emitRoundedInnerStrip emits one band row as column cells with sampled corners.
func emitRoundedInnerStrip(grad skin.Gradient, dest, band core.Rect, u0, v0, u1, v1 float32) {
	for c := range roundedRadialColumns {
		x0 := band.X + float32(c)*band.W/float32(roundedRadialColumns)
		x1 := band.X + float32(c+1)*band.W/float32(roundedRadialColumns)
		if x1 <= x0 {
			continue
		}
		nu0 := (x0 - dest.X) / dest.W
		nu1 := (x1 - dest.X) / dest.W
		nv0 := (band.Y - dest.Y) / dest.H
		nv1 := (band.Y + band.H - dest.Y) / dest.H
		emitGradientQuad(x0, band.Y, x1, band.Y+band.H,
			sampleInnerRGBA(grad, nu0, nv0), sampleInnerRGBA(grad, nu0, nv1),
			sampleInnerRGBA(grad, nu1, nv1), sampleInnerRGBA(grad, nu1, nv0),
			u0, v0, u1, v1)
	}
}

// drawRoundedSingleTexture tiles a stretched texture into rounded bands with
// source sub-rectangles that follow each band's destination slice.
func drawRoundedSingleTexture(texture rl.Texture2D, source, dest core.Rect, tint color.RGBA, radius float32) {
	count := roundedBandCount(radius)
	for i := range count {
		band := roundedBand(dest, radius, i, count)
		if band.W <= 0 || band.H <= 0 {
			continue
		}
		u0 := (band.X - dest.X) / dest.W
		u1 := (band.X + band.W - dest.X) / dest.W
		v0 := (band.Y - dest.Y) / dest.H
		v1 := (band.Y + band.H - dest.Y) / dest.H
		sub := core.Rect{
			X: source.X + u0*source.W,
			Y: source.Y + v0*source.H,
			W: (u1 - u0) * source.W,
			H: (v1 - v0) * source.H,
		}
		if sub.W > 0 && sub.H > 0 {
			drawSingleTexture(texture, sub, band, tint)
		}
	}
}

// bilinearColor samples the gradient corner quad at normalized (u, v).
func bilinearColor(topLeft, bottomLeft, bottomRight, topRight color.RGBA, u, v float32) color.RGBA {
	top := lerpColor(topLeft, topRight, u)
	bottom := lerpColor(bottomLeft, bottomRight, u)
	return lerpColor(top, bottom, v)
}

// lerpColor blends two colors by t in [0, 1].
func lerpColor(first, second color.RGBA, t float32) color.RGBA {
	return color.RGBA{
		R: lerpChannel(first.R, second.R, t),
		G: lerpChannel(first.G, second.G, t),
		B: lerpChannel(first.B, second.B, t),
		A: lerpChannel(first.A, second.A, t),
	}
}

// lerpChannel blends two channels by t in [0, 1].
func lerpChannel(first, second uint8, t float32) uint8 {
	return uint8(float32(first) + (float32(second)-float32(first))*t + 0.5)
}
