package dragdrop

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

func mustRegisterTarget(t *testing.T, controller *Controller, target *DropTarget) {
	t.Helper()
	if err := controller.RegisterTarget(target); err != nil {
		t.Fatalf("RegisterTarget(%+v): %v", target, err)
	}
}

// TestControllerThresholdTransitions checks the pending and dragging phases.
func TestControllerThresholdTransitions(t *testing.T) {
	controller := NewController(5)
	payload := NewPayload("dragSource", "icon", "data")
	controller.Begin(payload, core.Vec2{X: 10, Y: 10})
	if got := controller.Phase(); got != PhasePressed {
		t.Fatalf("phase after Begin = %v, want pressed", got)
	}
	controller.Move(core.Vec2{X: 12, Y: 12})
	if got := controller.Phase(); got != PhaseThresholdPending {
		t.Fatalf("phase below threshold = %v, want pending", got)
	}
	controller.Move(core.Vec2{X: 15, Y: 10})
	if got := controller.Phase(); got != PhaseDragging {
		t.Fatalf("phase at threshold = %v, want dragging", got)
	}
}

// TestControllerAcceptedAndRejectedDrops checks target acceptance callbacks.
func TestControllerAcceptedAndRejectedDrops(t *testing.T) {
	accepted := NewController(0)
	var delivered Payload
	target := &DropTarget{
		Name:   "accept",
		Bounds: core.Rect{X: 100, Y: 100, W: 50, H: 50},
		Accepts: func(payload Payload) bool {
			return payload.Kind == "icon"
		},
		OnDrop: func(payload Payload) {
			delivered = payload
		},
	}
	mustRegisterTarget(t, accepted, target)
	payload := NewPayload("dragSource", "icon", 123)
	accepted.Begin(payload, core.Vec2{X: 10, Y: 10})
	accepted.Move(core.Vec2{X: 110, Y: 110})
	if got := accepted.Phase(); got != PhaseOverTarget {
		t.Fatalf("phase over accepted target = %v", got)
	}
	if got := accepted.Drop(core.Vec2{X: 110, Y: 110}); got != PhaseDropped {
		t.Fatalf("accepted Drop = %v, want dropped", got)
	}
	if delivered != payload {
		t.Fatalf("delivered payload = %+v, want %+v", delivered, payload)
	}

	rejected := NewController(0)
	mustRegisterTarget(t, rejected, &DropTarget{
		Name:    "reject",
		Bounds:  target.Bounds,
		Accepts: func(Payload) bool { return false },
	})
	rejected.Begin(payload, core.Vec2{})
	rejected.Move(core.Vec2{X: 110, Y: 110})
	if got := rejected.Drop(core.Vec2{X: 110, Y: 110}); got != PhaseCanceled {
		t.Fatalf("rejected Drop = %v, want canceled", got)
	}
}

// TestControllerCancel checks cancellation clears the active target.
func TestControllerCancel(t *testing.T) {
	controller := NewController(1)
	mustRegisterTarget(t, controller, &DropTarget{Name: "target", Bounds: core.Rect{W: 20, H: 20}})
	controller.Begin(NewPayload("source", "kind", nil), core.Vec2{})
	controller.Move(core.Vec2{X: 1, Y: 0})
	if controller.CurrentTarget() == nil {
		t.Fatal("drag should hover target before cancellation")
	}
	controller.Cancel()
	if got := controller.Phase(); got != PhaseCanceled {
		t.Fatalf("Cancel phase = %v", got)
	}
	if controller.CurrentTarget() != nil {
		t.Fatal("Cancel must clear the active target")
	}
}

// TestControllerTargetReplacementAndRemoval checks registry refresh behavior.
func TestControllerTargetReplacementAndRemoval(t *testing.T) {
	controller := NewController(0)
	first := &DropTarget{Name: "target", Bounds: core.Rect{W: 20, H: 20}}
	mustRegisterTarget(t, controller, first)
	controller.Begin(NewPayload("source", "kind", nil), core.Vec2{})
	controller.Move(core.Vec2{X: 10, Y: 10})
	if controller.CurrentTarget() != first {
		t.Fatal("first target was not selected")
	}

	replacement := &DropTarget{Name: "target", Bounds: core.Rect{X: 40, Y: 40, W: 20, H: 20}}
	mustRegisterTarget(t, controller, replacement)
	if controller.CurrentTarget() != nil || controller.Phase() != PhaseDragging {
		t.Fatal("replacement must refresh a target that no longer contains the pointer")
	}
	controller.Move(core.Vec2{X: 50, Y: 50})
	if controller.CurrentTarget() != replacement {
		t.Fatal("replacement target was not selected")
	}
	if !controller.RemoveTarget("target") {
		t.Fatal("RemoveTarget must report the registered target")
	}
	if controller.CurrentTarget() != nil || controller.Phase() != PhaseDragging {
		t.Fatal("removal must clear the current target")
	}
	if controller.RemoveTarget("target") {
		t.Fatal("removing a missing target must report false")
	}
}

// TestControllerFirstRegistrationWinsOverlaps checks deterministic target order.
func TestControllerFirstRegistrationWinsOverlaps(t *testing.T) {
	controller := NewController(0)
	var drops []string
	first := &DropTarget{
		Name:   "first",
		Bounds: core.Rect{W: 20, H: 20},
		OnDrop: func(Payload) {
			drops = append(drops, "first")
		},
	}
	second := &DropTarget{
		Name:   "second",
		Bounds: core.Rect{W: 20, H: 20},
		OnDrop: func(Payload) {
			drops = append(drops, "second")
		},
	}
	mustRegisterTarget(t, controller, first)
	mustRegisterTarget(t, controller, second)
	controller.Begin(NewPayload("source", "kind", nil), core.Vec2{})
	controller.Move(core.Vec2{X: 10, Y: 10})
	if got := controller.CurrentTarget(); got != first {
		t.Fatalf("overlap target = %+v, want first", got)
	}
	if got := controller.Drop(core.Vec2{X: 10, Y: 10}); got != PhaseDropped {
		t.Fatalf("Drop = %v", got)
	}
	if len(drops) != 1 || drops[0] != "first" {
		t.Fatalf("drop order = %v, want first", drops)
	}

	replacement := &DropTarget{Name: "first", Bounds: first.Bounds, OnDrop: first.OnDrop}
	mustRegisterTarget(t, controller, replacement)
	controller.Begin(NewPayload("source", "kind", nil), core.Vec2{})
	controller.Move(core.Vec2{X: 10, Y: 10})
	if got := controller.CurrentTarget(); got != replacement {
		t.Fatalf("replacement changed precedence: got %+v", got)
	}
}

// TestControllerHoverResetsOutsideTarget checks target exit transitions.
func TestControllerHoverResetsOutsideTarget(t *testing.T) {
	controller := NewController(1)
	mustRegisterTarget(t, controller, &DropTarget{Name: "target", Bounds: core.Rect{X: 10, Y: 10, W: 20, H: 20}})
	controller.Begin(NewPayload("source", "kind", nil), core.Vec2{})
	controller.Move(core.Vec2{X: 15, Y: 15})
	if controller.Phase() != PhaseOverTarget || controller.CurrentTarget() == nil {
		t.Fatalf("inside target phase/target = %v/%+v", controller.Phase(), controller.CurrentTarget())
	}
	controller.Move(core.Vec2{X: 100, Y: 100})
	if controller.Phase() != PhaseDragging || controller.CurrentTarget() != nil {
		t.Fatalf("outside target phase/target = %v/%+v", controller.Phase(), controller.CurrentTarget())
	}
}

// TestControllersAreIsolated checks that controller state is not shared.
func TestControllersAreIsolated(t *testing.T) {
	first := NewController(0)
	second := NewController(0)
	firstTarget := &DropTarget{Name: "first", Bounds: core.Rect{W: 20, H: 20}}
	secondTarget := &DropTarget{Name: "second", Bounds: core.Rect{X: 50, Y: 50, W: 20, H: 20}}
	mustRegisterTarget(t, first, firstTarget)
	mustRegisterTarget(t, second, secondTarget)

	first.Begin(NewPayload("first-source", "kind", nil), core.Vec2{})
	first.Move(core.Vec2{X: 10, Y: 10})
	second.Begin(NewPayload("second-source", "kind", nil), core.Vec2{})
	second.Move(core.Vec2{X: 10, Y: 10})
	if first.CurrentTarget() != firstTarget {
		t.Fatal("first controller lost its own target")
	}
	if second.CurrentTarget() != nil || second.Phase() != PhaseDragging {
		t.Fatal("second controller saw first controller's target")
	}
	if first.Payload().ID != "first-source" || second.Payload().ID != "second-source" {
		t.Fatal("controllers shared a source payload")
	}
}

// TestControllerPayloadAndGhostUseOnePosition checks the single pointer state.
func TestControllerPayloadAndGhostUseOnePosition(t *testing.T) {
	var controller Controller
	if err := controller.RegisterTarget(&DropTarget{Name: "target", Bounds: core.Rect{W: 20, H: 20}}); err != nil {
		t.Fatalf("zero-value RegisterTarget: %v", err)
	}
	payload := NewPayload("source-id", "icon", "data")
	controller.Begin(payload, core.Vec2{X: 3, Y: 4})
	controller.Move(core.Vec2{X: 10, Y: 12})
	if got := controller.Payload(); got != payload {
		t.Fatalf("payload = %+v, want %+v", got, payload)
	}
	if got := controller.PointerPosition(); got != (core.Vec2{X: 10, Y: 12}) {
		t.Fatalf("pointer = %+v", got)
	}
	ghost := controller.Ghost()
	if ghost.Payload.ID != "source-id" || ghost.Pos != controller.PointerPosition() {
		t.Fatalf("ghost = %+v, pointer = %+v", ghost, controller.PointerPosition())
	}
	if bounds := ghost.Bounds(); bounds.W != 32 || bounds.H != 32 {
		t.Fatalf("ghost bounds = %+v", bounds)
	}
}

// TestControllerRejectsInvalidTargets checks registration validation errors.
func TestControllerRejectsInvalidTargets(t *testing.T) {
	controller := NewController(0)
	if err := controller.RegisterTarget(nil); err == nil || err.Error() != "dragdrop: nil target" {
		t.Fatalf("nil target error = %v", err)
	}
	if err := controller.RegisterTarget(&DropTarget{}); err == nil || err.Error() != "dragdrop: empty target name" {
		t.Fatalf("empty target error = %v", err)
	}
	var nilController *Controller
	if err := nilController.RegisterTarget(&DropTarget{Name: "target"}); err == nil || err.Error() != "dragdrop: nil controller" {
		t.Fatalf("nil controller error = %v", err)
	}
}
