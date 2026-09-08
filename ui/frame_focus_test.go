package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// TestFrameClickGainsAndLosesContainerFocus verifies click-to-focus semantics.
func TestFrameClickGainsAndLosesContainerFocus(t *testing.T) {
	u := New(400, 300)
	left := widgets.NewFrame("left", core.Rect{X: 10, Y: 10, W: 100, H: 100})
	right := widgets.NewFrame("right", core.Rect{X: 200, Y: 10, W: 100, H: 100})
	mustAdd(t, u, left, right)
	if u.ActiveFrame() != nil {
		t.Fatal("fresh UI must not hold container focus")
	}
	if !u.HandleMouse(MouseEvent{Pos: centerOf(left), Pressed: true}) || u.ActiveFrame() != left {
		t.Fatal("frame background press must gain container focus")
	}
	u.HandleMouse(MouseEvent{Pos: centerOf(left), Released: true})
	if u.ActiveFrame() != left {
		t.Fatal("release must retain container focus")
	}
	if !u.HandleMouse(MouseEvent{Pos: centerOf(right), Pressed: true}) || u.ActiveFrame() != right {
		t.Fatal("other-frame press must move container focus")
	}
	u.HandleMouse(MouseEvent{Pos: centerOf(right), Released: true})
	if u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 350, Y: 250}, Pressed: true}) || u.ActiveFrame() != nil {
		t.Fatal("empty press must clear container focus without consuming")
	}
	if !u.FocusFrame("left") || u.ActiveFrame() != left {
		t.Fatal("semantic frame focus failed")
	}
	if u.FocusFrame("missing") {
		t.Fatal("semantic frame focus accepted an unknown target")
	}
}

// TestFrameChildPressBubblesAndTextboxKeepsGlow verifies dual-slot coexistence.
func TestFrameChildPressBubblesAndTextboxKeepsGlow(t *testing.T) {
	u := New(400, 300)
	frame := widgets.NewFrame("demoFrame", core.Rect{X: 50, Y: 50, W: 200, H: 150})
	field := widgets.NewTextbox("field", core.Rect{X: 60, Y: 60, W: 100, H: 30}, 16)
	outside := widgets.NewButton("outside", core.Rect{X: 300, Y: 200, W: 80, H: 30}, "Out")
	mustAdd(t, u, frame, field, outside)
	mustParent(t, u, "field", "demoFrame")
	clickAt(u, centerOf(field))
	if u.Focused() != field || u.ActiveFrame() != frame {
		t.Fatal("textbox press must keep keyboard focus plus frame glow")
	}
	fires := 0
	u.OnHotkey("move", 'R', HotkeyOpts{Scope: "demoFrame", Consume: true}, func() { fires++ })
	if !u.HandleKey(KeyEvent{Chars: []rune("r"), Hotkeys: []rune("R")}) || field.Text() != "r" || fires != 0 {
		t.Fatalf("editing r must type without firing: text=%q fires=%d", field.Text(), fires)
	}
	if u.ActiveFrame() != frame {
		t.Fatal("typing must retain frame glow")
	}
	clickAt(u, centerOf(outside))
	if u.ActiveFrame() != nil {
		t.Fatal("press outside every frame must clear container focus")
	}
	if u.HandleKey(KeyEvent{Hotkeys: []rune("R")}) || fires != 0 {
		t.Fatal("out-of-scope R must pass to the game without firing")
	}
	if !u.FocusFrame("demoFrame") || u.Focused() != nil {
		t.Fatal("frame focus must clear keyboard focus")
	}
	if !u.HandleKey(KeyEvent{Hotkeys: []rune("r")}) || fires != 1 {
		t.Fatalf("in-scope R must fire and consume: fires=%d", fires)
	}
}

// TestFrameHotkeyFlags verifies scope, editing, and consume pass-through.
func TestFrameHotkeyFlags(t *testing.T) {
	u := New(400, 300)
	frame := widgets.NewFrame("demoFrame", core.Rect{X: 10, Y: 10, W: 200, H: 150})
	mustAdd(t, u, frame)
	shared := 0
	u.OnHotkey("shared", 'T', HotkeyOpts{Consume: false}, func() { shared++ })
	if u.HandleKey(KeyEvent{Hotkeys: []rune("T")}) || shared != 1 {
		t.Fatal("Consume:false must fire but pass through to the game")
	}
	if !u.RemoveHotkey("shared") || u.RemoveHotkey("shared") {
		t.Fatal("hotkey removal mismatch")
	}
	u.OnHotkey("gone", 'X', HotkeyOpts{Consume: true}, func() {})
	if !u.RemoveHotkey("gone") || u.HandleKey(KeyEvent{Hotkeys: []rune("X")}) {
		t.Fatal("removed hotkey must not consume")
	}
	blocked := 0
	u.OnHotkey("blocked", 'R', HotkeyOpts{Scope: "demoFrame", Consume: true}, func() { blocked++ })
	field := widgets.NewTextbox("field", core.Rect{X: 20, Y: 20, W: 100, H: 30}, 16)
	mustAdd(t, u, field)
	mustParent(t, u, "field", "demoFrame")
	u.Focus("field")
	if !u.HandleKey(KeyEvent{Hotkeys: []rune("R")}) || blocked != 0 {
		t.Fatal("disallowed-while-editing R must swallow without firing")
	}
	allowed := 0
	u.OnHotkey("allowed", 'C', HotkeyOpts{Scope: "demoFrame", AllowWhenEditing: true, Consume: true}, func() { allowed++ })
	if !u.HandleKey(KeyEvent{Hotkeys: []rune("C")}) || allowed != 1 {
		t.Fatal("allowed-while-editing hotkey must fire")
	}
	var nilUI *UI
	if nilUI.WantsTextInput() || nilUI.ActiveFrame() != nil {
		t.Fatal("nil UI focus queries must be empty")
	}
	nilUI.OnHotkey("x", 'X', HotkeyOpts{}, func() {})
	if nilUI.RemoveHotkey("x") || nilUI.FocusFrame("x") {
		t.Fatal("nil UI hotkey calls must not succeed")
	}
}

// TestFrameFullBufferTypingConsumes verifies intent over mutation for Chars.
func TestFrameFullBufferTypingConsumes(t *testing.T) {
	u := New(200, 100)
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 180, H: 30}, 1)
	mustAdd(t, u, field)
	calls := 0
	u.OnText("field", func(string) { calls++ })
	if !u.TypeText("field", "a") || calls != 1 {
		t.Fatal("seed edit failed")
	}
	if !u.HandleKey(KeyEvent{Chars: []rune("b")}) || field.Text() != "a" || calls != 1 {
		t.Fatalf("full-buffer typing must consume silently: text=%q calls=%d", field.Text(), calls)
	}
}

// TestFrameModalSwallowsIntent verifies menus capture keys while open.
func TestFrameModalSwallowsIntent(t *testing.T) {
	u := New(400, 300)
	frame := widgets.NewFrame("demoFrame", core.Rect{X: 10, Y: 10, W: 200, H: 150})
	mustAdd(t, u, frame)
	fires := 0
	u.OnHotkey("move", 'R', HotkeyOpts{Scope: "demoFrame", Consume: true}, func() { fires++ })
	u.FocusFrame("demoFrame")
	got := ""
	u.ShowContextMenu(menuTestItems(), core.Vec2{X: 100, Y: 100}, func(id string) { got = id })
	if u.ActiveFrame() != nil {
		t.Fatal("menu open must clear container focus")
	}
	if !u.HandleKey(KeyEvent{Chars: []rune("x")}) || got != "" {
		t.Fatal("menu-open typing must swallow without selecting")
	}
	if !u.HandleKey(KeyEvent{Hotkeys: []rune("R")}) || fires != 0 {
		t.Fatal("menu-open hotkey must swallow without firing")
	}
	if u.HandleKey(KeyEvent{}) {
		t.Fatal("empty poll while modal must not consume")
	}
	if !u.HandleKey(KeyEvent{Escape: true}) || u.HasOpenMenu() {
		t.Fatal("Escape must close the menu")
	}
}

// TestFrameEscapeDisableRemoveReconcile verifies focus loss paths.
func TestFrameEscapeDisableRemoveReconcile(t *testing.T) {
	u := New(400, 300)
	frame := widgets.NewFrame("demoFrame", core.Rect{X: 10, Y: 10, W: 200, H: 150})
	mustAdd(t, u, frame)
	if !u.FocusFrame("demoFrame") {
		t.Fatal("frame focus failed")
	}
	if !u.HandleKey(KeyEvent{Escape: true}) || u.ActiveFrame() != nil {
		t.Fatal("Escape must clear container focus when nothing else owns it")
	}
	if u.HandleKey(KeyEvent{Escape: true}) {
		t.Fatal("Escape with empty state must pass to the game")
	}
	if !u.FocusFrame("demoFrame") {
		t.Fatal("refocus failed")
	}
	frame.SetEnabled(false)
	u.HandleKey(KeyEvent{})
	if u.ActiveFrame() != nil {
		t.Fatal("disabled frame must release container focus")
	}
	frame.SetEnabled(true)
	if !u.FocusFrame("demoFrame") {
		t.Fatal("re-enable focus failed")
	}
	if !u.Remove("demoFrame") || u.ActiveFrame() != nil {
		t.Fatal("removal must clear container focus")
	}
	bad := widgets.NewFrame("bad", core.Rect{X: 10, Y: 10, W: 50, H: 50})
	bad.SetEnabled(false)
	mustAdd(t, u, bad)
	if u.FocusFrame("bad") {
		t.Fatal("disabled frame must refuse focus")
	}
	button := widgets.NewButton("btn", core.Rect{X: 10, Y: 10, W: 50, H: 20}, "B")
	mustAdd(t, u, button)
	if u.FocusFrame("btn") {
		t.Fatal("non-frame must refuse container focus")
	}
}

// TestFrameFocusedDrawsState verifies the glow snapshot for the active frame.
func TestFrameFocusedDrawsState(t *testing.T) {
	u := New(400, 300)
	frame := widgets.NewFrame("demoFrame", core.Rect{X: 10, Y: 10, W: 200, H: 150})
	other := widgets.NewFrame("other", core.Rect{X: 220, Y: 10, W: 100, H: 100})
	mustAdd(t, u, frame, other)
	if !u.FocusFrame("demoFrame") {
		t.Fatal("frame focus failed")
	}
	if state := u.visualState(frame); state != core.StateFocused {
		t.Fatalf("active frame state = %v", state)
	}
	if state := u.visualState(other); state != core.StateNormal {
		t.Fatalf("inactive frame state = %v", state)
	}
	recorder := attachDrawRecorder(t, u)
	u.Draw()
	focused := false
	for _, call := range recorder.Calls() {
		if call.Kind == core.WidgetFrame && call.State == core.StateFocused {
			focused = true
		}
	}
	if !focused {
		t.Fatal("active frame must draw a focused snapshot")
	}
}
