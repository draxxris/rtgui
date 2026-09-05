package dragdrop

import "github.com/draxxris/rtgui/core"

// Phase describes one drag session's current progress.
type Phase int

const (
	// PhaseIdle means that no drag session is active.
	PhaseIdle Phase = iota
	// PhasePressed means a press began below the drag threshold.
	PhasePressed
	// PhaseThresholdPending means motion has not crossed the threshold.
	PhaseThresholdPending
	// PhaseDragging means the pointer crossed the configured threshold.
	PhaseDragging
	// PhaseOverTarget means a dragging pointer is over a registered target.
	PhaseOverTarget
	// PhaseDropped means a target accepted the payload.
	PhaseDropped
	// PhaseCanceled means the session ended without an accepted drop.
	PhaseCanceled
)

// String returns a stable diagnostic name for p.
func (p Phase) String() string {
	switch p {
	case PhaseIdle:
		return "idle"
	case PhasePressed:
		return "pressed"
	case PhaseThresholdPending:
		return "threshold-pending"
	case PhaseDragging:
		return "dragging"
	case PhaseOverTarget:
		return "over-target"
	case PhaseDropped:
		return "dropped"
	case PhaseCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// Controller owns one drag session and its ordered drop-target registry.
// Call its methods from one owning goroutine.
type Controller struct {
	targets     map[string]*DropTarget
	targetOrder []string

	threshold float32
	phase     Phase
	payload   Payload
	pressPos  core.Vec2
	pointer   core.Vec2
	target    *DropTarget
}

// NewController returns a controller with threshold as its drag distance.
// Negative thresholds are treated as zero.
func NewController(threshold float32) *Controller {
	if threshold < 0 {
		threshold = 0
	}
	return &Controller{threshold: threshold}
}

// Begin starts a drag session with payload at pressPosition.
func (c *Controller) Begin(payload Payload, pressPosition core.Vec2) {
	if c == nil {
		return
	}
	c.phase = PhasePressed
	c.payload = payload
	c.pressPos = pressPosition
	c.pointer = pressPosition
	c.target = nil
}

// Move advances the active session to pointerPosition and refreshes its target.
func (c *Controller) Move(pointerPosition core.Vec2) {
	if c == nil {
		return
	}
	c.pointer = pointerPosition
	if c.phase == PhasePressed || c.phase == PhaseThresholdPending {
		dx := pointerPosition.X - c.pressPos.X
		dy := pointerPosition.Y - c.pressPos.Y
		if dx*dx+dy*dy >= c.threshold*c.threshold {
			c.phase = PhaseDragging
		} else {
			c.phase = PhaseThresholdPending
		}
	}
	if c.IsDragging() {
		c.refreshTarget()
	}
}

// Drop completes the active drag at pointerPosition. It reports PhaseDropped
// only when the first matching target accepts the payload.
func (c *Controller) Drop(pointerPosition core.Vec2) Phase {
	if c == nil {
		return PhaseCanceled
	}
	c.pointer = pointerPosition
	if !c.IsDragging() {
		c.phase = PhaseCanceled
		c.target = nil
		return c.phase
	}
	c.target = c.targetAt(pointerPosition)
	if c.target != nil && (c.target.Accepts == nil || c.target.Accepts(c.payload)) {
		if c.target.OnDrop != nil {
			c.target.OnDrop(c.payload)
		}
		c.phase = PhaseDropped
		c.target = nil
		return c.phase
	}
	c.phase = PhaseCanceled
	c.target = nil
	return c.phase
}

// Cancel ends the current session without invoking a target callback.
func (c *Controller) Cancel() {
	if c == nil {
		return
	}
	c.phase = PhaseCanceled
	c.target = nil
}

// Phase reports the controller's current drag phase.
func (c *Controller) Phase() Phase {
	if c == nil {
		return PhaseIdle
	}
	return c.phase
}

// IsDragging reports whether c has crossed its drag threshold.
func (c *Controller) IsDragging() bool {
	return c != nil && (c.phase == PhaseDragging || c.phase == PhaseOverTarget)
}

// Payload reports the current session payload.
func (c *Controller) Payload() Payload {
	if c == nil {
		return Payload{}
	}
	return c.payload
}

// PointerPosition reports the latest pointer position supplied to c.
func (c *Controller) PointerPosition() core.Vec2 {
	if c == nil {
		return core.Vec2{}
	}
	return c.pointer
}

// CurrentTarget reports the target currently under an active drag pointer.
func (c *Controller) CurrentTarget() *DropTarget {
	if c == nil {
		return nil
	}
	return c.target
}

// Ghost returns the current session payload at the current pointer position.
func (c *Controller) Ghost() Ghost {
	if c == nil {
		return Ghost{}
	}
	return NewGhost(c.payload, c.pointer)
}

// refreshTarget updates the target and phase after a drag motion or registry
// change. Its caller has already confirmed that the session is dragging.
func (c *Controller) refreshTarget() {
	c.target = c.targetAt(c.pointer)
	if c.target == nil {
		c.phase = PhaseDragging
		return
	}
	c.phase = PhaseOverTarget
}
