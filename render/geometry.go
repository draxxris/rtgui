package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// NinePatchConfig contains the destination border widths for a nine-patch.
type NinePatchConfig struct {
	// Left and Top are the destination's left and top border widths.
	Left, Top float32
	// Right and Bottom are the destination's right and bottom border widths.
	Right, Bottom float32
}

// NinePatchRects returns the nine destination rectangles for borders and dest.
// Negative borders and destination sizes are clamped. If borders are larger
// than a destination dimension, they are scaled proportionally.
func NinePatchRects(borders NinePatchConfig, dest core.Rect) [9]core.Rect {
	left, right := nonNegative(borders.Left), nonNegative(borders.Right)
	top, bottom := nonNegative(borders.Top), nonNegative(borders.Bottom)
	dest.W = nonNegative(dest.W)
	dest.H = nonNegative(dest.H)

	left, right = fitPair(left, right, dest.W)
	top, bottom = fitPair(top, bottom, dest.H)
	midW := nonNegative(dest.W - left - right)
	midH := nonNegative(dest.H - top - bottom)

	return tileRects(dest.X, dest.Y, left, top, midW, midH, right, bottom)
}

// NinePatchSourceRects returns source rectangles corresponding to the nine
// parts of a skin atlas region.
func NinePatchSourceRects(src core.Rect, patch skin.NinePatch) [9]core.Rect {
	left := nonNegative(float32(patch.Left))
	right := nonNegative(float32(patch.Right))
	top := nonNegative(float32(patch.Top))
	bottom := nonNegative(float32(patch.Bottom))
	src.W = nonNegative(src.W)
	src.H = nonNegative(src.H)

	left, right = fitPair(left, right, src.W)
	top, bottom = fitPair(top, bottom, src.H)
	midW := nonNegative(src.W - left - right)
	midH := nonNegative(src.H - top - bottom)

	return tileRects(src.X, src.Y, left, top, midW, midH, right, bottom)
}

// tileRects builds the shared row-major nine-patch rectangle layout.
func tileRects(x, y, left, top, midW, midH, right, bottom float32) [9]core.Rect {
	r0c0 := core.Rect{X: x, Y: y, W: left, H: top}
	r0c1 := core.Rect{X: x + left, Y: y, W: midW, H: top}
	r0c2 := core.Rect{X: x + left + midW, Y: y, W: right, H: top}
	r1c0 := core.Rect{X: x, Y: y + top, W: left, H: midH}
	r1c1 := core.Rect{X: x + left, Y: y + top, W: midW, H: midH}
	r1c2 := core.Rect{X: x + left + midW, Y: y + top, W: right, H: midH}
	r2c0 := core.Rect{X: x, Y: y + top + midH, W: left, H: bottom}
	r2c1 := core.Rect{X: x + left, Y: y + top + midH, W: midW, H: bottom}
	r2c2 := core.Rect{X: x + left + midW, Y: y + top + midH, W: right, H: bottom}
	return [9]core.Rect{r0c0, r0c1, r0c2, r1c0, r1c1, r1c2, r2c0, r2c1, r2c2}
}

// fitPair proportionally scales two nonnegative values into size when needed.
func fitPair(first, second, size float32) (float32, float32) {
	if size <= 0 {
		return 0, 0
	}
	sum := first + second
	if sum <= size || sum == 0 {
		return first, second
	}
	scale := size / sum
	return first * scale, second * scale
}

func nonNegative(value float32) float32 {
	if value < 0 {
		return 0
	}
	return value
}

// ContentRect returns the area available to content. A nine-patch border is
// the minimum inset; explicit padding may add to it.
func ContentRect(bounds core.Rect, descriptor skin.SkinDescriptor) core.Rect {
	left := maxFloat(0, descriptor.PaddingLeft)
	top := maxFloat(0, descriptor.PaddingTop)
	right := maxFloat(0, descriptor.PaddingRight)
	bottom := maxFloat(0, descriptor.PaddingBottom)
	if descriptor.HasNinePatch {
		left = maxFloat(left, float32(descriptor.NinePatch.Left))
		top = maxFloat(top, float32(descriptor.NinePatch.Top))
		right = maxFloat(right, float32(descriptor.NinePatch.Right))
		bottom = maxFloat(bottom, float32(descriptor.NinePatch.Bottom))
	}
	bounds.W = nonNegative(bounds.W)
	bounds.H = nonNegative(bounds.H)
	return core.Rect{
		X: bounds.X + left,
		Y: bounds.Y + top,
		W: nonNegative(bounds.W - left - right),
		H: nonNegative(bounds.H - top - bottom),
	}
}

func maxFloat(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
