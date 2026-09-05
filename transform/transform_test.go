package transform

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestViewportMapping checks logical and physical coordinate round trips.
func TestViewportMapping(t *testing.T) {
	vp := core.Viewport{Viewport: core.Rect{X: 10, Y: 20, W: 800, H: 600}, LogicalSize: core.Vec2{X: 800, Y: 600}}
	transform := New(vp)
	physical := core.Vec2{X: 60, Y: 70}
	logical := transform.PhysicalToViewport(physical)
	if logical != (core.Vec2{X: 50, Y: 50}) || transform.ViewportToPhysical(logical) != physical {
		t.Fatalf("mapping logical=%v physical=%v", logical, transform.ViewportToPhysical(logical))
	}
	bounds := core.Rect{X: 40, Y: 40, W: 20, H: 20}
	if !transform.HitTestPhysical(physical, bounds) || transform.HitTestPhysical(core.Vec2{X: 10, Y: 20}, bounds) {
		t.Fatal("physical hit testing failed")
	}
}

// TestPixelSnapAndIntersection checks rounding and rectangle intersection.
func TestPixelSnapAndIntersection(t *testing.T) {
	transform := New(core.Viewport{})
	transform.PixelSnap = true
	if transform.Snap(10.6) != 11 {
		t.Fatalf("snap=%v", transform.Snap(10.6))
	}
	rect := transform.SnapRect(core.Rect{X: 10.2, Y: 10.7, W: 20.4, H: 20.4})
	if rect != (core.Rect{X: 10, Y: 11, W: 20, H: 20}) {
		t.Fatalf("snapped rect=%v", rect)
	}
	intersection, ok := Intersect(core.Rect{W: 100, H: 100}, core.Rect{X: 50, Y: 50, W: 100, H: 100})
	if !ok || intersection.W != 50 || intersection.H != 50 {
		t.Fatalf("intersection=%v ok=%v", intersection, ok)
	}
}

// TestViewportScaling verifies the fixed-logical model: the design resolution
// stays put while the window rescales the mapping on each axis independently.
func TestViewportScaling(t *testing.T) {
	vp := core.Viewport{Viewport: core.Rect{W: 1600, H: 900}, LogicalSize: core.Vec2{X: 800, Y: 600}}
	transform := New(vp)
	sx, sy := transform.Scale()
	if sx != 2 || !approxFloat(sy, 1.5) {
		t.Fatalf("scale=%v,%v", sx, sy)
	}
	logical := transform.PhysicalToViewport(core.Vec2{X: 200, Y: 150})
	if logical != (core.Vec2{X: 100, Y: 100}) {
		t.Fatalf("scaled mapping=%v", logical)
	}
	if physical := transform.ViewportToPhysical(logical); physical != (core.Vec2{X: 200, Y: 150}) {
		t.Fatalf("round trip=%v", physical)
	}
	bounds := core.Rect{X: 90, Y: 90, W: 20, H: 20}
	if !transform.HitTestPhysical(core.Vec2{X: 200, Y: 150}, bounds) {
		t.Fatal("scaled hit test must hit")
	}
	if transform.HitTestPhysical(core.Vec2{X: 10, Y: 10}, bounds) {
		t.Fatal("scaled hit test must miss")
	}
	if err := transform.SetViewport(core.Viewport{Viewport: core.Rect{W: 100, H: 100}}); err == nil {
		t.Fatal("empty logical size must be rejected")
	}
	if sx, sy := (Transform{}).Scale(); sx != 1 || sy != 1 {
		t.Fatalf("zero transform must stay 1:1, got %v,%v", sx, sy)
	}
}

func approxFloat(a, b float32) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-4
}
