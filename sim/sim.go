package sim

import (
	"errors"
	"fmt"
	"sync"

	"rtgui/core"
	"rtgui/input"
	"rtgui/widgets"
)

// Sentinel errors returned by Register.
var (
	// ErrNilStage is returned when Register is called on a nil Stage.
	ErrNilStage = errors.New("sim: nil stage")
	// ErrNilWidget is returned when Register is called with a nil widget.
	ErrNilWidget = errors.New("sim: nil widget")
	// ErrEmptyName is returned when a widget carries no external name.
	ErrEmptyName = errors.New("sim: empty widget name")
	// ErrDuplicateWidget is returned when a name is already registered.
	ErrDuplicateWidget = errors.New("sim: duplicate widget name")
)

// Stage is an instance-owned registry plus human-event driver.
//
// The registry maps an external string name to a widget, preserves
// insertion order, and tracks the currently focused widget. The capture
// is either owned (created by NewStage) or borrowed (supplied to
// NewStageWithCapture) and is used for press/release driving.
type Stage struct {
	mu      sync.RWMutex
	widgets map[string]*widgets.Widget
	order   []string
	capture *input.Capture
	focused *widgets.Widget
}

// NewStage returns an empty Stage that owns a fresh input capture.
func NewStage() *Stage {
	return NewStageWithCapture(nil)
}

// NewStageWithCapture returns an empty Stage using the given capture.
// A nil capture is replaced with a fresh one so the Stage always has a
// usable capture for press/release driving.
func NewStageWithCapture(c *input.Capture) *Stage {
	if c == nil {
		c = input.NewCapture()
	}
	return &Stage{
		widgets: make(map[string]*widgets.Widget),
		capture: c,
	}
}

// widgetName extracts the external string identity of a widget.
//
// It reads the Name field directly; an empty name means the widget is
// nameless so registration rejects it instead of panicking.
func widgetName(w *widgets.Widget) string {
	if w == nil {
		return ""
	}
	return w.Name
}

// widgetCenter returns the center point of a widget's bounds in logical
// coordinates. Center-click keeps the harness headless and avoids any
// window-system dependency.
func widgetCenter(w *widgets.Widget) core.Vec2 {
	return core.Vec2{
		X: w.Bounds.X + w.Bounds.W/2,
		Y: w.Bounds.Y + w.Bounds.H/2,
	}
}

// Register adds a widget to the Stage registry under its external name.
// It returns an error when the stage or widget is nil, when the widget
// carries an empty name, or when the name is already registered. A
// duplicate registration never clobbers the existing entry and never
// changes insertion order.
func (s *Stage) Register(w *widgets.Widget) error {
	if s == nil {
		return ErrNilStage
	}
	if w == nil {
		return ErrNilWidget
	}
	name := widgetName(w)
	if name == "" {
		return fmt.Errorf("%w", ErrEmptyName)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.widgets == nil {
		s.widgets = make(map[string]*widgets.Widget)
	}
	if _, exists := s.widgets[name]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateWidget, name)
	}
	s.widgets[name] = w
	s.order = append(s.order, name)
	return nil
}

// lookup returns the widget registered under name, or nil when the stage
// is nil or the name is unknown. It holds only a read lock.
func (s *Stage) lookup(name string) *widgets.Widget {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.widgets == nil {
		return nil
	}
	return s.widgets[name]
}

// captureFor returns the Stage capture for press/release driving. A nil
// stage yields nil; widget press/release primitives already tolerate a
// nil capture.
func (s *Stage) captureFor() *input.Capture {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.capture
}

// switchFocus blurs the previously focused widget (when it differs from
// the target) and focuses the target, recording it as focused. Callers
// must have already validated the target.
func (s *Stage) switchFocus(target *widgets.Widget) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.focused
	if prev == target {
		return
	}
	if prev != nil {
		prev.Blur()
	}
	target.Focus()
	s.focused = target
}

// Click simulates a human click at the center of the named widget. It
// drives hover, press, and release through the existing widget
// primitives and always leaves the capture released. It reports false
// when the stage is nil, the name is unknown, the widget is disabled,
// or the press misses.
func (s *Stage) Click(name string) bool {
	w := s.lookup(name)
	if w == nil {
		return false
	}
	if !w.Enabled {
		return false
	}
	center := widgetCenter(w)
	cap := s.captureFor()
	w.UpdateHover(center)
	if !w.Press(center, cap) {
		if cap != nil {
			cap.Release()
		}
		return false
	}
	return w.Release(center, cap)
}

// Type simulates human typing into the named textbox. It focuses the
// target first (blurring the previously focused widget) and then appends
// each rune of text via TypeChar, which keeps multi-byte UTF-8 safe. An
// empty text is a no-op success that still focuses the target. It
// reports false when the stage is nil, the name is unknown, the target
// is not a textbox, or the target is disabled.
func (s *Stage) Type(name, text string) bool {
	w := s.lookup(name)
	if w == nil {
		return false
	}
	if w.Kind != core.WidgetTextbox {
		return false
	}
	if !w.Enabled {
		return false
	}
	s.switchFocus(w)
	for _, r := range text {
		w.TypeChar(r)
	}
	return true
}
