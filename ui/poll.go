package ui

import (
	"github.com/draxxris/rtgui/core"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// InputResult reports whether polled mouse or keyboard input was consumed by the UI.
type InputResult struct {
	MouseHandled bool
	KeyHandled   bool
}

// OnContextMenu registers a callback invoked when right-clicking outside open menus.
func (u *UI) OnContextMenu(fn func(pos core.Vec2)) {
	if u != nil {
		u.contextMenuHandler = fn
	}
}

// PollRaylibInput samples physical raylib pointer and keyboard state,
// routes the events through the UI, and updates the raylib mouse cursor.
// In headless environments where no raylib window is ready, it returns zero.
func (u *UI) PollRaylibInput() InputResult {
	if u == nil || !rl.IsWindowReady() {
		return InputResult{}
	}
	if !rl.IsWindowFocused() {
		u.CancelInput()
		return InputResult{}
	}
	mouseHandled := u.pollRaylibMouse()
	keyHandled := u.pollRaylibKeys()
	u.updateRaylibCursor()
	return InputResult{
		MouseHandled: mouseHandled,
		KeyHandled:   keyHandled,
	}
}

// pollRaylibMouse samples physical pointer coordinates and clicks into the UI.
func (u *UI) pollRaylibMouse() bool {
	pos := rl.GetMousePosition()
	mouse := u.ToLogical(core.Vec2{X: pos.X, Y: pos.Y})
	event := MouseEvent{
		Pos:          mouse,
		Pressed:      rl.IsMouseButtonPressed(rl.MouseButtonLeft),
		Down:         rl.IsMouseButtonDown(rl.MouseButtonLeft),
		Released:     rl.IsMouseButtonReleased(rl.MouseButtonLeft),
		RightPressed: rl.IsMouseButtonPressed(rl.MouseButtonRight),
		Wheel:        rl.GetMouseWheelMove(),
	}
	return u.HandleMouse(event)
}

// pollRaylibKeys samples typing, navigation, clipboard, and hotkey state into the UI.
func (u *UI) pollRaylibKeys() bool {
	u.charScratch = drainRaylibCharsInto(u.charScratch[:0])
	chars := u.charScratch
	event := KeyEvent{
		Chars:     chars,
		Backspace: raylibPressed(rl.KeyBackspace),
		Delete:    raylibPressed(rl.KeyDelete),
		Escape:    rl.IsKeyPressed(rl.KeyEscape),
		Hotkeys:   u.pollRaylibHotkeys(),
	}
	event.Left, event.Right, event.Home, event.End, event.Shift = raylibNavKeys()
	event.SelectAll, event.Copy, event.Cut, event.Paste = raylibClipboardKeys()
	if event.SelectAll || event.Copy || event.Cut || event.Paste {
		event.Chars = nil
	}
	return u.HandleKey(event)
}

// drainRaylibChars consumes pending printable runes from the raylib input queue.
func drainRaylibChars() []rune {
	return drainRaylibCharsInto(nil)
}

// drainRaylibCharsInto reuses caller storage for each polled character frame.
func drainRaylibCharsInto(out []rune) []rune {
	for cp := rl.GetCharPressed(); cp > 0; cp = rl.GetCharPressed() {
		if cp >= 32 && cp != 127 {
			out = append(out, rune(cp))
		}
	}
	return out
}

// raylibPressed reports whether key was pressed or repeated this frame.
func raylibPressed(key int32) bool {
	return rl.IsKeyPressed(key) || rl.IsKeyPressedRepeat(key)
}

// raylibNavKeys polls cursor navigation keys and shift modifiers.
func raylibNavKeys() (left, right, home, end, shift bool) {
	left = raylibPressed(rl.KeyLeft)
	right = raylibPressed(rl.KeyRight)
	home = raylibPressed(rl.KeyHome)
	end = raylibPressed(rl.KeyEnd)
	shift = rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	return left, right, home, end, shift
}

// raylibClipboardKeys polls Ctrl-A/C/X/V clipboard shortcuts.
func raylibClipboardKeys() (selectAll, copyKey, cutKey, pasteKey bool) {
	ctrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	if !ctrl {
		return false, false, false, false
	}
	return rl.IsKeyPressed(rl.KeyA), rl.IsKeyPressed(rl.KeyC), rl.IsKeyPressed(rl.KeyX), rl.IsKeyPressed(rl.KeyV)
}

// pollRaylibHotkeys checks active press edges for every registered UI hotkey.
func (u *UI) pollRaylibHotkeys() []rune {
	if len(u.hotkeys) == 0 {
		return nil
	}
	hotkeys := u.keyScratch[:0]
	for _, entry := range u.hotkeys {
		if (entry.key >= 'A' && entry.key <= 'Z') || (entry.key >= '0' && entry.key <= '9') {
			if rl.IsKeyPressed(int32(entry.key)) {
				hotkeys = append(hotkeys, entry.key)
			}
		}
	}
	u.keyScratch = hotkeys
	return hotkeys
}

// updateRaylibCursor adapts the mouse cursor icon to the currently hovered element.
func (u *UI) updateRaylibCursor() {
	if _, _, _, ok := u.HoveredLink(); ok {
		rl.SetMouseCursor(rl.MouseCursorPointingHand)
		return
	}
	if u.WantsTextInput() && u.hovered == u.focused {
		rl.SetMouseCursor(rl.MouseCursorIBeam)
		return
	}
	rl.SetMouseCursor(rl.MouseCursorDefault)
}
