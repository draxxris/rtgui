package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// TestTextboxArrowsMoveCaret verifies arrow, Home, and End positioning.
func TestTextboxArrowsMoveCaret(t *testing.T) {
	u := New(200, 100)
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 180, H: 30}, 16)
	mustAdd(t, u, field)
	field.SetText("abcd")
	u.Focus("field")
	if field.Caret() != 4 {
		t.Fatalf("initial caret = %d", field.Caret())
	}
	if !u.HandleKey(KeyEvent{Left: true}) || field.Caret() != 3 {
		t.Fatalf("left caret = %d", field.Caret())
	}
	if !u.HandleKey(KeyEvent{Home: true}) || field.Caret() != 0 {
		t.Fatalf("home caret = %d", field.Caret())
	}
	if !u.HandleKey(KeyEvent{Left: true}) {
		t.Fatal("left at start must consume editing intent")
	}
	if !u.HandleKey(KeyEvent{End: true}) || field.Caret() != 4 {
		t.Fatalf("end caret = %d", field.Caret())
	}
	if !u.HandleKey(KeyEvent{Right: true}) {
		t.Fatal("right at end must consume editing intent")
	}
	if !u.HandleKey(KeyEvent{Left: true, Shift: true}) || !field.HasSelection() {
		t.Fatal("shift-left must extend selection")
	}
	if start, end := field.Selection(); start != 3 || end != 4 {
		t.Fatalf("shift selection = %d/%d", start, end)
	}
}

// TestTextboxDeleteRemovesForward verifies DELETE key behavior.
func TestTextboxDeleteRemovesForward(t *testing.T) {
	u := New(200, 100)
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 180, H: 30}, 16)
	mustAdd(t, u, field)
	field.SetText("abcd")
	u.Focus("field")
	u.HandleKey(KeyEvent{Home: true})
	calls := 0
	u.OnText("field", func(string) { calls++ })
	if !u.HandleKey(KeyEvent{Delete: true}) || field.Text() != "bcd" || calls != 1 {
		t.Fatalf("delete forward = %q calls=%d", field.Text(), calls)
	}
	// DELETE with selection removes the range.
	field.SelectAll()
	if !u.HandleKey(KeyEvent{Delete: true}) || field.Text() != "" {
		t.Fatalf("delete selection = %q", field.Text())
	}
	// DELETE at end still belongs to the focused editor.
	if !u.HandleKey(KeyEvent{Delete: true}) {
		t.Fatal("delete at end must consume editing intent")
	}
}

// TestTextboxSelectAllAndClipboard verifies Ctrl-A/C/X/V flows.
func TestTextboxSelectAllAndClipboard(t *testing.T) {
	u := New(200, 100)
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 180, H: 30}, 16)
	mustAdd(t, u, field)
	field.SetText("hello")
	u.Focus("field")
	if !u.HandleKey(KeyEvent{SelectAll: true}) || !field.HasSelection() {
		t.Fatal("select-all failed")
	}
	if !u.HandleKey(KeyEvent{SelectAll: true}) {
		t.Fatal("redundant select-all must consume editing intent")
	}
	if !u.HandleKey(KeyEvent{Copy: true}) {
		t.Fatal("copy failed")
	}
	if got := u.ClipboardText(); got != "hello" {
		t.Fatalf("clipboard = %q", got)
	}
	// Cut removes and copies.
	textCalls := 0
	u.OnText("field", func(string) { textCalls++ })
	if !u.HandleKey(KeyEvent{Cut: true}) || field.Text() != "" || textCalls != 1 {
		t.Fatalf("cut = %q calls=%d", field.Text(), textCalls)
	}
	// Paste restores.
	if !u.HandleKey(KeyEvent{Paste: true}) || field.Text() != "hello" || textCalls != 2 {
		t.Fatalf("paste = %q calls=%d", field.Text(), textCalls)
	}
	// Copy without selection consumes intent without changing the clipboard.
	field.ClearSelection()
	u.SetClipboardText("held")
	if !u.HandleKey(KeyEvent{Copy: true}) {
		t.Fatal("copy without selection must consume editing intent")
	}
	if got := u.ClipboardText(); got != "held" {
		t.Fatalf("clipboard clobbered = %q", got)
	}
	// Paste with empty clipboard consumes intent without changing text.
	u.SetClipboardText("")
	if !u.HandleKey(KeyEvent{Paste: true}) {
		t.Fatal("paste of empty clipboard must consume editing intent")
	}
}

// TestTextboxTypingReplacesSelection verifies printable runes overwrite.
func TestTextboxTypingReplacesSelection(t *testing.T) {
	u := New(200, 100)
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 180, H: 30}, 16)
	mustAdd(t, u, field)
	field.SetText("abcd")
	u.Focus("field")
	field.SelectAll()
	calls := 0
	u.OnText("field", func(string) { calls++ })
	if !u.HandleKey(KeyEvent{Chars: []rune("X")}) || field.Text() != "X" || calls != 1 {
		t.Fatalf("type over selection = %q calls=%d", field.Text(), calls)
	}
	// Backspace with selection removes the range.
	field.SetText("abcd")
	field.SelectAll()
	if !u.HandleKey(KeyEvent{Backspace: true}) || field.Text() != "" {
		t.Fatalf("backspace selection = %q", field.Text())
	}
}

// TestTextboxClickPlacesCaret verifies mouse positioning and focus cleanup.
func TestTextboxClickPlacesCaret(t *testing.T) {
	u := New(200, 100)
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 180, H: 30}, 16)
	mustAdd(t, u, field)
	field.SetText("abcd")
	// Click near the left edge lands at the start.
	left := core.Vec2{X: 16, Y: 25}
	clickAt(u, left)
	if u.Focused() != field {
		t.Fatal("click did not focus")
	}
	if field.Caret() != 0 {
		t.Fatalf("left click caret = %d", field.Caret())
	}
	// Click far right lands at the end.
	right := core.Vec2{X: 180, Y: 25}
	clickAt(u, right)
	if field.Caret() != 4 {
		t.Fatalf("right click caret = %d", field.Caret())
	}
	// Empty-space press clears focus and selection without consuming.
	field.SelectAll()
	if u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 195, Y: 90}, Pressed: true}) {
		t.Fatal("empty press must not consume")
	}
	if u.Focused() != nil || field.HasSelection() {
		t.Fatal("empty press must clear focus and selection")
	}
}

// TestTextboxDrawsCaretAndSelection verifies recorder parts and skin insets.
func TestTextboxDrawsCaretAndSelection(t *testing.T) {
	u := New(200, 100)
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 180, H: 30}, 16)
	mustAdd(t, u, field)
	field.SetText("abcd")
	u.Focus("field")
	recorder := attachDrawRecorder(t, u)
	u.Draw()
	if !hasKindPart(recorder.Calls(), core.WidgetTextbox, skin.PartCaret) {
		t.Fatal("focused textbox must draw a caret")
	}
	if hasKindPart(recorder.Calls(), core.WidgetTextbox, skin.PartOverlay) {
		t.Fatal("idle textbox must not draw selection")
	}
	field.SelectAll()
	selected := attachDrawRecorder(t, u)
	u.Draw()
	if !hasKindPart(selected.Calls(), core.WidgetTextbox, skin.PartOverlay) {
		t.Fatal("selected textbox must draw a highlight")
	}
	// Unfocused textboxes hide the caret.
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 195, Y: 90}, Pressed: true})
	blurred := attachDrawRecorder(t, u)
	u.Draw()
	if hasKindPart(blurred.Calls(), core.WidgetTextbox, skin.PartCaret) {
		t.Fatal("unfocused textbox must not draw a caret")
	}
}

// TestTextboxSkinUsesBackgroundBorderAndPadding verifies CSS LOOK support.
func TestTextboxSkinUsesBackgroundBorderAndPadding(t *testing.T) {
	u := New(200, 100)
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 180, H: 30}, 16)
	mustAdd(t, u, field)
	theme := u.Theme()
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTextbox, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, G: 255, B: 255, A: 255}, HasTexture: true,
		PaddingLeft: 4, PaddingTop: 3, PaddingRight: 5, PaddingBottom: 6,
	})
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTextbox, Part: skin.PartBorder, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, G: 255, B: 255, A: 255}, HasTexture: true,
		HasNinePatch: true, NinePatch: skin.NinePatch{Left: 8, Top: 8, Right: 8, Bottom: 8},
	})
	content := theme.TextboxContent(field.Bounds(), core.StateNormal)
	// Left inset is max(padding 4, nine-patch 8) = 8; top is max(3, 8) = 8.
	if content.X != field.Bounds().X+8 || content.Y != field.Bounds().Y+8 {
		t.Fatalf("textbox content = %+v", content)
	}
	u.Focus("field")
	recorder := attachDrawRecorder(t, u)
	u.Draw()
	calls := recorder.Calls()
	if !hasKindPart(calls, core.WidgetTextbox, skin.PartBackground) {
		t.Fatal("textbox must draw background-image")
	}
	if !hasKindPart(calls, core.WidgetTextbox, skin.PartBorder) {
		t.Fatal("textbox must draw border-image")
	}
	if !hasKindPart(calls, core.WidgetTextbox, skin.PartCaret) {
		t.Fatal("skinned textbox must still draw a caret")
	}
	var nilUI *UI
	if nilUI.ClipboardText() != "" {
		t.Fatal("nil UI clipboard must be empty")
	}
	nilUI.SetClipboardText("x")
}

// TestTextboxClipboardRoundTrips verifies the in-memory fallback headless.
func TestTextboxClipboardRoundTrips(t *testing.T) {
	u := New(100, 100)
	u.SetClipboardText("held text")
	if got := u.ClipboardText(); got != "held text" {
		t.Fatalf("clipboard = %q", got)
	}
	// Pasting invalid UTF-8 never mutates.
	field := widgets.NewTextbox("field", core.Rect{X: 10, Y: 10, W: 80, H: 20}, 16)
	mustAdd(t, u, field)
	u.Focus("field")
	u.SetClipboardText(string([]byte{0xff}))
	if !u.HandleKey(KeyEvent{Paste: true}) || field.Text() != "" {
		t.Fatal("invalid clipboard paste must consume without mutation")
	}
}
