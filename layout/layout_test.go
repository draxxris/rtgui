package layout

import (
	"testing"

	"rtgui/core"
)

func TestAnchors(t *testing.T) {
	parent := core.Rect{X: 0, Y: 0, W: 800, H: 600}
	root := New("root", parent)
	// fixed size top-left
	child := New("child", core.Rect{X: 0, Y: 0, W: 100, H: 50})
	child.SetAnchor(AnchorTopLeft)
	child.SetOffset(core.Vec2{X: 10, Y: 10})
	child.SetFixedSize(core.Vec2{X: 100, Y: 50})
	root.AddChild(child)
	Arrange(root, parent)
	if child.Resolved.X != 10 || child.Resolved.Y != 10 {
		t.Fatalf("top-left %v", child.Resolved)
	}

	// center
	c2 := New("center", core.Rect{})
	c2.SetAnchor(AnchorCenter)
	c2.SetFixedSize(core.Vec2{X: 100, Y: 50})
	root.Children = nil
	root.AddChild(c2)
	Arrange(root, parent)
	if c2.Resolved.X != 350 || c2.Resolved.Y != 275 {
		t.Fatalf("center %v", c2.Resolved)
	}

	// bottom-right
	c3 := New("br", core.Rect{})
	c3.SetAnchor(AnchorBottomRight)
	c3.SetFixedSize(core.Vec2{X: 50, Y: 50})
	c3.SetOffset(core.Vec2{X: -10, Y: -10})
	root.Children = nil
	root.AddChild(c3)
	Arrange(root, parent)
	if c3.Resolved.X != 740 || c3.Resolved.Y != 540 {
		t.Fatalf("br %v", c3.Resolved)
	}

	// relative size
	c4 := New("rel", core.Rect{})
	c4.SetAnchor(AnchorTopLeft)
	c4.RelativeSize = &core.Vec2{X: 0.5, Y: 0.5}
	root.Children = nil
	root.AddChild(c4)
	Arrange(root, parent)
	if c4.Resolved.W != 400 || c4.Resolved.H != 300 {
		t.Fatalf("rel %v", c4.Resolved)
	}

	// min/max
	c5 := New("minmax", core.Rect{})
	c5.SetAnchor(AnchorTopLeft)
	c5.RelativeSize = &core.Vec2{X: 2, Y: 2}
	c5.MinSize = &core.Vec2{X: 100, Y: 100}
	c5.MaxSize = &core.Vec2{X: 500, Y: 500}
	root.Children = nil
	root.AddChild(c5)
	Arrange(root, parent)
	if c5.Resolved.W != 500 || c5.Resolved.H != 500 {
		t.Fatalf("minmax %v", c5.Resolved)
	}
}

func TestFrameRelativeMove(t *testing.T) {
	parent := core.Rect{X: 0, Y: 0, W: 800, H: 600}
	frame := New("frame", core.Rect{X: 10, Y: 10, W: 200, H: 200})
	child := New("child", core.Rect{X: 0, Y: 0, W: 50, H: 50})
	child.SetAnchor(AnchorTopLeft)
	child.SetFixedSize(core.Vec2{X: 50, Y: 50})
	frame.AddChild(child)
	Arrange(frame, parent)
	orig := child.Resolved
	MoveFrame(frame, core.Vec2{X: 20, Y: 30})
	if child.Resolved.X != orig.X+20 || child.Resolved.Y != orig.Y+30 {
		t.Fatalf("relative move failed orig %v now %v", orig, child.Resolved)
	}
}

func TestMeasure(t *testing.T) {
	n := New("n", core.Rect{X: 0, Y: 0, W: 100, H: 100})
	n.SetFixedSize(core.Vec2{X: 120, Y: 80})
	sz := Measure(n, core.Rect{X: 0, Y: 0, W: 800, H: 600})
	if sz.X != 120 || sz.Y != 80 {
		t.Fatalf("measure %v", sz)
	}
}

func TestViewportPlumbing(t *testing.T) {
	parent := core.Rect{X: 0, Y: 0, W: 640, H: 480}
	root := New("root", parent)
	child := New("child", core.Rect{})
	child.SetAnchor(AnchorCenter)
	child.SetFixedSize(core.Vec2{X: 100, Y: 100})
	root.AddChild(child)
	vp := core.Viewport{Viewport: parent, LogicalSize: core.Vec2{X: 640, Y: 480}}
	ApplyViewport(root, vp)
	if root.Resolved.W != 640 {
		t.Fatalf("root %v", root.Resolved)
	}
	// resize via explicit notify — no GUI-owned safe area
	vp2 := core.Viewport{Viewport: core.Rect{X: 0, Y: 0, W: 1920, H: 1080}, LogicalSize: core.Vec2{X: 1920, Y: 1080}}
	ApplyViewport(root, vp2)
	if root.Resolved.W != 1920 {
		t.Fatalf("resize %v", root.Resolved)
	}
}
