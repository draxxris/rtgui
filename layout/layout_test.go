package layout

import (
	"errors"
	"math"
	"testing"

	"github.com/draxxris/rtgui/core"
)

func mustAddChild(t *testing.T, parent, child *Node) {
	t.Helper()
	if err := parent.AddChild(child); err != nil {
		t.Fatalf("AddChild: %v", err)
	}
}

func mustSetPoint(t *testing.T, node *Node, source Anchor, target *Node, targetPoint Anchor, offset core.Vec2) {
	t.Helper()
	if err := node.SetPoint(source, target, targetPoint, offset); err != nil {
		t.Fatalf("SetPoint: %v", err)
	}
}

// requireRect compares logical geometry using the solver's documented tolerance.
func requireRect(t *testing.T, got, want core.Rect) {
	t.Helper()
	values := [][2]float32{{got.X, want.X}, {got.Y, want.Y}, {got.W, want.W}, {got.H, want.H}}
	for _, pair := range values {
		if math.Abs(float64(pair[0]-pair[1])) > float64(ConstraintTolerance) {
			t.Fatalf("bounds = %+v, want %+v", got, want)
		}
	}
}

// TestSinglePointRelations covers parent-edge spacing, centered placement,
// and bottom-right insets with a preferred authored size.
func TestSinglePointRelations(t *testing.T) {
	parentBounds := core.Rect{W: 800, H: 600}
	root := New("root", parentBounds)
	topRight := New("top-right", core.Rect{W: 100, H: 50})
	center := New("center", core.Rect{W: 100, H: 50})
	bottomRight := New("bottom-right", core.Rect{W: 50, H: 50})
	mustAddChild(t, root, topRight)
	mustAddChild(t, root, center)
	mustAddChild(t, root, bottomRight)
	mustSetPoint(t, topRight, AnchorTopLeft, nil, AnchorTopRight, core.Vec2{X: 8, Y: 12})
	mustSetPoint(t, center, AnchorCenter, root, AnchorCenter, core.Vec2{})
	mustSetPoint(t, bottomRight, AnchorBottomRight, nil, AnchorBottomRight, core.Vec2{X: -10, Y: -20})
	if err := ArrangeRoot(root, parentBounds); err != nil {
		t.Fatal(err)
	}
	requireRect(t, topRight.Bounds(), core.Rect{X: 808, Y: 12, W: 100, H: 50})
	requireRect(t, center.Bounds(), core.Rect{X: 350, Y: 275, W: 100, H: 50})
	requireRect(t, bottomRight.Bounds(), core.Rect{X: 740, Y: 530, W: 50, H: 50})
}

// TestTwoPointStretch covers independent horizontal, vertical, and full-axis
// point solutions.
func TestTwoPointStretch(t *testing.T) {
	parentBounds := core.Rect{W: 800, H: 600}
	root := New("root", parentBounds)
	horizontal := New("horizontal", core.Rect{H: 40})
	vertical := New("vertical", core.Rect{W: 70})
	full := New("full", core.Rect{})
	mustAddChild(t, root, horizontal)
	mustAddChild(t, root, vertical)
	mustAddChild(t, root, full)
	mustSetPoint(t, horizontal, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{X: 10, Y: 20})
	mustSetPoint(t, horizontal, AnchorTopRight, nil, AnchorTopRight, core.Vec2{X: -30, Y: 20})
	mustSetPoint(t, vertical, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{X: 15, Y: 10})
	mustSetPoint(t, vertical, AnchorBottomLeft, nil, AnchorBottomLeft, core.Vec2{X: 15, Y: -30})
	if err := full.SetAllPoints(root); err != nil {
		t.Fatal(err)
	}
	if err := ArrangeRoot(root, parentBounds); err != nil {
		t.Fatal(err)
	}
	requireRect(t, horizontal.Bounds(), core.Rect{X: 10, Y: 20, W: 760, H: 40})
	requireRect(t, vertical.Bounds(), core.Rect{X: 15, Y: 10, W: 70, H: 560})
	requireRect(t, full.Bounds(), parentBounds)
}

// TestReverseSiblingDependency verifies a target resolves before a dependent
// even when the dependent was registered first.
func TestReverseSiblingDependency(t *testing.T) {
	root := New("root", core.Rect{W: 400, H: 300})
	dependent := New("dependent", core.Rect{W: 30, H: 10})
	target := New("target", core.Rect{W: 40, H: 20})
	mustAddChild(t, root, dependent)
	mustAddChild(t, root, target)
	mustSetPoint(t, target, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{X: 100, Y: 50})
	mustSetPoint(t, dependent, AnchorTopLeft, target, AnchorBottomRight, core.Vec2{X: 5, Y: 7})
	order := make([]string, 0, 2)
	dependent.SetOnResize(func(*Node) { order = append(order, "dependent") })
	target.SetOnResize(func(*Node) { order = append(order, "target") })
	if err := ArrangeRoot(root, root.AuthoredBounds()); err != nil {
		t.Fatal(err)
	}
	requireRect(t, dependent.Bounds(), core.Rect{X: 145, Y: 77, W: 30, H: 10})
	if len(order) != 2 || order[0] != "target" || order[1] != "dependent" {
		t.Fatalf("resolution order = %v", order)
	}
}

// TestDependencyCacheInvalidation verifies point replacement and ownership
// removal rebuild the cached order before the next arrangement.
func TestDependencyCacheInvalidation(t *testing.T) {
	root := New("root", core.Rect{W: 200, H: 100})
	dependent := New("dependent", core.Rect{W: 10, H: 10})
	first := New("first", core.Rect{W: 10, H: 10})
	second := New("second", core.Rect{W: 10, H: 10})
	mustAddChild(t, root, dependent)
	mustAddChild(t, root, first)
	mustAddChild(t, root, second)
	mustSetPoint(t, first, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{X: 10})
	mustSetPoint(t, second, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{X: 50})
	mustSetPoint(t, dependent, AnchorTopLeft, first, AnchorTopRight, core.Vec2{})
	if err := ArrangeRoot(root, root.AuthoredBounds()); err != nil {
		t.Fatal(err)
	}
	if dependent.Bounds().X != 20 || root.cache.graphDirty {
		t.Fatalf("initial dependency result/cache = %+v/%v", dependent.Bounds(), root.cache.graphDirty)
	}
	mustSetPoint(t, dependent, AnchorTopLeft, second, AnchorTopRight, core.Vec2{})
	if !root.cache.graphDirty {
		t.Fatal("point target replacement did not invalidate dependency order")
	}
	if err := ArrangeRoot(root, root.AuthoredBounds()); err != nil {
		t.Fatal(err)
	}
	if dependent.Bounds().X != 60 {
		t.Fatalf("replacement dependency bounds = %+v", dependent.Bounds())
	}
	if !root.RemoveChild(second) {
		t.Fatal("RemoveChild rejected a direct child")
	}
	if err := ArrangeRoot(root, root.AuthoredBounds()); !errors.Is(err, ErrAnchorTargetOutsideTree) {
		t.Fatalf("removed target error = %v", err)
	}
}

// TestOwnershipValidation rejects nil children, multiple parents, and cycles
// before they can corrupt dependency ordering.
func TestOwnershipValidation(t *testing.T) {
	root := New("root", core.Rect{})
	child := New("child", core.Rect{})
	other := New("other", core.Rect{})
	if err := root.AddChild(nil); !errors.Is(err, ErrNilChild) {
		t.Fatalf("nil child error = %v", err)
	}
	mustAddChild(t, root, child)
	if err := other.AddChild(child); !errors.Is(err, ErrMultipleParents) {
		t.Fatalf("multiple parent error = %v", err)
	}
	if err := child.AddChild(root); !errors.Is(err, ErrOwnershipCycle) {
		t.Fatalf("ownership cycle error = %v", err)
	}
	if root.Parent() != nil || child.Parent() != root || len(root.Children()) != 1 {
		t.Fatal("failed ownership operations mutated the valid tree")
	}
}

// TestAnchorGraphValidation rejects sibling cycles, self-targeting, and
// explicit targets outside the arranged ownership tree.
func TestAnchorGraphValidation(t *testing.T) {
	root := New("root", core.Rect{W: 100, H: 100})
	first := New("first", core.Rect{W: 10, H: 10})
	second := New("second", core.Rect{W: 10, H: 10})
	mustAddChild(t, root, first)
	mustAddChild(t, root, second)
	mustSetPoint(t, first, AnchorTopLeft, second, AnchorTopLeft, core.Vec2{})
	mustSetPoint(t, second, AnchorTopLeft, first, AnchorTopLeft, core.Vec2{})
	if err := ArrangeRoot(root, root.AuthoredBounds()); !errors.Is(err, ErrAnchorCycle) {
		t.Fatalf("anchor cycle error = %v", err)
	}
	if err := first.SetPoint(AnchorCenter, first, AnchorCenter, core.Vec2{}); !errors.Is(err, ErrAnchorCycle) {
		t.Fatalf("self anchor error = %v", err)
	}

	outsideRoot := New("outside-root", core.Rect{W: 100, H: 100})
	inside := New("inside", core.Rect{W: 10, H: 10})
	outside := New("outside", core.Rect{W: 10, H: 10})
	mustAddChild(t, outsideRoot, inside)
	mustSetPoint(t, inside, AnchorTopLeft, outside, AnchorTopLeft, core.Vec2{})
	if err := ArrangeRoot(outsideRoot, outsideRoot.AuthoredBounds()); !errors.Is(err, ErrAnchorTargetOutsideTree) {
		t.Fatalf("outside target error = %v", err)
	}
}

// TestConflictingThirdEquation ensures extra equations validate instead of
// silently overriding the deterministic two-equation solution.
func TestConflictingThirdEquation(t *testing.T) {
	root := New("root", core.Rect{W: 100, H: 100})
	child := New("child", core.Rect{H: 20})
	mustAddChild(t, root, child)
	mustSetPoint(t, child, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{})
	mustSetPoint(t, child, AnchorTopRight, nil, AnchorTopRight, core.Vec2{})
	mustSetPoint(t, child, AnchorBottomLeft, nil, AnchorBottomLeft, core.Vec2{X: 1})
	if err := ArrangeRoot(root, root.AuthoredBounds()); !errors.Is(err, ErrConstraintConflict) {
		t.Fatalf("third equation error = %v", err)
	}
}

// TestAnchorNames covers every enum value and stable unknown diagnostics.
func TestAnchorNames(t *testing.T) {
	want := []string{"top-left", "top", "top-right", "left", "center", "right", "bottom-left", "bottom", "bottom-right"}
	for index, name := range want {
		if got := AnchorName(Anchor(index)); got != name {
			t.Fatalf("AnchorName(%d) = %q, want %q", index, got, name)
		}
	}
	if AnchorName(Anchor(-1)) != "unknown" || AnchorName(AnchorBottomRight+1) != "unknown" {
		t.Fatal("unknown anchors did not use the stable fallback")
	}
}

// TestPointMutationValidation covers replacement, clear operations, and
// invalid enum handling without exposing mutable relation fields.
func TestPointMutationValidation(t *testing.T) {
	node := New("node", core.Rect{W: 10, H: 10})
	mustSetPoint(t, node, AnchorTopLeft, nil, AnchorTopLeft, core.Vec2{})
	mustSetPoint(t, node, AnchorBottomRight, nil, AnchorBottomRight, core.Vec2{})
	mustSetPoint(t, node, AnchorTopLeft, nil, AnchorCenter, core.Vec2{})
	if node.PointCount() != 2 || node.points[0].source != AnchorTopLeft {
		t.Fatal("point replacement changed relation count or order")
	}
	if err := node.ClearPoint(AnchorTopLeft); err != nil || node.PointCount() != 1 {
		t.Fatalf("ClearPoint = %v count=%d", err, node.PointCount())
	}
	node.ClearAllPoints()
	if node.PointCount() != 0 {
		t.Fatal("ClearAllPoints retained relations")
	}
	if err := node.SetPoint(Anchor(-1), nil, AnchorTopLeft, core.Vec2{}); !errors.Is(err, ErrInvalidAnchor) {
		t.Fatalf("invalid SetPoint error = %v", err)
	}
	if err := node.ClearPoint(Anchor(99)); !errors.Is(err, ErrInvalidAnchor) {
		t.Fatalf("invalid ClearPoint error = %v", err)
	}
}
