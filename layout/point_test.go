package layout

import (
	"errors"
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestPreferredSizeRules checks fixed, relative, authored, minimum, and
// maximum behavior when fewer than two independent equations exist.
func TestPreferredSizeRules(t *testing.T) {
	root := New("root", core.Rect{W: 800, H: 600})
	fixed := New("fixed", core.Rect{W: 10, H: 10})
	fixed.SetRelativeSize(core.Vec2{X: 0.5, Y: 0.5})
	fixed.SetFixedSize(core.Vec2{X: 120, Y: 80})
	mustSetPoint(t, fixed, AnchorCenter, nil, AnchorCenter, core.Vec2{})
	relative := New("relative", core.Rect{})
	relative.SetRelativeSize(core.Vec2{X: 0.5, Y: 0.25})
	mustSetPoint(t, relative, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{})
	limited := New("limited", core.Rect{X: 10, Y: 20, W: 10, H: 200})
	limited.SetMinSize(core.Vec2{X: 20, Y: 30})
	limited.SetMaxSize(core.Vec2{X: 80, Y: 90})
	mustAddChild(t, root, fixed)
	mustAddChild(t, root, relative)
	mustAddChild(t, root, limited)
	if err := ArrangeRoot(root, root.AuthoredBounds()); err != nil {
		t.Fatal(err)
	}
	requireRect(t, fixed.Bounds(), core.Rect{X: 340, Y: 260, W: 120, H: 80})
	requireRect(t, relative.Bounds(), core.Rect{W: 400, H: 150})
	requireRect(t, limited.Bounds(), core.Rect{X: 10, Y: 20, W: 20, H: 90})
}

// TestPointDerivedSizeLimits rejects inferred widths that violate minimum or
// maximum limits instead of clamping and breaking a point relation.
func TestPointDerivedSizeLimits(t *testing.T) {
	minimumRoot, minimum := horizontalFillNode(t)
	minimum.SetMinSize(core.Vec2{X: 101})
	if err := ArrangeRoot(minimumRoot, minimumRoot.AuthoredBounds()); !errors.Is(err, ErrMinimumSize) {
		t.Fatalf("minimum error = %v", err)
	}
	maximumRoot, maximum := horizontalFillNode(t)
	maximum.SetMaxSize(core.Vec2{X: 99, Y: 100})
	if err := ArrangeRoot(maximumRoot, maximumRoot.AuthoredBounds()); !errors.Is(err, ErrMaximumSize) {
		t.Fatalf("maximum error = %v", err)
	}
}

// horizontalFillNode returns a child whose width is determined by parent edges.
func horizontalFillNode(t *testing.T) (*Node, *Node) {
	t.Helper()
	root := New("root", core.Rect{W: 100, H: 100})
	child := New("child", core.Rect{H: 20})
	mustAddChild(t, root, child)
	mustSetPoint(t, child, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{})
	mustSetPoint(t, child, AnchorTopRight, nil, AnchorTopRight, core.Vec2{})
	return root, child
}

// TestNegativeAndInvalidSizesReturnErrors checks impossible point solutions,
// bad authored values, and contradictory limit configuration.
func TestNegativeAndInvalidSizesReturnErrors(t *testing.T) {
	root := New("root", core.Rect{W: 100, H: 100})
	negative := New("negative", core.Rect{H: 10})
	mustAddChild(t, root, negative)
	mustSetPoint(t, negative, AnchorTopLeft, nil, AnchorTopRight, core.Vec2{})
	mustSetPoint(t, negative, AnchorTopRight, nil, AnchorTopLeft, core.Vec2{})
	if err := ArrangeRoot(root, root.AuthoredBounds()); !errors.Is(err, ErrNegativeSize) {
		t.Fatalf("negative point size error = %v", err)
	}

	invalid := New("invalid", core.Rect{W: 10, H: 10})
	invalid.SetMinSize(core.Vec2{X: 20})
	invalid.SetMaxSize(core.Vec2{X: 10, Y: 10})
	if err := Arrange(invalid, core.Rect{W: 100, H: 100}); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("invalid limits error = %v", err)
	}
	if err := Arrange(nil, core.Rect{}); !errors.Is(err, ErrNilNode) {
		t.Fatalf("nil Arrange error = %v", err)
	}
	if err := ArrangeRoot(New("root", core.Rect{}), core.Rect{W: -1}); !errors.Is(err, ErrNegativeSize) {
		t.Fatalf("invalid viewport error = %v", err)
	}
}

// TestMoveFrameRequiresArrangeAndDoesNotDoubleMove verifies authored movement,
// stale resolved bounds before arrangement, and stable subsequent passes.
func TestMoveFrameRequiresArrangeAndDoesNotDoubleMove(t *testing.T) {
	frame := New("frame", core.Rect{X: 10, Y: 20, W: 200, H: 100})
	child := New("child", core.Rect{W: 50, H: 20})
	mustAddChild(t, frame, child)
	mustSetPoint(t, child, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{X: 5, Y: 7})
	if err := Arrange(frame, core.Rect{}); err != nil {
		t.Fatal(err)
	}
	if frame.NeedsArrange() {
		t.Fatal("successful arrangement left the tree dirty")
	}
	originalFrame, originalChild := frame.Bounds(), child.Bounds()
	MoveFrame(frame, core.Vec2{X: 20, Y: 30})
	if !frame.NeedsArrange() || frame.Bounds() != originalFrame || child.Bounds() != originalChild {
		t.Fatal("MoveFrame did not dirty the tree or changed resolved bounds early")
	}
	if err := Arrange(frame, core.Rect{}); err != nil {
		t.Fatal(err)
	}
	requireRect(t, frame.Bounds(), core.Rect{X: 30, Y: 50, W: 200, H: 100})
	requireRect(t, child.Bounds(), core.Rect{X: 35, Y: 57, W: 50, H: 20})
	first := child.Bounds()
	if err := Arrange(frame, core.Rect{}); err != nil {
		t.Fatal(err)
	}
	if child.Bounds() != first {
		t.Fatalf("repeated arrangement moved child from %+v to %+v", first, child.Bounds())
	}
}

// TestMoveFrameUpdatesPointOffsets moves a fully constrained frame without
// changing its point-derived size.
func TestMoveFrameUpdatesPointOffsets(t *testing.T) {
	root := New("root", core.Rect{W: 100, H: 80})
	frame := New("frame", core.Rect{})
	mustAddChild(t, root, frame)
	if err := frame.SetAllPoints(nil); err != nil {
		t.Fatal(err)
	}
	if err := ArrangeRoot(root, root.AuthoredBounds()); err != nil {
		t.Fatal(err)
	}
	MoveFrame(frame, core.Vec2{X: 4, Y: 6})
	if err := ArrangeRoot(root, root.AuthoredBounds()); err != nil {
		t.Fatal(err)
	}
	requireRect(t, frame.Bounds(), core.Rect{X: 4, Y: 6, W: 100, H: 80})
}

// TestMeasureDoesNotTraverseChildren protects the single-pass arrangement
// model while retaining focused preferred-size measurement.
func TestMeasureDoesNotTraverseChildren(t *testing.T) {
	parent := New("parent", core.Rect{W: 10, H: 20})
	child := New("child", core.Rect{W: 30, H: 40})
	mustAddChild(t, parent, child)
	calls := 0
	child.SetOnResize(func(*Node) { calls++ })
	if got := Measure(parent, core.Rect{W: 100, H: 100}); got != (core.Vec2{X: 10, Y: 20}) {
		t.Fatalf("Measure = %+v", got)
	}
	if calls != 0 || child.ResolvedBounds() != (core.Rect{}) {
		t.Fatal("Measure traversed or arranged a child")
	}
	if err := Arrange(parent, core.Rect{}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("child resolved %d times, want once", calls)
	}
}

// TestViewportFunctionsPropagateLayoutErrors verifies both viewport entry
// points retain meaningful graph and constraint failures.
func TestViewportFunctionsPropagateLayoutErrors(t *testing.T) {
	root := New("root", core.Rect{})
	child := New("child", core.Rect{W: 10, H: 10})
	outside := New("outside", core.Rect{})
	mustAddChild(t, root, child)
	mustSetPoint(t, child, AnchorTopLeft, outside, AnchorTopLeft, core.Vec2{})
	viewport := core.Viewport{Viewport: core.Rect{W: 100, H: 100}}
	if err := ApplyViewport(root, viewport); !errors.Is(err, ErrAnchorTargetOutsideTree) {
		t.Fatalf("ApplyViewport error = %v", err)
	}
	if err := ArrangeRoot(root, viewport.Viewport); !errors.Is(err, ErrAnchorTargetOutsideTree) {
		t.Fatalf("ArrangeRoot error = %v", err)
	}
}
