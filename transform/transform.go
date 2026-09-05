// Package transform maps between physical window coordinates and logical UI
// coordinates. The logical size is the fixed design resolution chosen at
// startup; the physical viewport tracks the live window. Resizing the window
// rescales the mapping instead of reflowing the layout, so components scale
// with the window. State is explicit so applications can own more than one UI.
package transform

import (
	"math"

	"github.com/draxxris/rtgui/core"
)

// Transform stores the fixed logical-to-physical viewport mapping.
type Transform struct {
	// Viewport contains the physical rectangle and logical design size.
	Viewport core.Viewport
	// PixelSnap rounds logical positions and sizes when enabled.
	PixelSnap bool
}

// New returns a transform initialized with viewport.
func New(viewport core.Viewport) *Transform { return &Transform{Viewport: viewport} }

// SetViewport validates and replaces the physical and logical dimensions.
func (t *Transform) SetViewport(viewport core.Viewport) error {
	if viewport.Viewport.W <= 0 || viewport.Viewport.H <= 0 {
		return core.StatusInvalidArg
	}
	if viewport.LogicalSize.X <= 0 || viewport.LogicalSize.Y <= 0 {
		return core.StatusInvalidArg
	}
	t.Viewport = viewport
	return nil
}

// Scale reports the physical-per-logical stretch on each axis: window size
// over design size. Axes are independent (stretch); a uniform factor is the
// caller's choice. Non-positive dimensions fall back to 1 so zero-value
// transforms keep the old 1:1 offset behavior instead of dividing by zero.
func (t Transform) Scale() (sx, sy float32) {
	viewport, logical := t.Viewport.Viewport, t.Viewport.LogicalSize
	if viewport.W <= 0 || viewport.H <= 0 || logical.X <= 0 || logical.Y <= 0 {
		return 1, 1
	}
	return viewport.W / logical.X, viewport.H / logical.Y
}

// ViewportToPhysical maps a logical point into the physical window rectangle.
func (t Transform) ViewportToPhysical(p core.Vec2) core.Vec2 {
	sx, sy := t.Scale()
	viewport := t.Viewport.Viewport
	return core.Vec2{X: p.X*sx + viewport.X, Y: p.Y*sy + viewport.Y}
}

// PhysicalToViewport maps a physical window point into logical coordinates.
func (t Transform) PhysicalToViewport(p core.Vec2) core.Vec2 {
	sx, sy := t.Scale()
	viewport := t.Viewport.Viewport
	return core.Vec2{X: (p.X - viewport.X) / sx, Y: (p.Y - viewport.Y) / sy}
}

// HitTestPhysical maps input from the window into logical coordinates before
// testing it against bounds.
func (t Transform) HitTestPhysical(input core.Vec2, bounds core.Rect) bool {
	return t.HitTestPhysicalPoint(input, bounds)
}

// HitTest tests a point already expressed in logical coordinates.
func HitTest(input core.Vec2, bounds core.Rect) bool { return bounds.Contains(input) }

// HitTestPhysicalPoint maps a physical point before testing logical bounds.
func (t Transform) HitTestPhysicalPoint(input core.Vec2, bounds core.Rect) bool {
	return HitTest(t.PhysicalToViewport(input), bounds)
}

// Snap rounds v when pixel snapping is enabled.
func (t Transform) Snap(v float32) float32 {
	if !t.PixelSnap {
		return v
	}
	return float32(math.Round(float64(v)))
}

// SnapVec2 rounds both components according to PixelSnap.
func (t Transform) SnapVec2(v core.Vec2) core.Vec2 {
	return core.Vec2{X: t.Snap(v.X), Y: t.Snap(v.Y)}
}

// SnapRect rounds the origin and size according to PixelSnap.
func (t Transform) SnapRect(r core.Rect) core.Rect {
	return core.Rect{X: t.Snap(r.X), Y: t.Snap(r.Y), W: t.Snap(r.W), H: t.Snap(r.H)}
}

// Intersect returns the positive-area intersection of a and b.
func Intersect(a, b core.Rect) (core.Rect, bool) {
	x1 := max(a.X, b.X)
	y1 := max(a.Y, b.Y)
	x2 := min(a.X+a.W, b.X+b.W)
	y2 := min(a.Y+a.H, b.Y+b.H)
	if x2 <= x1 || y2 <= y1 {
		return core.Rect{}, false
	}
	return core.Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}, true
}

func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
