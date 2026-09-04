package transform

import (
	"testing"

	"rtgui/core"
)

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
