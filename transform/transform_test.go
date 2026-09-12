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

// TestUIScale verifies WoW-style magnification multiplies the mapping,
// round-trips input, and rejects invalid scales.
func TestUIScale(t *testing.T) {
	vp := core.Viewport{Viewport: core.Rect{W: 800, H: 600}, LogicalSize: core.Vec2{X: 800, Y: 600}}
	tr := New(vp)
	if got := tr.GetUIScale(); got != 1 {
		t.Fatalf("default UI scale = %v, want 1", got)
	}
	if err := tr.SetUIScale(2); err != nil {
		t.Fatal(err)
	}
	sx, sy := tr.Scale()
	if sx != 2 || sy != 2 {
		t.Fatalf("scaled scale = %v/%v", sx, sy)
	}
	if got := tr.ViewportToPhysical(core.Vec2{X: 10, Y: 20}); got != (core.Vec2{X: 20, Y: 40}) {
		t.Fatalf("scaled physical = %+v", got)
	}
	if got := tr.PhysicalToViewport(core.Vec2{X: 20, Y: 40}); got != (core.Vec2{X: 10, Y: 20}) {
		t.Fatalf("scaled logical = %+v", got)
	}
	for _, bad := range []float32{0, -1} {
		if err := tr.SetUIScale(bad); err == nil {
			t.Fatalf("invalid scale %v accepted", bad)
		}
	}
	if got := tr.GetUIScale(); got != 2 {
		t.Fatalf("rejected scale mutated to %v", got)
	}
	if got := (Transform{}).GetUIScale(); got != 1 {
		t.Fatalf("zero UI scale = %v, want 1", got)
	}
}

func approxFloat(a, b float32) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-4
}
