// Package transform maps between physical window coordinates and logical UI
// coordinates. The logical size is the fixed design resolution chosen at
// startup; the physical viewport tracks the live window and always equals
// the window rectangle at origin zero. Resizing the window rescales the
// mapping instead of reflowing the layout, so components scale with the
// window. An independent WoW-style UI scale multiplies that mapping for
// user-selected magnification; layout is authored for scale 1 and may clip
// when scaled. State is explicit so applications can own more than one UI.
package transform

import (
	"math"

	"github.com/draxxris/rtgui/core"
)

// Transform stores the fixed logical-to-physical viewport mapping.
type Transform struct {
	// Viewport contains the physical rectangle and logical design size.
	// The physical rectangle always equals the window at origin zero;
	// margins are layout insets, never viewport offsets.
	Viewport core.Viewport
	// UIScale is the WoW-style user magnification multiplying the
	// window-derived mapping. Zero means 1; use SetUIScale to validate.
	UIScale float32
	// PixelSnap rounds logical positions and sizes when enabled.
	PixelSnap bool
}

// New returns a transform initialized with viewport and scale 1.
func New(viewport core.Viewport) *Transform {
	return &Transform{Viewport: viewport, UIScale: 1}
}

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

// SetUIScale records WoW-style user magnification on top of the
// window-derived mapping. Scale must be finite and positive; values
// above 4 are rejected as unusable. Invalid input keeps the old scale.
func (t *Transform) SetUIScale(scale float32) error {
	if t == nil {
		return core.StatusInvalidArg
	}
	if math.IsNaN(float64(scale)) || math.IsInf(float64(scale), 0) || scale <= 0 || scale > 4 {
		return core.StatusInvalidArg
	}
	t.UIScale = scale
	return nil
}

// GetUIScale reports the effective user magnification, treating zero
// and invalid values as 1 so zero-value transforms stay 1:1.
func (t Transform) GetUIScale() float32 {
	if math.IsNaN(float64(t.UIScale)) || math.IsInf(float64(t.UIScale), 0) || t.UIScale <= 0 {
		return 1
	}
	return t.UIScale
}

// Scale reports the physical-per-logical stretch on each axis: window size
// over design size times the UI scale. Axes share the UI factor; the window
// portion stays independent stretch. Non-positive dimensions fall back to 1
// so zero-value transforms keep 1:1 behavior instead of dividing by zero.
func (t Transform) Scale() (sx, sy float32) {
	viewport, logical := t.Viewport.Viewport, t.Viewport.LogicalSize
	if viewport.W <= 0 || viewport.H <= 0 || logical.X <= 0 || logical.Y <= 0 {
		return 1, 1
	}
	ui := t.GetUIScale()
	return viewport.W / logical.X * ui, viewport.H / logical.Y * ui
}

// ViewportToPhysical maps a logical point into the physical window
// rectangle, including the UI scale factor.
func (t Transform) ViewportToPhysical(p core.Vec2) core.Vec2 {
	sx, sy := t.Scale()
	viewport := t.Viewport.Viewport
	return core.Vec2{X: p.X*sx + viewport.X, Y: p.Y*sy + viewport.Y}
}

// PhysicalToViewport maps a physical window point into logical
// coordinates, inverting the UI scale factor.
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
