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
	// RightPressed is the right-button press edge for menus or drag cancellation.
	RightPressed bool
	// Wheel is the vertical wheel delta for this frame.
	Wheel float32
}

// KeyEvent is one frame of polled keyboard state.
type KeyEvent struct {
	// Chars contains runes read during this frame.
	Chars []rune
	// Backspace requests deletion of the selection or rune before the caret.
	Backspace bool
	// Delete requests deletion of the selection or rune after the caret.
	Delete bool
	// Escape requests focus clearing.
	Escape bool
	// Left requests caret motion one rune left.
	Left bool
	// Right requests caret motion one rune right.
	Right bool
	// Home requests caret motion to the start of the text.
	Home bool
	// End requests caret motion to the end of the text.
	End bool
	// Shift extends the selection during Left/Right/Home/End motion.
	Shift bool
	// SelectAll requests full-text selection (Ctrl-A).
	SelectAll bool
	// Copy requests copying the selection to the clipboard (Ctrl-C).
	Copy bool
	// Cut requests cutting the selection to the clipboard (Ctrl-X).
	Cut bool
	// Paste requests inserting clipboard text at the caret (Ctrl-V).
	Paste bool
	// Hotkeys carries bare hotkey press edges for this frame as canonical
	// runes (R for KeyR). The library matches them against OnHotkey
	// registrations after text, so callers build one KeyEvent per frame
	// instead of polling IsKeyPressed separately.
	Hotkeys []rune
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
	if event.Pressed || event.RightPressed || event.Wheel != 0 {
		u.HideTooltip()
	}
	if event.RightPressed {
		return u.handleRightPress(event.Pos)
	}
	if event.Pressed && u.mouseCaptured {
		return true
	}
	if u.HasOpenMenu() {
		captured := u.finishMouse(event, false)
		return u.handleOpenMenu(event) || captured
	}
	if u.openDropdown() != nil {
		captured := u.finishMouse(event, false)
		return u.handleOpenDropdown(event) || captured
	}
	if u.routeDrag(event) {
		return true
	}
	return u.handleWidgetMouse(event)
}

// handleWidgetMouse dispatches a non-modal frame through ordinary widget gestures.
func (u *UI) handleWidgetMouse(event MouseEvent) bool {
	u.updateHover(event.Pos)
	u.refreshLinkTip()
	handled := u.handleWheel(event)
	handled = u.handlePress(event) || handled
	u.capturePress(event)
	handled = u.handleDrag(event) || handled
	handled = u.finishMouse(event, handled)
	handled = u.handleRelease(event) || handled
	return handled
}

// handleRightPress cancels drags or opens menus without leaking world input.
func (u *UI) handleRightPress(pos core.Vec2) bool {
	if u.HasOpenMenu() {
		return true
	}
	if u.dragSource != nil {
		u.cancelDrag()
		u.pressed = nil
		return true
	}
	if u.contextMenuHandler != nil {
		u.contextMenuHandler(pos)
		return true
	}
	return u.hitSurface(pos) != nil
}

// HandleKey routes one physical keyboard frame. Escape dismisses menu,
// tooltip, keyboard focus, then container focus; modal popups swallow intent;
// text wins over scoped hotkeys; hotkeys consume per registration; the rest
// passes to the host game. It reports consumption, not mutation: printable
// typing into a full buffer still consumes so the game never observes it.
func (u *UI) HandleKey(event KeyEvent) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	if event.Escape {
		return u.handleEscape()
	}
	if u.HasOpenMenu() || u.openDropdown() != nil {
		return hasKeyIntent(event)
	}
	if u.WantsTextInput() {
		mutated := u.handleText(event)
		if mutated || hasEditingIntent(event) {
			return true
		}
	}
	return u.fireScopedHotkeys(event.Hotkeys)
}

// handleEscape dismisses one layer per frame in menu, tooltip, keyboard
// focus, then container focus order. It reports whether any layer owned the
// press; empty state passes through so the host game still observes Escape.
func (u *UI) handleEscape() bool {
	if u.dragSource != nil {
		u.cancelDrag()
		u.pressed = nil
		return true
	}
	u.linkArmedSeg = -1
	u.clearLinkTip()
	if u.HasOpenMenu() {
		u.closeMenuState()
		return true
	}
	if u.HideTooltip() {
		return true
	}
	if u.clearFocus() {
		return true
	}
	return u.clearActiveFrame()
}

// hasKeyIntent reports whether a keyboard frame carries any actionable
// content. Empty poll frames never consume, even while modal state stands,
// so the host game is not starved by an open menu with no fresh presses.
// Escape is excluded: HandleKey returns for it before this helper runs.
func hasKeyIntent(event KeyEvent) bool {
	if len(event.Chars) > 0 || len(event.Hotkeys) > 0 {
		return true
	}
	return hasEditingIntent(event)
}

// hasEditingIntent consumes editing keys even at a caret or buffer boundary.
func hasEditingIntent(event KeyEvent) bool {
	return hasPrintableChars(event.Chars) || event.Backspace || event.Delete || event.Left || event.Right || event.Home || event.End || event.SelectAll || event.Copy || event.Cut || event.Paste
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
	if target == nil {
		u.diagnose("ui.Activate: widget %q not found", name)
		return false
	}
	if !u.available(target) {
		u.diagnose("ui.Activate: widget %q is disabled", name)
		return false
	}
	if !isPressable(target.Kind()) {
		u.diagnose("ui.Activate: widget %q is %v, not pressable", name, target.Kind())
		return false
	}
	switch w := target.(type) {
	case *widgets.Dropdown:
		if u.focused == target {
			u.clearFocus()
		} else {
			u.setFocus(target)
		}
	case *widgets.TabBar:
		if w.TabCount() == 0 || w.SelectedTab() < 0 {
			u.diagnose("ui.Activate: tab bar %q has no valid selection", name)
			return false
		}
	case *widgets.List:
		if _, ok := w.Selected(); !ok {
			u.diagnose("ui.Activate: list %q has no valid selection", name)
			return false
		}
	case *widgets.Textbox:
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
	if target == nil {
		u.diagnose("ui.SelectTab: widget %q not found", name)
		return false
	}
	tb, ok := target.(*widgets.TabBar)
	if !ok {
		u.diagnose("ui.SelectTab: widget %q is %v, expected WidgetTabBar", name, target.Kind())
		return false
	}
	if !u.available(tb) {
		u.diagnose("ui.SelectTab: widget %q is disabled", name)
		return false
	}
	if index < 0 || index >= tb.TabCount() {
		u.diagnose("ui.SelectTab: index %d out of range [0, %d) for %q", index, tb.TabCount(), name)
		return false
	}
	u.commitTabSelection(tb, index)
	return true
}

// TypeText focuses a valid textbox and inserts printable runes at the caret
// through the shared editing helper. Valid empty or full-buffer requests are
// handled but callbacks run only when text changes.
func (u *UI) TypeText(name, value string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	target := u.Lookup(name)
	if target == nil {
		u.diagnose("ui.TypeText: widget %q not found", name)
		return false
	}
	tb, ok := target.(*widgets.Textbox)
	if !ok {
		u.diagnose("ui.TypeText: widget %q is %v, expected WidgetTextbox", name, target.Kind())
		return false
	}
	if !u.available(tb) {
		u.diagnose("ui.TypeText: widget %q is disabled", name)
		return false
	}
	if !utf8.ValidString(value) {
		u.diagnose("ui.TypeText: invalid UTF-8 string for %q", name)
		return false
	}
	u.setFocus(tb)
	if u.editText(tb, []rune(value), false, false) {
		u.fireOnText(tb.Name(), tb.Text())
	}
	return true
}

// Focus gives keyboard focus to a valid enabled textbox or dropdown without
// changing hover or firing a callback. Focusing inside a frame keeps that
// frame active for glow; focusing outside every frame clears container focus.
func (u *UI) Focus(name string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	target := u.Lookup(name)
	if target == nil {
		u.diagnose("ui.Focus: widget %q not found", name)
		return false
	}
	if !u.available(target) {
		u.diagnose("ui.Focus: widget %q is disabled", name)
		return false
	}
	if !isFocusable(target.Kind()) {
		u.diagnose("ui.Focus: widget %q (%v) is not focusable", name, target.Kind())
		return false
	}
	u.setFocus(target)
	return true
}

// reconcileInteraction clears transient owners whose widgets became disabled.
// A disabled focused textbox also forgets its selection so it never
// resurfaces on re-enable. A disabled active frame releases container focus.
func (u *UI) reconcileInteraction() {
	if u == nil {
		return
	}
	if u.hovered != nil && !u.available(u.hovered) {
		u.setHovered(nil)
	}
	if u.pressed != nil && !u.available(u.pressed) {
		u.pressed = nil
	}
	if u.focused != nil && !u.available(u.focused) {
		u.clearFocus()
	}
	if u.activeFrame != nil && !u.available(u.activeFrame) {
		u.clearActiveFrame()
	}
	if u.scrollThumbDragging != nil && !u.available(u.scrollThumbDragging) {
		u.scrollThumbDragging = nil
	}
	if u.scrollThumbHovered != nil && !u.available(u.scrollThumbHovered) {
		u.scrollThumbHovered = nil
	}
	u.reconcileAuxiliary()
}

// updateHover stores only the topmost enabled pressable widget under pos.
func (u *UI) updateHover(pos core.Vec2) {
	u.setHovered(u.hitSurface(pos))
	u.updateScrollThumbHover(pos)
}

// handleWheel scrolls the innermost overflowing scroll container under the
// pointer. Panels and lists share one owner walk so the deepest container
// owns the gesture; other surfaces only block pass-through.
func (u *UI) handleWheel(event MouseEvent) bool {
	if event.Wheel == 0 {
		return false
	}
	owner := u.innermostScrollOwner(event.Pos)
	if owner == nil {
		return u.hitSurface(event.Pos) != nil
	}
	offset, max, _ := scrollState(owner)
	if max <= 0 {
		return true
	}
	setScrollState(owner, offset-event.Wheel*28)
	return true
}

// handlePress starts a gesture on the topmost eligible widget. Frame
// background presses set container focus and consume; empty-space presses
// clear both keyboard and container focus but pass through to the host game.
func (u *UI) handlePress(event MouseEvent) bool {
	if !event.Pressed {
		return false
	}
	if u.handleScrollbarPress(event.Pos) {
		return true
	}
	target := u.hitInteractive(event.Pos)
	frame := u.innermostFrameAt(event.Pos)
	if target == nil && frame == nil {
		u.clearFocus()
		u.clearActiveFrame()
		return u.hitSurface(event.Pos) != nil
	}
	if frame != nil {
		u.setActiveFrame(frame)
	}
	if target == nil {
		u.clearFocus()
		return true
	}
	if frame == nil {
		u.clearActiveFrame()
	}
	if isFocusable(target.Kind()) {
		u.setFocus(target)
	}
	u.pressed = target
	if sl, ok := target.(*widgets.Slider); ok {
		u.setSliderFromX(sl, event.Pos.X)
	}
	if tb, ok := target.(*widgets.Textbox); ok && u.theme != nil {
		u.placeTextboxCaret(tb, event.Pos.X)
	}
	if rt, ok := target.(*widgets.RichText); ok {
		u.linkArmedSeg = u.richLinkSegAt(rt, event.Pos)
	}
	return true
}

// openDropdown returns the focused enabled dropdown while its popup is visible.
func (u *UI) openDropdown() *widgets.Dropdown {
	if u == nil || u.focused == nil || !u.focused.Enabled() {
		return nil
	}
	dd, ok := u.focused.(*widgets.Dropdown)
	if !ok {
		return nil
	}
	return dd
}

// handleOpenDropdown gives a visible popup exclusive press routing while
// retaining library ownership of row hit testing and selection.
func (u *UI) handleOpenDropdown(event MouseEvent) bool {
	dropdown := u.openDropdown()
	if dropdown == nil {
		return false
	}
	u.setHovered(nil)
	u.clearLinkTip()
	if dropdown.HitTest(event.Pos) {
		u.setHovered(dropdown)
	}
	if event.Pressed {
		return u.pressOpenDropdown(dropdown, event.Pos)
	}
	if event.Released && u.pressed == dropdown {
		return u.releaseOpenDropdown(dropdown, event.Pos)
	}
	return event.Wheel != 0 || event.Down || event.Released
}

// pressOpenDropdown starts control or row activation, or closes on an outside press.
func (u *UI) pressOpenDropdown(dropdown *widgets.Dropdown, pos core.Vec2) bool {
	if dropdown.HitTest(pos) {
		u.clearFocus()
		u.pressed = dropdown
		u.mouseCaptured = true
		return true
	}
	if u.dropdownPopupIndex(dropdown, pos) >= 0 {
		u.pressed = dropdown
		return true
	}
	u.clearFocus()
	u.pressed = nil
	u.mouseCaptured = true
	return true
}

// releaseOpenDropdown commits a released row and closes the popup. Releasing
// outside cancels selection but still consumes the gesture.
func (u *UI) releaseOpenDropdown(dropdown *widgets.Dropdown, pos core.Vec2) bool {
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

// handleDrag maps an active slider's pointer X or scrollbar drag through the shared value helper.
func (u *UI) handleDrag(event MouseEvent) bool {
	if !event.Down || event.Pressed {
		return false
	}
	if u.handleScrollbarDrag(event.Pos) {
		return true
	}
	if u.pressed == nil {
		return false
	}
	if sl, ok := u.pressed.(*widgets.Slider); ok {
		u.setSliderFromX(sl, event.Pos.X)
		return true
	}
	return false
}

// handleRelease ends an active gesture. Tab bars resolve the released tab
// cell and rich text resolves the released segment before firing
// kind-specific callbacks; other releases outside consume without activation.
func (u *UI) handleRelease(event MouseEvent) bool {
	if !event.Released {
		return false
	}
	if u.handleScrollbarRelease() {
		return true
	}
	if u.pressed == nil {
		return false
	}
	active := u.pressed
	u.pressed = nil
	if !active.Enabled() {
		u.linkArmedSeg = -1
		return true
	}
	if tab, ok := active.(*widgets.TabBar); ok {
		return u.releaseTabBar(tab, event.Pos)
	}
	if list, ok := active.(*widgets.List); ok {
		return u.releaseList(list, event.Pos)
	}
	if rt, ok := active.(*widgets.RichText); ok {
		return u.releaseRichText(rt, event.Pos)
	}
	if active.HitTest(event.Pos) && u.hitSurface(event.Pos) == active {
		u.activateWidget(active)
	}
	return true
}

// releaseTabBar commits a released tab cell and closes the gesture.
// Releasing outside any cell consumes without activation. Committing the
// already-selected tab still fires OnClick but not OnTabSelect.
func (u *UI) releaseTabBar(bar *widgets.TabBar, pos core.Vec2) bool {
	index := u.tabIndexAt(bar, pos)
	if index < 0 {
		return true
	}
	u.commitTabSelection(bar, index)
	return true
}

// commitTabSelection records index on bar and fires selection callbacks.
// OnTabSelect runs only after a real index change; OnClick always runs.
func (u *UI) commitTabSelection(bar *widgets.TabBar, index int) {
	if bar.SetSelectedTab(index) {
		u.fireOnTabSelect(bar.Name(), index)
	}
	if u.Lookup(bar.Name()) == bar {
		u.fireOnClick(bar.Name())
	}
}

// tabIndexAt resolves the skin-aware tab cell under pos, or -1.
func (u *UI) tabIndexAt(bar *widgets.TabBar, pos core.Vec2) int {
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
	u.setHovered(nil)
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
func (u *UI) activateWidget(target widgets.Widget) bool {
	if !u.available(target) || !isPressable(target.Kind()) {
		return false
	}
	if cb, ok := target.(*widgets.Checkbox); ok {
		cb.SetChecked(!cb.Checked())
	}
	u.fireOnClick(target.Name())
	return true
}

// handleText edits the focused textbox and reports whether any caret,
// selection, clipboard, or text state changed. Text callbacks fire once with
// the final value after at least one real text mutation.
func (u *UI) handleText(event KeyEvent) bool {
	target, ok := u.focused.(*widgets.Textbox)
	if !ok || target == nil || !target.Enabled() {
		return false
	}
	handled := false
	mutated := false
	if event.SelectAll && target.SelectAll() {
		handled = true
	}
	clipboardHandled, clipboardTextMutated := u.handleTextboxClipboard(target, event)
	if clipboardHandled {
		handled = true
	}
	if clipboardTextMutated {
		mutated = true
	}
	if u.handleTextboxNavigation(target, event) {
		handled = true
	}
	if u.editText(target, event.Chars, event.Backspace, event.Delete) {
		mutated = true
		handled = true
	}
	if mutated {
		u.fireOnText(target.Name(), target.Text())
	}
	return handled
}

// handleTextboxClipboard applies copy, cut, and paste through the shared
// clipboard. It reports handling when an action ran and mutation when text
// changed (cut or paste insertion, including selection removal).
func (u *UI) handleTextboxClipboard(target *widgets.Textbox, event KeyEvent) (bool, bool) {
	handled := false
	mutated := false
	if event.Copy && target.HasSelection() {
		u.setClipboard(target.SelectedText())
		handled = true
	}
	if event.Cut && target.HasSelection() {
		u.setClipboard(target.SelectedText())
		if target.DeleteSelection() {
			mutated = true
		}
		handled = true
	}
	if event.Paste {
		if text := u.getClipboard(); text != "" && utf8.ValidString(text) {
			if target.InsertString(text) {
				mutated = true
				handled = true
			}
		}
	}
	return handled, mutated
}

// handleTextboxNavigation moves the textbox caret, extending the selection
// while Shift is held. It reports whether caret or selection changed.
func (u *UI) handleTextboxNavigation(target *widgets.Textbox, event KeyEvent) bool {
	handled := false
	extend := event.Shift
	if event.Left && target.MoveCaret(-1, extend) {
		handled = true
	}
	if event.Right && target.MoveCaret(1, extend) {
		handled = true
	}
	if event.Home && target.MoveCaretTo(0, extend) {
		handled = true
	}
	if event.End && target.MoveCaretTo(target.RuneCount(), extend) {
		handled = true
	}
	return handled
}

// editText applies printable runes plus backspace and delete, firing no
// callback itself; handleText and TypeText own the single OnText fire after
// at least one real mutation. It reports whether text changed.
func (u *UI) editText(target *widgets.Textbox, chars []rune, backspace, del bool) bool {
	mutated := false
	for _, char := range chars {
		if char >= 32 && char != 127 {
			mutated = target.TypeChar(char) || mutated
		}
	}
	if backspace {
		mutated = target.Backspace() || mutated
	}
	if del {
		mutated = target.Delete() || mutated
	}
	return mutated
}

// placeTextboxCaret moves the caret to the click X and clears any selection.
func (u *UI) placeTextboxCaret(target *widgets.Textbox, x float32) {
	if target == nil || u.theme == nil {
		return
	}
	index := u.theme.TextboxCaretIndex(target.Bounds(), core.StateFocused, target.Text(), x)
	target.SetCaret(index)
}

// setFocus switches focus, clearing any selection held by the previous
// textbox so stale highlights never resurface. It bubbles container focus
// so a field inside a frame keeps that frame glowing while text keeps
// editing rights; a focus outside every frame clears container focus.
func (u *UI) setFocus(target widgets.Widget) {
	if u == nil {
		return
	}
	if previous, ok := u.focused.(*widgets.Textbox); ok && previous != nil && previous != target {
		previous.ClearSelection()
	}
	u.focused = target
	u.bubbleActiveFrameFor(target)
}

// clearFocus releases focus, clearing any textbox selection first. It
// reports whether focus was held.
func (u *UI) clearFocus() bool {
	if u == nil || u.focused == nil {
		return false
	}
	if tb, ok := u.focused.(*widgets.Textbox); ok && tb != nil {
		tb.ClearSelection()
	}
	u.focused = nil
	return true
}

// hitInteractive returns the topmost enabled pressable widget under pos.
func (u *UI) hitInteractive(pos core.Vec2) widgets.Widget {
	w := u.hitSurface(pos)
	if u.available(w) && isPressable(w.Kind()) {
		return w
	}
	return nil
}

// isPressable reports the kinds that can own a UI press gesture.
func isPressable(kind core.WidgetKind) bool {
	switch kind {
	case core.WidgetButton, core.WidgetCheckbox, core.WidgetTextbox, core.WidgetSlider, core.WidgetDropdown, core.WidgetTabBar, core.WidgetRichText, core.WidgetList:
		return true
	default:
		return false
	}
}

func isFocusable(kind core.WidgetKind) bool {
	return kind == core.WidgetTextbox || kind == core.WidgetDropdown
}

// setSliderFromX converts logical X to a clamped value and reports only changes.
func (u *UI) setSliderFromX(widget *widgets.Slider, x float32) {
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
