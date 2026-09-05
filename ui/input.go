package ui

import (
	"unicode/utf8"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// MouseEvent is one frame of polled pointer state in logical coordinates.
type MouseEvent struct {
	// Pos is the logical pointer position for this frame.
	Pos core.Vec2
	// Pressed is true on the left-button press edge.
	Pressed bool
	// Down is true while the left button remains held.
	Down bool
	// Released is true on the left-button release edge.
	Released bool
	// Wheel is the vertical wheel delta for this frame.
	Wheel float32
}

// KeyEvent is one frame of polled keyboard state.
type KeyEvent struct {
	// Chars contains runes read during this frame.
	Chars []rune
	// Backspace requests one final-rune deletion.
	Backspace bool
	// Escape requests focus clearing.
	Escape bool
}

// HandleMouse reconciles disabled owners, then routes one physical pointer
// frame. Hover alone and misses remain available to application input.
func (u *UI) HandleMouse(event MouseEvent) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	u.pointer = event.Pos
	if event.Pressed && u.pressed != nil {
		return true
	}
	if u.openDropdown() != nil {
		return u.handleOpenDropdown(event)
	}
	u.updateHover(event.Pos)
	handled := u.handleWheel(event)
	handled = u.handlePress(event) || handled
	handled = u.handleDrag(event) || handled
	handled = u.handleRelease(event) || handled
	return handled
}

// HandleKey routes one physical keyboard frame to the focused textbox. It
// reports edits only when text changes; Escape reports a real focus clear.
func (u *UI) HandleKey(event KeyEvent) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	if event.Escape && u.clearFocus() {
		return true
	}
	return u.handleText(event)
}

// Activate performs the same kind-specific activation and callbacks as a
// successful physical click without manufacturing a pointer or hover state.
func (u *UI) Activate(name string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	target := u.Lookup(name)
	if target == nil || !target.Enabled() || !isPressable(target.Kind()) {
		return false
	}
	switch target.Kind() {
	case core.WidgetDropdown:
		if u.focused == target {
			u.clearFocus()
		} else {
			u.setFocus(target)
		}
	case core.WidgetTextbox:
		u.setFocus(target)
	}
	return u.activateWidget(target)
}

// TypeText focuses a valid textbox and appends printable runes through the
// shared editing helper. Valid empty or full-buffer requests are handled but
// callbacks run only when text changes.
func (u *UI) TypeText(name, value string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	target := u.Lookup(name)
	if target == nil || !target.Enabled() || target.Kind() != core.WidgetTextbox || !utf8.ValidString(value) {
		return false
	}
	u.setFocus(target)
	u.editText(target, []rune(value), false)
	return true
}

// Focus gives keyboard focus to a valid enabled textbox or dropdown without
// changing hover or firing a callback.
func (u *UI) Focus(name string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	target := u.Lookup(name)
	if target == nil || !target.Enabled() || !isFocusable(target.Kind()) {
		return false
	}
	u.setFocus(target)
	return true
}

// reconcileInteraction clears transient owners whose widgets became disabled.
func (u *UI) reconcileInteraction() {
	if u.hovered != nil && !u.hovered.Enabled() {
		u.hovered = nil
	}
	if u.pressed != nil && !u.pressed.Enabled() {
		u.pressed = nil
	}
	if u.focused != nil && !u.focused.Enabled() {
		u.focused = nil
	}
}

// updateHover stores only the topmost enabled pressable widget under pos.
func (u *UI) updateHover(pos core.Vec2) { u.hovered = u.hitInteractive(pos) }

// handleWheel scrolls the topmost scroll panel under the pointer.
func (u *UI) handleWheel(event MouseEvent) bool {
	if event.Wheel == 0 {
		return false
	}
	target := u.topmostAt(event.Pos, core.WidgetScrollPanel)
	if target == nil {
		return false
	}
	return target.ScrollBy(0, -event.Wheel*28)
}

// handlePress starts a gesture on the topmost eligible widget. Empty-space
// presses clear focus but pass through to application input.
func (u *UI) handlePress(event MouseEvent) bool {
	if !event.Pressed {
		return false
	}
	target := u.hitInteractive(event.Pos)
	if target == nil {
		u.clearFocus()
		return false
	}
	if isFocusable(target.Kind()) {
		u.setFocus(target)
	}
	u.pressed = target
	if target.Kind() == core.WidgetSlider {
		u.setSliderFromX(target, event.Pos.X)
	}
	return true
}

// openDropdown returns the focused enabled dropdown while its popup is visible.
func (u *UI) openDropdown() *widgets.Widget {
	if u == nil || u.focused == nil || !u.focused.Enabled() || u.focused.Kind() != core.WidgetDropdown {
		return nil
	}
	return u.focused
}

// handleOpenDropdown gives a visible popup exclusive press routing while
// retaining library ownership of row hit testing and selection.
func (u *UI) handleOpenDropdown(event MouseEvent) bool {
	dropdown := u.openDropdown()
	if dropdown == nil {
		return false
	}
	u.hovered = nil
	if dropdown.HitTest(event.Pos) {
		u.hovered = dropdown
	}
	if event.Pressed {
		return u.pressOpenDropdown(dropdown, event.Pos)
	}
	if event.Released && u.pressed == dropdown {
		return u.releaseOpenDropdown(dropdown, event.Pos)
	}
	return event.Down && u.pressed == dropdown
}

// pressOpenDropdown starts control or row activation, or closes on an outside press.
func (u *UI) pressOpenDropdown(dropdown *widgets.Widget, pos core.Vec2) bool {
	if dropdown.HitTest(pos) {
		u.clearFocus()
		u.pressed = dropdown
		return true
	}
	if dropdown.DropdownIndexAt(pos) >= 0 {
		u.pressed = dropdown
		return true
	}
	u.clearFocus()
	u.pressed = nil
	return true
}

// releaseOpenDropdown commits a released row and closes the popup. Releasing
// outside cancels selection but still consumes the gesture.
func (u *UI) releaseOpenDropdown(dropdown *widgets.Widget, pos core.Vec2) bool {
	u.pressed = nil
	if dropdown.HitTest(pos) {
		u.fireOnClick(dropdown.Name())
		return true
	}
	index := dropdown.DropdownIndexAt(pos)
	u.clearFocus()
	if index >= 0 {
		dropdown.SetDropdownIndex(index)
		u.fireOnClick(dropdown.Name())
	}
	return true
}

// handleDrag maps an active slider's pointer X through the shared value helper.
func (u *UI) handleDrag(event MouseEvent) bool {
	if !event.Down || event.Pressed || u.pressed == nil || u.pressed.Kind() != core.WidgetSlider {
		return false
	}
	u.setSliderFromX(u.pressed, event.Pos.X)
	return true
}

// handleRelease ends an active gesture. Release outside consumes without activation.
func (u *UI) handleRelease(event MouseEvent) bool {
	if !event.Released || u.pressed == nil {
		return false
	}
	active := u.pressed
	u.pressed = nil
	if active.Enabled() && active.HitTest(event.Pos) {
		u.activateWidget(active)
	}
	return true
}

// activateWidget applies domain mutation before the shared click callback.
func (u *UI) activateWidget(target *widgets.Widget) bool {
	if target == nil || !target.Enabled() || !isPressable(target.Kind()) {
		return false
	}
	if target.Kind() == core.WidgetCheckbox {
		target.SetChecked(!target.Checked())
	}
	u.fireOnClick(target.Name())
	return true
}

// handleText edits the focused textbox and reports only a real mutation.
func (u *UI) handleText(event KeyEvent) bool {
	target := u.focused
	if target == nil || target.Kind() != core.WidgetTextbox || !target.Enabled() {
		return false
	}
	return u.editText(target, event.Chars, event.Backspace)
}

// editText applies printable runes and optional backspace, firing one callback
// with the final value only after at least one real mutation.
func (u *UI) editText(target *widgets.Widget, chars []rune, backspace bool) bool {
	mutated := false
	for _, char := range chars {
		if char >= 32 && char != 127 {
			mutated = target.TypeChar(char) || mutated
		}
	}
	if backspace {
		mutated = target.Backspace() || mutated
	}
	if mutated {
		u.fireOnText(target.Name(), target.Text())
	}
	return mutated
}

func (u *UI) setFocus(target *widgets.Widget) { u.focused = target }

func (u *UI) clearFocus() bool {
	if u.focused == nil {
		return false
	}
	u.focused = nil
	return true
}

// hitInteractive returns the topmost enabled pressable widget under pos.
func (u *UI) hitInteractive(pos core.Vec2) *widgets.Widget {
	for index := len(u.order) - 1; index >= 0; index-- {
		widget := u.widgets[u.order[index]]
		if widget != nil && widget.Enabled() && isPressable(widget.Kind()) && widget.HitTest(pos) {
			return widget
		}
	}
	return nil
}

// topmostAt returns the topmost enabled widget of kind under pos.
func (u *UI) topmostAt(pos core.Vec2, kind core.WidgetKind) *widgets.Widget {
	for index := len(u.order) - 1; index >= 0; index-- {
		widget := u.widgets[u.order[index]]
		if widget != nil && widget.Enabled() && widget.Kind() == kind && widget.HitTest(pos) {
			return widget
		}
	}
	return nil
}

// isPressable reports the kinds that can own a UI press gesture.
func isPressable(kind core.WidgetKind) bool {
	switch kind {
	case core.WidgetButton, core.WidgetCheckbox, core.WidgetTextbox, core.WidgetSlider, core.WidgetDropdown:
		return true
	default:
		return false
	}
}

func isFocusable(kind core.WidgetKind) bool {
	return kind == core.WidgetTextbox || kind == core.WidgetDropdown
}

// setSliderFromX converts logical X to a clamped value and reports only changes.
func (u *UI) setSliderFromX(widget *widgets.Widget, x float32) {
	if widget == nil {
		return
	}
	bounds := widget.Bounds()
	if bounds.W <= 0 {
		return
	}
	if widget.SetValue((x - bounds.X) / bounds.W) {
		u.fireOnChange(widget.Name(), widget.Value())
	}
}
