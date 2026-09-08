package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/dragdrop"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/widgets"
)

// mustParent links registered widgets for ownership regression tests.
func mustParent(t *testing.T, u *UI, child, parent string) {
	t.Helper()
	if err := u.SetParent(child, parent); err != nil {
		t.Fatal(err)
	}
}

// TestOverlayBlocksAllPointerEdges checks opaque, disabled, and transparent overlays.
func TestOverlayBlocksAllPointerEdges(t *testing.T) {
	u := New(300, 200)
	back := widgets.NewButton("back", core.Rect{W: 100, H: 100}, "back")
	front := widgets.NewFrame("front", back.Bounds())
	mustAdd(t, u, back, front)
	calls := 0
	back.OnClick(func() { calls++ })
	clickAt(u, core.Vec2{X: 20, Y: 20})
	front.SetEnabled(false)
	clickAt(u, core.Vec2{X: 20, Y: 20})
	if calls != 0 {
		t.Fatal("opaque overlay leaked a click")
	}
	if !u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 20, Y: 20}, Wheel: 1}) {
		t.Fatal("overlay leaked wheel input")
	}
	front.SetInputTransparent(true)
	clickAt(u, core.Vec2{X: 20, Y: 20})
	if calls != 1 {
		t.Fatal("explicit transparency did not pass input")
	}
}

// TestParentOwnsVisibilityFocusAndDisposal checks hierarchy semantics beyond geometry.
func TestParentOwnsVisibilityFocusAndDisposal(t *testing.T) {
	u := New(300, 200)
	frame := widgets.NewFrame("parent", core.Rect{X: 10, Y: 10, W: 150, H: 100})
	child := widgets.NewTextbox("child", core.Rect{X: 5, Y: 5, W: 60, H: 30}, 64)
	mustAdd(t, u, frame, child)
	mustParent(t, u, "child", "parent")
	if err := layout.Arrange(frame.Frame(), core.Rect{}); err != nil {
		t.Fatal(err)
	}
	if !u.Focus("child") || u.ActiveFrame() != frame {
		t.Fatal("focus did not follow ancestry")
	}
	frame.SetVisible(false)
	u.Draw()
	if u.Focused() != nil || u.Focus("child") {
		t.Fatal("hidden parent retained child focus")
	}
	frame.SetVisible(true)
	frame.SetEnabled(false)
	if u.Focus("child") {
		t.Fatal("disabled parent allowed child focus")
	}
	child.OnText(func(string) {})
	u.Remove("parent")
	if u.Lookup("child") != nil || child.Callbacks().Text != nil || child.Owner() != nil {
		t.Fatal("subtree not disposed")
	}
}

// TestSubtreesStackTogether prevents a background child jumping above a front root.
func TestSubtreesStackTogether(t *testing.T) {
	u := New(300, 200)
	back := widgets.NewFrame("back", core.Rect{W: 150, H: 150})
	front := widgets.NewFrame("front", back.Bounds())
	child := widgets.NewButton("child", core.Rect{W: 50, H: 50}, "child")
	mustAdd(t, u, back, front, child)
	mustParent(t, u, "child", "back")
	if u.hitSurface(core.Vec2{X: 10, Y: 10}) != front {
		t.Fatal("late child escaped its parent's stacking order")
	}
	u.BringToFront("child")
	if u.hitSurface(core.Vec2{X: 10, Y: 10}) != child {
		t.Fatal("raising a subtree failed")
	}
	if u.SetParent("back", "child") == nil || back.Frame().Parent() != nil {
		t.Fatal("cycle mutation was not atomic")
	}
}

// TestScrollChildCoordinatesAndRemoval checks shared scrolling geometry and capture cleanup.
func TestScrollChildCoordinatesAndRemoval(t *testing.T) {
	u := New(300, 200)
	panel := widgets.NewScrollPanel("scroll", core.Rect{W: 100, H: 100}).SetMaxScroll(core.Vec2{Y: 300})
	child := widgets.NewButton("child", core.Rect{X: 10, Y: 110, W: 40, H: 25}, "child")
	mustAdd(t, u, panel, child)
	mustParent(t, u, "child", "scroll")
	if err := layout.Arrange(panel.Frame(), core.Rect{}); err != nil {
		t.Fatal(err)
	}
	if u.hitSurface(core.Vec2{X: 15, Y: 115}) != nil {
		t.Fatal("unclipped child accepted input")
	}
	panel.SetScroll(core.Vec2{Y: 60})
	if u.hitSurface(core.Vec2{X: 15, Y: 55}) != child {
		t.Fatal("scroll drawing and hit geometry disagree")
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 90, Y: 18}, Pressed: true, Down: true})
	if u.scrollThumbDragging == nil {
		t.Fatal("thumb not captured")
	}
	u.Remove("scroll")
	before := panel.Scroll()
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 90, Y: 90}, Down: true})
	if panel.Scroll() != before || u.scrollThumbDragging != nil {
		t.Fatal("removed thumb still dragged")
	}
}

// dragFixture builds two reusable drag surfaces with one payload factory.
func dragFixture(t *testing.T) (*UI, *widgets.Button, *widgets.Frame) {
	t.Helper()
	u := New(300, 200)
	source := widgets.NewButton("source", core.Rect{W: 50, H: 50}, "item")
	target := widgets.NewFrame("target", core.Rect{X: 100, W: 100, H: 100})
	mustAdd(t, u, source, target)
	u.OnDrag("source", func() dragdrop.Payload { return dragdrop.NewPayload("source", "item", 7) })
	return u, source, target
}

// TestDragArbitratesClickAndRelease checks clicks, drags, and duplicate releases.
func TestDragArbitratesClickAndRelease(t *testing.T) {
	u, source, _ := dragFixture(t)
	clicks, drops := 0, 0
	source.OnClick(func() { clicks++ })
	u.OnDrop("target", nil, func(dragdrop.Payload) { drops++ })
	clickAt(u, core.Vec2{X: 10, Y: 10})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 10, Y: 10}, Pressed: true, Down: true})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 120, Y: 20}, Down: true})
	if u.Pressed() != nil || !u.DragController().CanDrop() {
		t.Fatal("drag retained click ownership")
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 120, Y: 20}, Released: true})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 120, Y: 20}, Released: true})
	if clicks != 1 || drops != 1 || u.drag.Payload().Data != nil {
		t.Fatal("gesture delivered twice or retained payload")
	}
}

// TestDragRejectsForegroundAndCancels checks rejection and Escape ownership.
func TestDragRejectsForegroundAndCancels(t *testing.T) {
	u, _, target := dragFixture(t)
	front := widgets.NewFrame("front", target.Bounds())
	mustAdd(t, u, front)
	drops := 0
	u.OnDrop("target", nil, func(dragdrop.Payload) { drops++ })
	u.OnDrop("front", func(dragdrop.Payload) bool { return false }, func(dragdrop.Payload) { drops++ })
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 10, Y: 10}, Pressed: true})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 120, Y: 20}, Released: true})
	if drops != 0 || u.drag.Phase() != dragdrop.PhaseCanceled {
		t.Fatal("rejection fell through to a background target")
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 10, Y: 10}, Pressed: true})
	if !u.HandleKey(KeyEvent{Escape: true}) {
		t.Fatal("Escape did not cancel drag")
	}
	if !u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 280, Y: 180}, Released: true}) {
		t.Fatal("canceled gesture leaked its release")
	}
}

// TestWidgetCannotShareCallbackOwner checks cross-UI registration and disposal.
func TestWidgetCannotShareCallbackOwner(t *testing.T) {
	a, b := New(100, 100), New(100, 100)
	w := widgets.NewButton("same", core.Rect{W: 30, H: 30}, "same")
	mustAdd(t, a, w)
	if b.Add(w) != ErrWidgetOwned {
		t.Fatal("widget gained two callback owners")
	}
	a.Remove("same")
	mustAdd(t, b, w)
}

// TestCallbackReplacementDoesNotActivateReusedName guards reentrant removal.
func TestCallbackReplacementDoesNotActivateReusedName(t *testing.T) {
	u := New(100, 100)
	tab := widgets.NewTabBar("tabs", core.Rect{W: 100, H: 30}, []string{"a", "b"}, 0)
	mustAdd(t, u, tab)
	clicks := 0
	tab.OnTabSelect(func(int) {
		u.Remove("tabs")
		replacement := widgets.NewButton("tabs", core.Rect{}, "new").OnClick(func() { clicks++ })
		if err := u.Add(replacement); err != nil {
			t.Fatal(err)
		}
	})
	u.SelectTab("tabs", 1)
	if clicks != 0 {
		t.Fatal("old selection activated a replacement widget")
	}
}
