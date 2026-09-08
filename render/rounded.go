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
// gradient, and single textures tile into bands; patched textures fall back
// to sharp drawing since their tiles cannot follow the arc.
func drawRoundedBackground(descriptor skin.SkinDescriptor, dest core.Rect, tint color.RGBA, radius float32) {
	if descriptor.HasBackgroundColor && descriptor.BackgroundColor.A > 0 {
		drawRoundedColor(descriptor.BackgroundColor.RGBA(), dest, radius)
	}
	if descriptor.HasGradient {
		drawRoundedGradient(descriptor.Gradient, dest, radius)
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

// drawRoundedGradient tiles a gradient into rounded bands. Each band samples
// the sharp quad's bilinear surface at its own corners, so the union matches
// drawLinearGradient exactly.
func drawRoundedGradient(grad skin.LinearGradient, dest core.Rect, radius float32) {
	c0 := grad.Stops[0].Color.RGBA()
	c1 := grad.Stops[1].Color.RGBA()
	topLeft, bottomLeft, bottomRight, topRight := gradientQuadColors(grad.Direction, c0, c1)
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
		rl.DrawRectangleGradientEx(toRaylibRect(band),
			bilinearColor(topLeft, bottomLeft, bottomRight, topRight, u0, v0),
			bilinearColor(topLeft, bottomLeft, bottomRight, topRight, u0, v1),
			bilinearColor(topLeft, bottomLeft, bottomRight, topRight, u1, v1),
			bilinearColor(topLeft, bottomLeft, bottomRight, topRight, u1, v0))
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
