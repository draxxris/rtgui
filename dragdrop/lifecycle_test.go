package dragdrop

import (
	"github.com/draxxris/rtgui/core"
	"testing"
)

// TestDropCallbackCanStartNewSession verifies old completion cannot corrupt Begin.
func TestDropCallbackCanStartNewSession(t *testing.T) {
	c := NewController(0)
	mustRegisterTarget(t, c, &DropTarget{Name: "target", Bounds: core.Rect{W: 20, H: 20}, OnDrop: func(Payload) {
		c.Begin(NewPayload("next", "item", 2), core.Vec2{})
	}})
	c.Begin(NewPayload("old", "item", 1), core.Vec2{})
	c.Move(core.Vec2{X: 5, Y: 5})
	if c.Drop(core.Vec2{X: 5, Y: 5}) != PhaseDropped {
		t.Fatal("old drop did not complete")
	}
	if c.Phase() != PhasePressed || c.Payload().ID != "next" {
		t.Fatal("old callback corrupted new session")
	}
}

// TestAcceptanceReentryCannotDeliverStalePayload guards accidental predicate mutation.
func TestAcceptanceReentryCannotDeliverStalePayload(t *testing.T) {
	c := NewController(0)
	delivered := false
	mustRegisterTarget(t, c, &DropTarget{Name: "target", Bounds: core.Rect{W: 20, H: 20}, Accepts: func(Payload) bool {
		c.Begin(NewPayload("next", "item", 2), core.Vec2{})
		return true
	}, OnDrop: func(Payload) { delivered = true }})
	c.Begin(NewPayload("old", "item", 1), core.Vec2{})
	c.Move(core.Vec2{X: 5, Y: 5})
	c.Drop(core.Vec2{X: 5, Y: 5})
	if delivered || c.Payload().ID != "next" || c.Phase() != PhasePressed {
		t.Fatal("predicate mutation delivered a stale payload")
	}
}

// TestTerminalSessionReleasesPayload verifies both terminal paths drop references.
func TestTerminalSessionReleasesPayload(t *testing.T) {
	c := NewController(0)
	mustRegisterTarget(t, c, &DropTarget{Name: "target", Bounds: core.Rect{W: 20, H: 20}})
	c.Begin(NewPayload("item", "item", make([]byte, 1024)), core.Vec2{})
	c.Move(core.Vec2{X: 5, Y: 5})
	c.Drop(core.Vec2{X: 5, Y: 5})
	if c.Payload().Data != nil {
		t.Fatal("drop retained payload")
	}
	c.Begin(NewPayload("item", "item", make([]byte, 1024)), core.Vec2{})
	c.Cancel()
	if c.Payload().Data != nil {
		t.Fatal("cancel retained payload")
	}
}
