package layout

import (
	"testing"

	"rtgui/core"
)

func TestOpposingAnchorStretch(t *testing.T) {
	parent := core.Rect{X: 0, Y: 0, W: 800, H: 600}

	// Full stretch: top-left to bottom-right fills parent
	child := New("stretch", core.Rect{})
	child.SetAnchors(AnchorTopLeft, AnchorBottomRight)
	child.SetOffset(core.Vec2{X: 10, Y: 20})
	child.SetOpposingOffset(core.Vec2{X: -10, Y: -20})
	Arrange(child, parent)
	if child.Resolved.X != 10 || child.Resolved.Y != 20 {
		t.Fatalf("full stretch pos %v", child.Resolved)
	}
	if child.Resolved.W != 780 || child.Resolved.H != 560 {
		t.Fatalf("full stretch size %v", child.Resolved)
	}

	// Horizontal only: top-left to top-right should stretch X, not Y
	hOnly := New("hOnly", core.Rect{})
	hOnly.SetAnchors(AnchorTopLeft, AnchorTopRight)
	hOnly.SetOffset(core.Vec2{X: 0, Y: 10})
	hOnly.SetOpposingOffset(core.Vec2{X: 0, Y: 0})
	hOnly.FixedSize = nil // ensure no fixed
	// Provide fixed height to isolate X stretch
	// For this test, we want Y not stretched, so set authored height
	hOnly.Rect.H = 50
	hOnly.FixedSize = nil
	Arrange(hOnly, parent)
	if hOnly.Resolved.W != 800 {
		t.Fatalf("hOnly width %v want 800", hOnly.Resolved.W)
	}
	// Y should not be stretched to full height; should be offset + authored height (or parent H if no authored)
	// Since we set Rect.H=50 and no stretchY, Y height should be 50, not 600
	if hOnly.Resolved.H == 600 {
		t.Fatalf("hOnly incorrectly stretched Y to %v", hOnly.Resolved.H)
	}

	// Vertical only: left to bottom-left stretches Y
	vOnly := New("vOnly", core.Rect{})
	vOnly.SetAnchors(AnchorTopLeft, AnchorBottomLeft)
	vOnly.SetOffset(core.Vec2{X: 5, Y: 0})
	vOnly.SetOpposingOffset(core.Vec2{X: 0, Y: 0})
	vOnly.Rect.W = 100
	Arrange(vOnly, parent)
	if vOnly.Resolved.H != 600 {
		t.Fatalf("vOnly height %v want 600", vOnly.Resolved.H)
	}
	if vOnly.Resolved.W == 800 {
		t.Fatalf("vOnly incorrectly stretched X to 800")
	}

	// FixedSize overrides opposing anchor
	fixed := New("fixed", core.Rect{})
	fixed.SetAnchors(AnchorTopLeft, AnchorBottomRight)
	fixed.SetFixedSize(core.Vec2{X: 100, Y: 50})
	Arrange(fixed, parent)
	if fixed.Resolved.W != 100 || fixed.Resolved.H != 50 {
		t.Fatalf("FixedSize should override stretch %v", fixed.Resolved)
	}

	// RelativeSize also overrides
	rel := New("rel", core.Rect{})
	rel.SetAnchors(AnchorTopLeft, AnchorBottomRight)
	rel.RelativeSize = &core.Vec2{X: 0.5, Y: 0.5}
	Arrange(rel, parent)
	if rel.Resolved.W != 400 || rel.Resolved.H != 300 {
		t.Fatalf("RelativeSize should override stretch %v", rel.Resolved)
	}

	// StretchX/Y flags allow one-axis stretch even when not full edge pair
	// AnchorCenter to Center with StretchX should stretch X using parent edges + offsets
	stretchFlag := New("flag", core.Rect{})
	stretchFlag.SetAnchor(AnchorCenter)
	stretchFlag.SetOpposingAnchor(AnchorCenter)
	stretchFlag.SetStretch(true, false)
	stretchFlag.SetOffset(core.Vec2{X: -100, Y: 0})
	stretchFlag.SetOpposingOffset(core.Vec2{X: 100, Y: 0})
	Arrange(stretchFlag, parent)
	// Width = (parent.W + OpposingOffset.X) - Offset.X = (800+100)-(-100)=1000
	if stretchFlag.Resolved.W != 1000 {
		t.Fatalf("StretchX flag width %v want 1000", stretchFlag.Resolved.W)
	}
}

func TestMoveFrameConsistency(t *testing.T) {
	parent := core.Rect{X: 0, Y: 0, W: 800, H: 600}
	frame := New("frame", core.Rect{X: 10, Y: 10, W: 200, H: 200})
	child := New("child", core.Rect{X: 0, Y: 0, W: 50, H: 50})
	child.SetAnchor(AnchorTopLeft)
	child.SetOffset(core.Vec2{X: 5, Y: 5})
	child.SetFixedSize(core.Vec2{X: 50, Y: 50})
	frame.AddChild(child)
	Arrange(frame, parent)
	origChild := child.Resolved
	// Move frame and verify immediate resolved moves
	MoveFrame(frame, core.Vec2{X: 20, Y: 30})
	if child.Resolved.X != origChild.X+20 || child.Resolved.Y != origChild.Y+30 {
		t.Fatalf("immediate move failed orig %v now %v", origChild, child.Resolved)
	}
	immediate := child.Resolved
	// Arrange again with same parent — should produce same result, not double-move
	Arrange(frame, parent)
	if child.Resolved.X != immediate.X || child.Resolved.Y != immediate.Y {
		t.Fatalf("Arrange after MoveFrame double-moved: immediate %v now %v", immediate, child.Resolved)
	}
	// Also test nested frame move
	nested := New("nested", core.Rect{X: 10, Y: 10, W: 50, H: 50})
	nestedChild := New("nestedChild", core.Rect{X: 0, Y: 0, W: 10, H: 10})
	nestedChild.SetFixedSize(core.Vec2{X: 10, Y: 10})
	nested.AddChild(nestedChild)
	frame.AddChild(nested)
	Arrange(frame, parent)
	origNestedChild := nestedChild.Resolved
	MoveFrame(nested, core.Vec2{X: 7, Y: 7})
	if nestedChild.Resolved.X != origNestedChild.X+7 {
		t.Fatalf("nested move failed")
	}
}
