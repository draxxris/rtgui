package dragdrop

import (
	"rtgui/core"
	"rtgui/input"
)

// Drag phases: idle, pressed, threshold-pending, dragging, over-target, dropped, canceled

type Phase int

const (
	PhaseIdle Phase = iota
	PhasePressed
	PhaseThresholdPending
	PhaseDragging
	PhaseOverTarget
	PhaseDropped
	PhaseCanceled
)

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

// hashSourceID derives the internal numeric capture ID from the external
// string source name. It mirrors widgets.hashName and layout.hashID (FNV-1a
// with offset 2166136261 and prime 16777619, with 0 remapped to 1 since
// Capture zero means no capture). Duplicated here intentionally: dragdrop
// must not import widgets or layout just for the hash.
func hashSourceID(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	if h == 0 {
		h = 1
	}
	return h
}

type DragState struct {
	Phase      Phase
	SourceID   string
	Payload    Payload
	PressPos   core.Vec2
	CurrentPos core.Vec2
	Threshold  float32
	LongPress  bool
	Captured   bool
	GhostPos   core.Vec2
	Capture    *input.Capture
}

func NewDragState(threshold float32, capture *input.Capture) *DragState {
	if capture == nil {
		capture = input.NewCapture()
	}
	return &DragState{Phase: PhaseIdle, Threshold: threshold, Capture: capture}
}

// OnPress starts a drag press from the named source, recording positions
// and payload and capturing pointer input under the hashed source ID.
// Starting a new press always resets a previous terminal/hover state.
func (d *DragState) OnPress(sourceID string, pos core.Vec2, payload Payload) {
	d.Phase = PhasePressed
	d.SourceID = sourceID
	d.PressPos = pos
	d.CurrentPos = pos
	d.Payload = payload
	d.GhostPos = pos
	d.Captured = d.capture().Set(hashSourceID(sourceID)) == nil
}

func (d *DragState) OnMove(pos core.Vec2) {
	d.CurrentPos = pos
	d.GhostPos = pos
	if d.Phase == PhasePressed || d.Phase == PhaseThresholdPending {
		dx := pos.X - d.PressPos.X
		dy := pos.Y - d.PressPos.Y
		dist := dx*dx + dy*dy
		if dist >= d.Threshold*d.Threshold {
			d.Phase = PhaseDragging
		} else {
			d.Phase = PhaseThresholdPending
		}
	}
	// Recompute hover on every drag motion. In particular, moving out of a
	// target returns to dragging, and moving between targets updates the
	// highlighted target rather than leaving a stale OverTarget phase.
	if d.Phase == PhaseDragging || d.Phase == PhaseOverTarget {
		if HitTarget(pos) != nil {
			d.Phase = PhaseOverTarget
		} else {
			d.Phase = PhaseDragging
		}
	}
}

func (d *DragState) OnRelease(pos core.Vec2) Phase {
	d.CurrentPos = pos
	defer func() {
		d.capture().Release()
		d.Captured = false
	}()
	if d.Phase == PhaseDragging || d.Phase == PhaseOverTarget {
		if target := HitTarget(pos); target != nil {
			accepts := target.Accepts == nil || target.Accepts(d.Payload)
			if accepts {
				if target.OnDrop != nil {
					target.OnDrop(d.Payload)
				}
				d.Phase = PhaseDropped
				return d.Phase
			}
		}
	}
	// Invalid drops preserve the source payload/state and cancel.
	d.Phase = PhaseCanceled
	return d.Phase
}

func (d *DragState) Cancel(reason string) {
	_ = reason
	d.Phase = PhaseCanceled
	d.capture().Release()
	d.Captured = false
}
func (d *DragState) IsDragging() bool { return d.Phase == PhaseDragging || d.Phase == PhaseOverTarget }

func (d *DragState) capture() *input.Capture {
	if d.Capture == nil {
		d.Capture = input.NewCapture()
	}
	return d.Capture
}
