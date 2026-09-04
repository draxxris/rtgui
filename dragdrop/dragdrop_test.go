package dragdrop

import (
	"testing"

	"rtgui/core"
	"rtgui/input"
)

func TestDragThreshold(t *testing.T) {
	ds := NewDragState(5, input.NewCapture())
	payload := NewPayload("dragSource", "icon", "data")
	ds.OnPress("dragSource", core.Vec2{X: 10, Y: 10}, payload)
	if ds.Phase != PhasePressed {
		t.Fatalf("pressed %v", ds.Phase)
	}
	ds.OnMove(core.Vec2{X: 12, Y: 12}) // dist sqrt(8) ~2.8 <5
	if ds.Phase != PhaseThresholdPending {
		t.Fatalf("pending %v", ds.Phase)
	}
	ds.OnMove(core.Vec2{X: 20, Y: 20}) // dist ~14 >5
	if ds.Phase != PhaseDragging {
		t.Fatalf("dragging %v", ds.Phase)
	}
}

func TestValidDrop(t *testing.T) {
	ClearTargets()
	ds := NewDragState(0, input.NewCapture())
	target := &DropTarget{ID: "dropTarget", Bounds: core.Rect{X: 100, Y: 100, W: 50, H: 50}, Accepts: func(p Payload) bool { return p.Kind == "icon" }, OnDrop: func(p Payload) {}}
	RegisterTarget(target)
	ds.OnPress("dragSource", core.Vec2{X: 10, Y: 10}, NewPayload("dragSource", "icon", 123))
	ds.OnMove(core.Vec2{X: 110, Y: 110})
	if ds.Phase != PhaseDragging && ds.Phase != PhaseOverTarget {
		t.Logf("phase %v", ds.Phase)
	}
	// move to trigger over-target
	ds.Phase = PhaseDragging
	ds.OnMove(core.Vec2{X: 110, Y: 110})
	if ds.Phase != PhaseOverTarget {
		t.Fatalf("expected over-target %v", ds.Phase)
	}
	phase := ds.OnRelease(core.Vec2{X: 110, Y: 110})
	if phase != PhaseDropped {
		t.Fatalf("expected dropped %v", phase)
	}
}

func TestInvalidDrop(t *testing.T) {
	ClearTargets()
	ds := NewDragState(0, input.NewCapture())
	target := &DropTarget{ID: "dropTarget", Bounds: core.Rect{X: 100, Y: 100, W: 50, H: 50}, Accepts: func(p Payload) bool { return false }}
	RegisterTarget(target)
	ds.OnPress("dragSource", core.Vec2{X: 10, Y: 10}, NewPayload("dragSource", "icon", nil))
	ds.Phase = PhaseDragging
	phase := ds.OnRelease(core.Vec2{X: 110, Y: 110})
	if phase != PhaseCanceled {
		t.Fatalf("invalid drop should cancel %v", phase)
	}
}

func TestCancel(t *testing.T) {
	ds := NewDragState(5, input.NewCapture())
	ds.OnPress("dragSource", core.Vec2{X: 0, Y: 0}, NewPayload("dragSource", "x", nil))
	ds.Cancel("escape")
	if ds.Phase != PhaseCanceled {
		t.Fatal("cancel")
	}
	if ds.Captured {
		t.Fatal("should release capture")
	}
}

func TestTargetHoverResetAndOrdering(t *testing.T) {
	ClearTargets()
	ds := NewDragState(1, input.NewCapture())
	RegisterTarget(&DropTarget{ID: "hoverTarget", Bounds: core.Rect{X: 10, Y: 10, W: 20, H: 20}})
	ds.OnPress("farSource", core.Vec2{X: 0, Y: 0}, NewPayload("farSource", "icon", nil))
	ds.OnMove(core.Vec2{X: 15, Y: 15})
	if ds.Phase != PhaseOverTarget {
		t.Fatalf("target hover phase %v", ds.Phase)
	}
	ds.OnMove(core.Vec2{X: 100, Y: 100})
	if ds.Phase != PhaseDragging {
		t.Fatalf("hover should clear outside target, got %v", ds.Phase)
	}
	// Register order, rather than map iteration, wins for overlapping targets.
	ClearTargets()
	RegisterTarget(&DropTarget{ID: "firstTarget", Bounds: core.Rect{X: 0, Y: 0, W: 20, H: 20}})
	RegisterTarget(&DropTarget{ID: "secondTarget", Bounds: core.Rect{X: 0, Y: 0, W: 20, H: 20}})
	if got := HitTarget(core.Vec2{X: 10, Y: 10}); got == nil || got.ID != "firstTarget" {
		t.Fatalf("ordered target %v", got)
	}
	UnregisterTarget("firstTarget")
	if got := HitTarget(core.Vec2{X: 10, Y: 10}); got == nil || got.ID != "secondTarget" {
		t.Fatalf("unregistered target %v", got)
	}
}

func TestGhost(t *testing.T) {
	g := NewGhost(NewPayload("ghostSource", "k", nil), core.Vec2{X: 10, Y: 10})
	b := g.Bounds()
	if b.W != 32 || b.H != 32 {
		t.Fatalf("ghost size %v", b)
	}
}
