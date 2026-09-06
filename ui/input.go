package ui

import (
	"unicode/utf8"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
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
// frame. An open context menu has exclusive routing; hover alone and misses
// remain available to application input. Any press or wheel hides an
// explicitly shown tooltip.
func (u *UI) HandleMouse(event MouseEvent) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	u.pointer = event.Pos
	if event.Pressed || event.Wheel != 0 {
		u.tooltipText = ""
	}
	if event.Pressed && u.pressed != nil {
		return true
	}
	if u.HasOpenMenu() {
		return u.handleOpenMenu(event)
	}
	if u.openDropdown() != nil {
		return u.handleOpenDropdown(event)
	}
	u.updateHover(event.Pos)
	u.refreshLinkTip()
	handled := u.handleWheel(event)
	handled = u.handlePress(event) || handled
	handled = u.handleDrag(event) || handled
	handled = u.handleRelease(event) || handled
	return handled
}

// HandleKey routes one physical keyboard frame. Escape closes an open menu
// first, then hides an explicit tooltip, then clears focus; text routes to
// the focused textbox. It reports edits only when text changes.
func (u *UI) HandleKey(event KeyEvent) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	if event.Escape {
		u.linkArmedSeg = -1
		u.clearLinkTip()
		if u.HasOpenMenu() {
			u.closeMenuState()
			return true
		}
		if u.tooltipText != "" {
			u.tooltipText = ""
			return true
		}
		if u.clearFocus() {
			return true
		}
		return false
	}
	return u.handleText(event)
}

// Activate performs the same kind-specific activation and callbacks as a
// successful physical click without manufacturing a pointer or hover state.
// Tab bars re-fire for the current selection; use SelectTab to change it.
// Rich text re-fires OnClick; use ActivateLink for a specific link.
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
	case core.WidgetTabBar:
		if target.TabCount() == 0 || target.SelectedTab() < 0 {
			return false
		}
	case core.WidgetTextbox:
		u.setFocus(target)
	}
	return u.activateWidget(target)
}

// SelectTab changes a tab bar selection through the shared callback path
// without manufacturing pointer or hover state. It fires OnTabSelect only
// after a real index change and always fires OnClick on a valid selection.
func (u *UI) SelectTab(name string, index int) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	target := u.Lookup(name)
	if target == nil || !target.Enabled() || target.Kind() != core.WidgetTabBar {
		return false
	}
	if index < 0 || index >= target.TabCount() {
		return false
	}
	u.commitTabSelection(target, index)
	return true
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
	if target.Kind() == core.WidgetRichText {
		u.linkArmedSeg = u.richLinkSegAt(target, event.Pos)
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
	u.clearLinkTip()
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
	if u.dropdownPopupIndex(dropdown, pos) >= 0 {
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
	index := u.dropdownPopupIndex(dropdown, pos)
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

// handleRelease ends an active gesture. Tab bars resolve the released tab
// cell and rich text resolves the released segment before firing
// kind-specific callbacks; other releases outside consume without activation.
func (u *UI) handleRelease(event MouseEvent) bool {
	if !event.Released || u.pressed == nil {
		return false
	}
	active := u.pressed
	u.pressed = nil
	if !active.Enabled() {
		u.linkArmedSeg = -1
		return true
	}
	if active.Kind() == core.WidgetTabBar {
		return u.releaseTabBar(active, event.Pos)
	}
	if active.Kind() == core.WidgetRichText {
		return u.releaseRichText(active, event.Pos)
	}
	if active.HitTest(event.Pos) {
		u.activateWidget(active)
	}
	return true
}

// releaseTabBar commits a released tab cell and closes the gesture.
// Releasing outside any cell consumes without activation. Committing the
// already-selected tab still fires OnClick but not OnTabSelect.
func (u *UI) releaseTabBar(bar *widgets.Widget, pos core.Vec2) bool {
	index := u.tabIndexAt(bar, pos)
	if index < 0 {
		return true
	}
	u.commitTabSelection(bar, index)
	return true
}

// commitTabSelection records index on bar and fires selection callbacks.
// OnTabSelect runs only after a real index change; OnClick always runs.
func (u *UI) commitTabSelection(bar *widgets.Widget, index int) {
	if bar.SetSelectedTab(index) {
		u.fireOnTabSelect(bar.Name(), index)
	}
	u.fireOnClick(bar.Name())
}

// tabIndexAt resolves the skin-aware tab cell under pos, or -1.
func (u *UI) tabIndexAt(bar *widgets.Widget, pos core.Vec2) int {
	if bar == nil || u.theme == nil {
		return -1
	}
	content := u.theme.TabContent(bar.Bounds(), core.StateNormal)
	return render.TabIndexAt(content, bar.TabCount(), pos)
}

// handleOpenMenu gives a visible context menu exclusive press routing while
// retaining library ownership of row hit testing and selection. Hover motion
// alone stays available to application input; wheel is consumed.
func (u *UI) handleOpenMenu(event MouseEvent) bool {
	u.hovered = nil
	u.clearLinkTip()
	if event.Wheel != 0 {
		return true
	}
	if event.Pressed {
		return u.pressOpenMenu(event.Pos)
	}
	if event.Released && u.menuDown {
		return u.releaseOpenMenu(event.Pos)
	}
	if event.Released {
		return true
	}
	return event.Down && u.menuDown
}

// pressOpenMenu arms a selectable row or dismisses on an outside press.
// Pressing a disabled row or separator keeps the menu open with no arm.
func (u *UI) pressOpenMenu(pos core.Vec2) bool {
	index := u.menuRowAt(pos)
	if index < 0 {
		u.closeMenuState()
		return true
	}
	if u.menuSelectable(index) {
		u.menuArmed = index
		u.menuDown = true
		return true
	}
	u.menuArmed = -1
	u.menuDown = true
	return true
}

// releaseOpenMenu commits the armed row on a matching release. Releasing
// outside dismisses; releasing on another row keeps the menu open.
func (u *UI) releaseOpenMenu(pos core.Vec2) bool {
	u.menuDown = false
	index := u.menuRowAt(pos)
	if index < 0 {
		u.closeMenuState()
		return true
	}
	if index == u.menuArmed && u.menuSelectable(index) {
		u.commitMenuRow(index)
		return true
	}
	u.menuArmed = -1
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
	case core.WidgetButton, core.WidgetCheckbox, core.WidgetTextbox, core.WidgetSlider, core.WidgetDropdown, core.WidgetTabBar, core.WidgetRichText:
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
