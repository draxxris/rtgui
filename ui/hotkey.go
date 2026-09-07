package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// HotkeyOpts tunes one scoped hotkey registration. Scope names an active
// frame that must own container focus; empty means global. AllowWhenEditing
// permits firing while a textbox holds keyboard focus; otherwise text wins
// and the press is swallowed. Consume reports to the host game loop whether
// a fired hotkey owns the frame: true consumes, false fires but passes
// through so the game may also act.
type HotkeyOpts struct {
	Scope            string
	AllowWhenEditing bool
	Consume          bool
}

// hotkeyEntry is one library-owned scoped action. Keys are canonical
// upper-ASCII runes so Shift never changes identity. The slice stays tiny
// and registration-time only; per-frame matching allocates nothing.
type hotkeyEntry struct {
	id               string
	key              rune
	scope            string
	allowWhenEditing bool
	consume          bool
	fn               func()
}

// OnHotkey registers or replaces a scoped hotkey by id and runs its callback
// synchronously from HandleKey when scope and editing rules match. A nil
// callback is ignored and never removes an existing registration; use
// RemoveHotkey to delete an id. Keys match case-insensitively; callers pass
// press edges in KeyEvent.Hotkeys once per frame. Registration retains across
// widget removal like other callbacks; scope names that vanish simply stop
// matching until re-registered.
func (u *UI) OnHotkey(id string, key rune, opts HotkeyOpts, fn func()) {
	if u == nil || id == "" || key == 0 || fn == nil {
		return
	}
	canonical := upperASCII(key)
	for index := range u.hotkeys {
		if u.hotkeys[index].id != id {
			continue
		}
		u.hotkeys[index].key = canonical
		u.hotkeys[index].scope = opts.Scope
		u.hotkeys[index].allowWhenEditing = opts.AllowWhenEditing
		u.hotkeys[index].consume = opts.Consume
		u.hotkeys[index].fn = fn
		return
	}
	u.hotkeys = append(u.hotkeys, hotkeyEntry{
		id:               id,
		key:              canonical,
		scope:            opts.Scope,
		allowWhenEditing: opts.AllowWhenEditing,
		consume:          opts.Consume,
		fn:               fn,
	})
}

// RemoveHotkey deletes a hotkey registration by id and reports a removal.
func (u *UI) RemoveHotkey(id string) bool {
	if u == nil || id == "" {
		return false
	}
	for index := range u.hotkeys {
		if u.hotkeys[index].id != id {
			continue
		}
		copy(u.hotkeys[index:], u.hotkeys[index+1:])
		u.hotkeys = u.hotkeys[:len(u.hotkeys)-1]
		return true
	}
	return false
}

// ActiveFrame reports the container-focus owner, or nil when no frame holds
// it. Click a frame background or a child inside it to gain it; presses
// outside every frame move or lose it. Textbox focus coexists: the frame
// keeps its glow while typing, but text wins over frame hotkeys.
func (u *UI) ActiveFrame() widgets.Widget {
	if u == nil {
		return nil
	}
	return u.activeFrame
}

// FocusFrame gives container focus to an enabled frame by name without
// changing hover or firing a callback. It clears keyboard focus first so a
// held textbox selection never resurfaces under the newly active frame.
func (u *UI) FocusFrame(name string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	target := u.Lookup(name)
	if target == nil {
		u.diagnose("ui.FocusFrame: frame %q not found", name)
		return false
	}
	if !target.Enabled() {
		u.diagnose("ui.FocusFrame: frame %q is disabled", name)
		return false
	}
	if target.Kind() != core.WidgetFrame {
		u.diagnose("ui.FocusFrame: widget %q is %v, expected WidgetFrame", name, target.Kind())
		return false
	}
	u.clearFocus()
	u.setActiveFrame(target)
	return true
}

// WantsTextInput reports whether a focused enabled textbox should receive
// printable input. Hosts check it before polling bare-letter globals, and
// HandleKey uses it to prioritize text over scoped hotkeys.
func (u *UI) WantsTextInput() bool {
	if u == nil || u.focused == nil || !u.focused.Enabled() {
		return false
	}
	return u.focused.Kind() == core.WidgetTextbox
}

// setActiveFrame switches container focus to an enabled frame. A nil target
// clears. Callers resolve bubbling first; this helper only stores.
func (u *UI) setActiveFrame(target widgets.Widget) {
	if u == nil {
		return
	}
	if target == nil {
		u.activeFrame = nil
		return
	}
	if target.Kind() != core.WidgetFrame || !target.Enabled() {
		return
	}
	u.activeFrame = target
}

// clearActiveFrame releases container focus and reports whether one was held.
func (u *UI) clearActiveFrame() bool {
	if u == nil || u.activeFrame == nil {
		return false
	}
	u.activeFrame = nil
	return true
}

// innermostFrameAt returns the topmost enabled frame containing pos, or nil.
// Reverse registry order matches draw stacking so overlapping panels resolve
// to the visible one. The scan allocates nothing and stays linear in the
// widget count.
func (u *UI) innermostFrameAt(pos core.Vec2) widgets.Widget {
	if u == nil {
		return nil
	}
	for index := len(u.order) - 1; index >= 0; index-- {
		widget := u.widgets[u.order[index]]
		if widget == nil || !widget.Enabled() || widget.Kind() != core.WidgetFrame {
			continue
		}
		if widget.HitTest(pos) {
			return widget
		}
	}
	return nil
}

// bubbleActiveFrameFor keeps container focus coherent with keyboard focus.
// A focused widget inside a frame keeps that frame active for glow; a focus
// outside every frame clears container focus so scoped hotkeys stop firing.
func (u *UI) bubbleActiveFrameFor(target widgets.Widget) {
	if u == nil || target == nil {
		return
	}
	bounds := target.Bounds()
	center := core.Vec2{X: bounds.X + bounds.W/2, Y: bounds.Y + bounds.H/2}
	if frame := u.innermostFrameAt(center); frame != nil {
		u.setActiveFrame(frame)
		return
	}
	u.clearActiveFrame()
}

// hasPrintableChars reports whether any rune would survive the shared text
// filter. Full-buffer typing still counts as intent so the host game never
// observes a character the user meant for the field.
func hasPrintableChars(chars []rune) bool {
	for _, char := range chars {
		if char >= 32 && char != 127 {
			return true
		}
	}
	return false
}

// upperASCII canonicalizes hotkey identity so Shift never forks R from r.
// Non-ASCII runes pass through unchanged; hotkeys are ASCII by contract.
func upperASCII(key rune) rune {
	if key >= 'a' && key <= 'z' {
		return key - ('a' - 'A')
	}
	return key
}

// hotkeyPressed reports whether canonical key appears in the frame presses.
// Comparison is upper-ASCII on both sides and allocates nothing.
func hotkeyPressed(key rune, presses []rune) bool {
	for _, press := range presses {
		if upperASCII(press) == key {
			return true
		}
	}
	return false
}

// fireScopedHotkeys matches frame presses against the registry in order and
// fires at most one callback per frame. It reports consumption only:
// disallowed-while-editing presses swallow without firing so the game never
// double-handles a suppressed bare letter, while Consume:false fires but
// passes through for shared globals.
func (u *UI) fireScopedHotkeys(presses []rune) bool {
	if u == nil || len(presses) == 0 || len(u.hotkeys) == 0 {
		return false
	}
	editing := u.WantsTextInput()
	activeName := ""
	if u.activeFrame != nil {
		activeName = u.activeFrame.Name()
	}
	for index := range u.hotkeys {
		entry := &u.hotkeys[index]
		if !hotkeyPressed(entry.key, presses) {
			continue
		}
		if entry.scope != "" && activeName != entry.scope {
			continue
		}
		if editing && !entry.allowWhenEditing {
			return true
		}
		if entry.fn != nil {
			entry.fn()
		}
		return entry.consume
	}
	return false
}
