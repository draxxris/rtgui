// Package transform maps between physical window coordinates and logical UI
// coordinates. State is explicit so applications can own more than one UI.
package transform

import (
	"math"

	"rtgui/core"
)

type Transform struct {
	Viewport  core.Viewport
	PixelSnap bool
}

func New(viewport core.Viewport) *Transform { return &Transform{Viewport: viewport} }

func (t *Transform) SetViewport(viewport core.Viewport) error {
	if viewport.Viewport.W <= 0 || viewport.Viewport.H <= 0 {
		return core.StatusInvalidArg
	}
	t.Viewport = viewport
	return nil
}

func (t Transform) ViewportToPhysical(p core.Vec2) core.Vec2 {
	viewport := t.Viewport.Viewport
	return core.Vec2{X: p.X + viewport.X, Y: p.Y + viewport.Y}
}

func (t Transform) PhysicalToViewport(p core.Vec2) core.Vec2 {
	viewport := t.Viewport.Viewport
	return core.Vec2{X: p.X - viewport.X, Y: p.Y - viewport.Y}
}

// HitTestPhysical maps input from the window into logical coordinates before
// testing it against bounds.
func (t Transform) HitTestPhysical(input core.Vec2, bounds core.Rect) bool {
	return t.HitTestPhysicalPoint(input, bounds)
}

// HitTest tests a point already expressed in logical coordinates.
func HitTest(input core.Vec2, bounds core.Rect) bool { return bounds.Contains(input) }

func (t Transform) HitTestPhysicalPoint(input core.Vec2, bounds core.Rect) bool {
	return HitTest(t.PhysicalToViewport(input), bounds)
}

func (t Transform) Snap(v float32) float32 {
	if !t.PixelSnap {
		return v
	}
	return float32(math.Round(float64(v)))
}

func (t Transform) SnapVec2(v core.Vec2) core.Vec2 {
	return core.Vec2{X: t.Snap(v.X), Y: t.Snap(v.Y)}
}

func (t Transform) SnapRect(r core.Rect) core.Rect {
	return core.Rect{X: t.Snap(r.X), Y: t.Snap(r.Y), W: t.Snap(r.W), H: t.Snap(r.H)}
}

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
