package ui

import (
	"rtgui/core"
	"rtgui/widgets"
)

// MouseEvent is one frame of polled pointer state in logical coordinates.
// Callers build Pos via ToLogical from the physical raylib mouse position.
type MouseEvent struct {
	Pos      core.Vec2
	Pressed  bool
	Down     bool
	Released bool
	Wheel    float32
}

// KeyEvent is one frame of polled keyboard state.
// Chars holds runes drained from GetCharPressed; Backspace/Escape mirror the
// corresponding raylib key queries for this frame.
type KeyEvent struct {
	Chars     []rune
	Backspace bool
	Escape    bool
}

// HandleMouse routes one frame of pointer state through hover, press, drag,
// release, and wheel handling. It reports true only when the UI consumed the
// input; hover alone, misses, empty-space clicks, and off-widget wheel all
// return false so game camera/world input can run.
func (u *UI) HandleMouse(e MouseEvent) bool {
	if u == nil {
		return false
	}
	u.updateHover(e.Pos)
	mouseHandled := u.handleWheel(e)
	mouseHandled = u.handlePress(e) || mouseHandled
	mouseHandled = u.handleDrag(e) || mouseHandled
	mouseHandled = u.handleRelease(e) || mouseHandled
	return mouseHandled
}

// HandleKey routes one frame of keyboard state to the focused textbox.
// It reports true only when the UI consumed the input: an Escape that blurred
// focus, or chars/backspace that reached a focused, enabled textbox.
// Unfocused input returns false so game hotkeys can run.
func (u *UI) HandleKey(e KeyEvent) bool {
	if u == nil {
		return false
	}
	if e.Escape {
		if u.clearFocus() {
			return true
		}
	}
	return u.handleText(e)
}

// updateHover refreshes hover states without consuming anything.
// Only pressable widgets highlight: passive kinds (panels, labels, frames,
// scroll areas, progress) stay Normal so merely passing over them shows
// nothing. The pressed widget keeps its Pressed state and the focused widget
// keeps Focused so mouse movement alone never steals capture or focus.
func (u *UI) updateHover(pos core.Vec2) {
	for _, name := range u.order {
		w := u.widgets[name]
		if w == nil || w == u.pressed || w == u.focused {
			continue
		}
		if !isPressable(w.Kind) {
			if w.Enabled {
				w.State = core.StateNormal
			}
			continue
		}
		w.UpdateHover(pos)
	}
	u.refreshDisabled()
}

// refreshDisabled pins disabled widgets to StateDisabled even when they are
// pressed or focused, mirroring the gallery draw path.
func (u *UI) refreshDisabled() {
	for _, name := range u.order {
		w := u.widgets[name]
		if w != nil && !w.Enabled {
			w.State = core.StateDisabled
		}
	}
}

// handleWheel scrolls the scroll panel under the cursor, if any.
// Wheel anywhere else is left for the game camera and returns false.
func (u *UI) handleWheel(e MouseEvent) bool {
	if e.Wheel == 0 {
		return false
	}
	target := u.topmostAt(e.Pos, core.WidgetScrollPanel)
	if target == nil {
		return false
	}
	target.ScrollBy(0, -e.Wheel*28)
	return true
}

// handlePress starts a press gesture on the topmost interactive widget.
// A miss blurs focus but returns false so the game still gets the click.
func (u *UI) handlePress(e MouseEvent) bool {
	if !e.Pressed {
		return false
	}
	hit := u.hitInteractive(e.Pos)
	if hit == nil {
		u.clearFocus()
		return false
	}
	if hit.Kind == core.WidgetTextbox {
		u.setFocus(hit)
	}
	if !hit.Press(e.Pos, u.capture) {
		return false
	}
	u.pressed = hit
	if hit.Kind == core.WidgetSlider {
		u.setSliderFromX(hit, e.Pos.X)
	}
	return true
}

// handleDrag tracks an active slider press while the button is held.
// Gestures that did not start on a widget never consume here.
func (u *UI) handleDrag(e MouseEvent) bool {
	if !e.Down || e.Pressed {
		return false
	}
	active := u.pressed
	if active == nil || active.Kind != core.WidgetSlider {
		return false
	}
	u.setSliderFromX(active, e.Pos.X)
	return true
}

// handleRelease ends the active press gesture, if any.
// A release always consumes once a press started (capture ownership), but the
// click callback fires only when Release reports a real click.
func (u *UI) handleRelease(e MouseEvent) bool {
	if !e.Released {
		return false
	}
	active := u.pressed
	if active == nil {
		return false
	}
	u.pressed = nil
	clicked := active.Release(e.Pos, u.capture)
	if clicked {
		u.fireOnClick(active.Name)
	}
	return true
}

// handleText appends chars and backspace to the focused textbox.
// It returns true only when something was routed to an enabled textbox.
func (u *UI) handleText(e KeyEvent) bool {
	target := u.focused
	if target == nil || target.Kind != core.WidgetTextbox || !target.Enabled {
		return false
	}
	if len(e.Chars) == 0 && !e.Backspace {
		return false
	}
	mutated := false
	for _, r := range e.Chars {
		if r < 32 || r == 127 {
			continue
		}
		target.TypeChar(r)
		mutated = true
	}
	if e.Backspace {
		target.Backspace()
		mutated = true
	}
	if mutated {
		u.fireOnText(target.Name, target.TextBuf.String())
	}
	return mutated
}

// setFocus moves focus to the target, blurring the previous widget.
func (u *UI) setFocus(target *widgets.Widget) {
	if u.focused == target {
		return
	}
	if u.focused != nil {
		u.focused.Blur()
	}
	u.focused = target
	if u.focused != nil {
		u.focused.Focus()
	}
}

// clearFocus blurs the focused widget. It reports whether anything blurred,
// so Escape over empty focus can pass through to the game.
func (u *UI) clearFocus() bool {
	if u.focused == nil {
		return false
	}
	u.focused.Blur()
	u.focused = nil
	return true
}

// hitInteractive returns the topmost enabled pressable widget under pos.
// Pressable in v1 is button, checkbox, textbox, slider, and dropdown; passive kinds
// (progress, label, rectangle, frame, scroll panel) never capture.
func (u *UI) hitInteractive(pos core.Vec2) *widgets.Widget {
	for i := len(u.order) - 1; i >= 0; i-- {
		w := u.widgets[u.order[i]]
		if w == nil || !w.Enabled || !isPressable(w.Kind) {
			continue
		}
		if w.HitTest(pos) {
			return w
		}
	}
	return nil
}

// topmostAt returns the topmost enabled widget of the given kind under pos.
func (u *UI) topmostAt(pos core.Vec2, kind core.WidgetKind) *widgets.Widget {
	for i := len(u.order) - 1; i >= 0; i-- {
		w := u.widgets[u.order[i]]
		if w == nil || !w.Enabled || w.Kind != kind {
			continue
		}
		if w.HitTest(pos) {
			return w
		}
	}
	return nil
}

// isPressable reports whether the kind participates in press gestures and
// hover highlighting. Passive kinds (progress, label, rectangle, frame,
// scroll panel) never capture and never highlight; dropdown presses open the
// popup via OnClick and are consumed.
func isPressable(kind core.WidgetKind) bool {
	switch kind {
	case core.WidgetButton, core.WidgetCheckbox, core.WidgetTextbox, core.WidgetSlider, core.WidgetDropdown:
		return true
	default:
		return false
	}
}

// setSliderFromX converts a logical X into a 0..1 slider value and fires
// OnChange when the value moved.
func (u *UI) setSliderFromX(w *widgets.Widget, x float32) {
	if w == nil || w.Bounds.W <= 0 {
		return
	}
	before := w.Value
	w.SetSlider((x - w.Bounds.X) / w.Bounds.W)
	if w.Value != before {
		u.fireOnChange(w.Name, w.Value)
	}
}
