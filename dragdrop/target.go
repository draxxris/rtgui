package dragdrop

import "rtgui/core"

type DropTarget struct {
	ID      string
	Bounds  core.Rect
	Accepts func(Payload) bool
	OnDrop  func(Payload)
}

var (
	targets     = map[string]*DropTarget{}
	targetOrder []string
)

// RegisterTarget replaces an existing ID without changing registration order.
// Ordered lookup makes overlapping targets deterministic. Clobbering is kept
// intentionally: unlike sim.Stage.Register (which errors on duplicates),
// drag targets are transient overlay registrations where re-registering the
// same string ID updates bounds/handlers in place.
func RegisterTarget(t *DropTarget) {
	if t == nil {
		return
	}
	if _, exists := targets[t.ID]; !exists {
		targetOrder = append(targetOrder, t.ID)
	}
	targets[t.ID] = t
}

// UnregisterTarget removes the named target from the registry and order.
func UnregisterTarget(id string) {
	delete(targets, id)
	for i, targetID := range targetOrder {
		if targetID == id {
			targetOrder = append(targetOrder[:i], targetOrder[i+1:]...)
			break
		}
	}
}

func ClearTargets() {
	targets = map[string]*DropTarget{}
	targetOrder = nil
}

// HitTarget returns the first registered geometric target under pos. Acceptance
// is deliberately checked at release, so a rejected target can still receive
// deterministic hover feedback.
func HitTarget(pos core.Vec2) *DropTarget {
	for _, id := range targetOrder {
		t := targets[id]
		if t == nil {
			continue
		}
		if pos.X >= t.Bounds.X && pos.X <= t.Bounds.X+t.Bounds.W &&
			pos.Y >= t.Bounds.Y && pos.Y <= t.Bounds.Y+t.Bounds.H {
			return t
		}
	}
	return nil
}
