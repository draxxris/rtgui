package render

import (
	"fmt"

	"github.com/draxxris/rtgui/core"
)

// DrawRecorder retains a bounded snapshot of one diagnostic draw frame.
type DrawRecorder struct {
	calls          []DrawCall
	maxCalls       int
	truncated      bool
	lastWidgetInfo core.WidgetInfo
}

// NewDrawRecorder returns a recorder that retains at most maxCalls per frame.
func NewDrawRecorder(maxCalls int) (*DrawRecorder, error) {
	if maxCalls <= 0 {
		return nil, fmt.Errorf("render: draw recorder limit must be positive, got %d", maxCalls)
	}
	return &DrawRecorder{
		calls:    make([]DrawCall, 0, maxCalls),
		maxCalls: maxCalls,
	}, nil
}

// Calls returns a defensive snapshot of the current frame's retained calls.
func (r *DrawRecorder) Calls() []DrawCall {
	if r == nil {
		return nil
	}
	calls := make([]DrawCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

// Truncated reports whether the current frame exceeded the recorder limit.
func (r *DrawRecorder) Truncated() bool {
	return r != nil && r.truncated
}

// LastWidgetInfo reports the latest whole-widget snapshot in the current frame.
func (r *DrawRecorder) LastWidgetInfo() core.WidgetInfo {
	if r == nil {
		return core.WidgetInfo{}
	}
	return r.lastWidgetInfo
}

// beginFrame reuses the bounded call storage for a new diagnostic frame.
func (r *DrawRecorder) beginFrame() {
	if r == nil {
		return
	}
	r.calls = r.calls[:0]
	r.truncated = false
	r.lastWidgetInfo = core.WidgetInfo{}
}

// record appends call only while the configured bound has room.
func (r *DrawRecorder) record(call DrawCall) {
	if r == nil {
		return
	}
	if len(r.calls) >= r.maxCalls {
		r.truncated = true
		return
	}
	r.calls = append(r.calls, call)
}

// setLastWidgetInfo records the latest whole-widget snapshot for this frame.
func (r *DrawRecorder) setLastWidgetInfo(info core.WidgetInfo) {
	if r != nil {
		r.lastWidgetInfo = info
	}
}
