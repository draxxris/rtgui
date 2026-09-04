package ui

import (
	"testing"

	"rtgui/core"
	"rtgui/widgets"
)

// newSampleUI builds the Target Sample widget set for integration tests.
func newSampleUI() (*UI, *widgets.Widget, *widgets.Widget, *widgets.Widget, *widgets.Widget, *widgets.Widget) {
	u := New(800, 600)
	button := widgets.NewButton("primaryButton", core.Rect{X: 40, Y: 40, W: 200, H: 42}, "Primary")
	checkbox := widgets.NewCheckbox("enableBox", core.Rect{X: 40, Y: 100, W: 200, H: 38}, true)
	textbox := widgets.NewTextbox("inputBox", core.Rect{X: 40, Y: 160, W: 300, H: 42}, 128)
	slider := widgets.NewSlider("valueSlider", core.Rect{X: 40, Y: 220, W: 300, H: 42}, 0.35)
	progress := widgets.NewProgressBar("valueProgress", core.Rect{X: 40, Y: 280, W: 300, H: 24}, 0.35)
	u.Add(button, checkbox, textbox, slider, progress)
	return u, button, checkbox, textbox, slider, progress
}

// centerOf returns the center point of a widget's bounds in logical coords.
func centerOf(w *widgets.Widget) core.Vec2 {
	return core.Vec2{X: w.Bounds.X + w.Bounds.W/2, Y: w.Bounds.Y + w.Bounds.H/2}
}

// clickAt drives a full press/release cycle at pos and reports handling.
func clickAt(u *UI, pos core.Vec2) (pressed, released bool) {
	pressed = u.HandleMouse(MouseEvent{Pos: pos, Pressed: true})
	released = u.HandleMouse(MouseEvent{Pos: pos, Released: true})
	return pressed, released
}

// TestAddPolicy verifies nil/empty are ignored and duplicates overwrite order.
func TestAddPolicy(t *testing.T) {
	u := New(800, 600)
	a := widgets.NewButton("a", core.Rect{X: 0, Y: 0, W: 50, H: 20}, "A")
	b := widgets.NewButton("b", core.Rect{X: 60, Y: 0, W: 50, H: 20}, "B")
	u.Add(nil, widgets.NewButton("", core.Rect{W: 10, H: 10}, "empty"), a, b)
	if len(u.order) != 2 {
		t.Fatalf("expected 2 widgets, got %d", len(u.order))
	}
	shadow := widgets.NewButton("a", core.Rect{X: 5, Y: 5, W: 10, H: 10}, "shadow")
	u.Add(shadow)
	if len(u.order) != 2 || u.order[0] != "a" {
		t.Fatalf("duplicate must overwrite without changing order: %v", u.order)
	}
	if u.Lookup("a") != shadow {
		t.Fatal("duplicate Add must replace the entry")
	}
}

// TestHoverAloneNeverConsumes verifies mouse movement only updates state.
func TestHoverAloneNeverConsumes(t *testing.T) {
	u, button, _, _, _, _ := newSampleUI()
	if u.HandleMouse(MouseEvent{Pos: centerOf(button)}) {
		t.Fatal("hover-only must return false for game pass-through")
	}
	if button.State != core.StateHovered {
		t.Fatalf("hover should set Hovered, got %d", button.State)
	}
}

// TestClickFiresOnClick verifies press/release consumes and fires once.
func TestClickFiresOnClick(t *testing.T) {
	u, button, _, _, _, _ := newSampleUI()
	calls := 0
	u.OnClick("primaryButton", func() { calls++ })
	pressed, released := clickAt(u, centerOf(button))
	if !pressed || !released {
		t.Fatalf("click must be handled: pressed=%v released=%v", pressed, released)
	}
	if calls != 1 {
		t.Fatalf("expected 1 OnClick, got %d", calls)
	}
	if u.Capture().IsCaptured() {
		t.Fatal("capture must be released after click")
	}
}

// TestCheckboxToggleVisible verifies the callback sees post-toggle state.
func TestCheckboxToggleVisible(t *testing.T) {
	u, _, checkbox, _, _, _ := newSampleUI()
	if !checkbox.Checked {
		t.Fatal("sample checkbox must start checked")
	}
	var seen bool
	seenSet := false
	u.OnClick("enableBox", func() { seen, seenSet = checkbox.Checked, true })
	clickAt(u, centerOf(checkbox))
	if !seenSet {
		t.Fatal("OnClick must fire")
	}
	if seen {
		t.Fatal("checkbox must toggle before OnClick fires (started checked, must read false)")
	}
	if checkbox.Checked {
		t.Fatal("checkbox must be unchecked after click")
	}
}

// TestEmptySpaceBlursAndPassesThrough verifies misses blur but return false.
func TestEmptySpaceBlursAndPassesThrough(t *testing.T) {
	u, _, _, textbox, _, _ := newSampleUI()
	clickAt(u, centerOf(textbox))
	if u.Focused() != textbox {
		t.Fatal("textbox press must focus")
	}
	if u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 790, Y: 590}, Pressed: true}) {
		t.Fatal("empty-space press must return false so the game gets the click")
	}
	if u.Focused() != nil {
		t.Fatal("empty-space press must blur focus")
	}
}

// TestDisabledPassesThrough verifies disabled widgets never consume or fire.
func TestDisabledPassesThrough(t *testing.T) {
	u, button, _, _, _, _ := newSampleUI()
	button.Enabled = false
	fired := false
	u.OnClick("primaryButton", func() { fired = true })
	pressed, released := clickAt(u, centerOf(button))
	if pressed || released {
		t.Fatal("disabled widget must not consume press/release")
	}
	if fired {
		t.Fatal("disabled widget must not fire OnClick")
	}
}

// TestSliderDragFiresOnChange verifies press/drag/release value flow.
func TestSliderDragFiresOnChange(t *testing.T) {
	u, _, _, _, slider, _ := newSampleUI()
	var got []float32
	u.OnChange("valueSlider", func(v float32) { got = append(got, v) })
	left := core.Vec2{X: slider.Bounds.X + 1, Y: slider.Bounds.Y + slider.Bounds.H/2}
	if !u.HandleMouse(MouseEvent{Pos: left, Pressed: true}) {
		t.Fatal("slider press must be handled")
	}
	mid := core.Vec2{X: slider.Bounds.X + slider.Bounds.W/2, Y: left.Y}
	if !u.HandleMouse(MouseEvent{Pos: mid, Down: true}) {
		t.Fatal("slider drag must be handled")
	}
	if slider.Value < 0.49 || slider.Value > 0.51 {
		t.Fatalf("mid drag should give ~0.5, got %v", slider.Value)
	}
	if len(got) == 0 {
		t.Fatal("OnChange must fire on drag")
	}
	if !u.HandleMouse(MouseEvent{Pos: mid, Released: true}) {
		t.Fatal("slider release must be handled (capture ownership)")
	}
}

// TestReleaseOffWidgetConsumesButNoClick verifies drag-off consumes silently.
func TestReleaseOffWidgetConsumesButNoClick(t *testing.T) {
	u, button, _, _, _, _ := newSampleUI()
	fired := false
	u.OnClick("primaryButton", func() { fired = true })
	if !u.HandleMouse(MouseEvent{Pos: centerOf(button), Pressed: true}) {
		t.Fatal("press must be handled")
	}
	if !u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 790, Y: 590}, Released: true}) {
		t.Fatal("release after press must consume even off-widget")
	}
	if fired {
		t.Fatal("off-widget release must not fire OnClick")
	}
}

// TestWheelRouting verifies wheel is handled only over a scroll panel.
func TestWheelRouting(t *testing.T) {
	u := New(800, 600)
	scroll := widgets.NewScrollPanel("scroll", core.Rect{X: 100, Y: 100, W: 200, H: 200})
	u.Add(scroll)
	over := MouseEvent{Pos: core.Vec2{X: 150, Y: 150}, Wheel: 1}
	if !u.HandleMouse(over) {
		t.Fatal("wheel over scroll panel must be handled")
	}
	if scroll.Scroll.Y == 0 {
		t.Fatal("wheel must move ScrollBy")
	}
	off := MouseEvent{Pos: core.Vec2{X: 10, Y: 10}, Wheel: 1}
	if u.HandleMouse(off) {
		t.Fatal("wheel off-widget must return false for camera zoom")
	}
}

// TestOnTextUTF8 verifies focus-first typing with multi-byte runes.
func TestOnTextUTF8(t *testing.T) {
	u, _, _, textbox, _, _ := newSampleUI()
	var got string
	u.OnText("inputBox", func(s string) { got = s })
	clickAt(u, centerOf(textbox))
	if !u.HandleKey(KeyEvent{Chars: []rune("aé😀中")}) {
		t.Fatal("typing into focused textbox must be handled")
	}
	if textbox.TextBuf.String() != "aé😀中" {
		t.Fatalf("UTF-8 buffer mismatch: %q", textbox.TextBuf.String())
	}
	if got != "aé😀中" {
		t.Fatalf("OnText must receive full string, got %q", got)
	}
	if !u.HandleKey(KeyEvent{Backspace: true}) {
		t.Fatal("backspace must be handled")
	}
	if textbox.TextBuf.String() != "aé😀" {
		t.Fatalf("backspace must drop one rune, got %q", textbox.TextBuf.String())
	}
}

// TestUnfocusedKeysPassThrough verifies game hotkeys survive without focus.
func TestUnfocusedKeysPassThrough(t *testing.T) {
	u, _, _, _, _, _ := newSampleUI()
	if u.HandleKey(KeyEvent{Chars: []rune("wasd")}) {
		t.Fatal("keys with no focus must return false for game hotkeys")
	}
}

// TestEscapeBlursOnlyWhenEffective verifies Escape consumption rules.
func TestEscapeBlursOnlyWhenEffective(t *testing.T) {
	u, _, _, textbox, _, _ := newSampleUI()
	if u.HandleKey(KeyEvent{Escape: true}) {
		t.Fatal("Escape with no focus must pass through")
	}
	clickAt(u, centerOf(textbox))
	if !u.HandleKey(KeyEvent{Escape: true}) {
		t.Fatal("Escape that blurs must be handled")
	}
	if u.Focused() != nil {
		t.Fatal("Escape must clear focus")
	}
}

// TestUnknownAndNilCallbacks verifies storage and removal semantics.
func TestUnknownAndNilCallbacks(t *testing.T) {
	u, button, _, _, _, _ := newSampleUI()
	fired := false
	u.OnClick("future", func() { fired = true })
	clickAt(u, centerOf(button))
	if fired {
		t.Fatal("unknown-name callback must not fire spuriously")
	}
	u.OnClick("primaryButton", func() { fired = true })
	u.OnClick("primaryButton", nil)
	clickAt(u, centerOf(button))
	if fired {
		t.Fatal("nil must remove the registration")
	}
	late := widgets.NewButton("future", core.Rect{X: 400, Y: 400, W: 100, H: 30}, "F")
	u.Add(late)
	clickAt(u, centerOf(late))
	if !fired {
		t.Fatal("callback stored before Add must fire once the widget arrives")
	}
}

// TestDrawLogsNames verifies headless Draw output carries string identity.
func TestDrawLogsNames(t *testing.T) {
	u, _, _, _, _, _ := newSampleUI()
	u.Draw()
	log := u.Theme().DrawLog()
	if len(log) == 0 {
		t.Fatal("Draw must log calls headlessly")
	}
	last := u.Theme().LastWidgetInfo()
	if last.Name == "" {
		t.Fatal("LastWidgetInfo must carry the string name")
	}
}

// TestIntegrationSample replicates the Target Sample callback wiring.
func TestIntegrationSample(t *testing.T) {
	u, button, checkbox, textbox, slider, progress := newSampleUI()
	status := "click something"
	u.OnClick("primaryButton", func() { status = "Primary clicked" })
	u.OnClick("enableBox", func() {
		button.Enabled = checkbox.Checked
		if button.Enabled {
			status = "button enabled=true"
		} else {
			status = "button enabled=false"
		}
	})
	u.OnText("inputBox", func(s string) { status = "typed: " + s })
	u.OnChange("valueSlider", func(v float32) {
		progress.Value = v
		status = "slider moved"
	})
	clickAt(u, centerOf(button))
	if status != "Primary clicked" {
		t.Fatalf("button click status: %q", status)
	}
	clickAt(u, centerOf(checkbox))
	if button.Enabled {
		t.Fatal("unchecking must disable the button")
	}
	clickAt(u, centerOf(textbox))
	if !u.HandleKey(KeyEvent{Chars: []rune("hi")}) {
		t.Fatal("typing must be handled")
	}
	if status != "typed: hi" {
		t.Fatalf("text status: %q", status)
	}
	mid := core.Vec2{X: slider.Bounds.X + slider.Bounds.W, Y: slider.Bounds.Y + 5}
	u.HandleMouse(MouseEvent{Pos: mid, Pressed: true})
	u.HandleMouse(MouseEvent{Pos: mid, Released: true})
	if progress.Value < 0.99 {
		t.Fatalf("progress must sync via OnChange, got %v", progress.Value)
	}
	u.Draw()
	if len(u.Theme().DrawLog()) == 0 {
		t.Fatal("integration must produce draw calls")
	}
}

// TestResizeAndToLogical verifies the design resolution stays fixed while the
// physical window rescales the mapping: a maximized window scales components
// instead of reflowing them.
func TestResizeAndToLogical(t *testing.T) {
	u := New(800, 600)
	u.Add(widgets.NewButton("b", core.Rect{X: 10, Y: 10, W: 100, H: 30}, "B"))
	u.Resize(1600, 1200)
	sx, sy := u.Scale()
	if sx != 2 || sy != 2 {
		t.Fatalf("2x window must give 2x scale, got %v,%v", sx, sy)
	}
	got := u.ToLogical(core.Vec2{X: 100, Y: 50})
	if got.X != 50 || got.Y != 25 {
		t.Fatalf("physical must map into fixed logical: %+v", got)
	}
	// Widget bounds follow the design resolution, never the window.
	if b := u.Lookup("b"); b == nil || b.Bounds.W != 100 {
		t.Fatal("Resize must not reflow widget bounds")
	}
	before := len(u.Theme().DrawLog())
	u.Resize(-1, 0)
	u.Draw()
	if len(u.Theme().DrawLog()) == before {
		t.Fatal("invalid resize must not break subsequent Draw")
	}
}

// TestNilSafety verifies nil receivers never panic and never consume.
func TestNilSafety(t *testing.T) {
	var u *UI
	u.Add(widgets.NewButton("x", core.Rect{W: 10, H: 10}, "X"))
	u.Resize(100, 100)
	if u.HandleMouse(MouseEvent{Pressed: true}) {
		t.Fatal("nil UI must not consume mouse")
	}
	if u.HandleKey(KeyEvent{Chars: []rune("a")}) {
		t.Fatal("nil UI must not consume keys")
	}
	u.OnClick("x", func() {})
	u.OnChange("x", func(float32) {})
	u.OnText("x", func(string) {})
	u.Draw()
	if u.Theme() != nil || u.Capture() != nil || u.Focused() != nil || u.Lookup("x") != nil {
		t.Fatal("nil accessors must return nil")
	}
}

// TestDropdownPressFiresOnClick verifies dropdown presses are consumed and fire.
func TestDropdownPressFiresOnClick(t *testing.T) {
	u := New(800, 600)
	d := widgets.NewDropdown("dd", core.Rect{X: 10, Y: 10, W: 200, H: 42}, []string{"A", "B"}, 0)
	u.Add(d)
	fired := false
	u.OnClick("dd", func() { fired = true })
	pressed, released := clickAt(u, centerOf(d))
	if !pressed || !released {
		t.Fatalf("dropdown click must be handled: pressed=%v released=%v", pressed, released)
	}
	if !fired {
		t.Fatal("dropdown OnClick must fire to open the popup")
	}
}

// TestPassiveWidgetsIgnoreHover verifies panels, labels, frames, and the
// rectangle primitive never highlight or consume pointer input.
func TestPassiveWidgetsIgnoreHover(t *testing.T) {
	u := New(800, 600)
	rect := widgets.NewFrame("demoPanel", core.Rect{X: 40, Y: 40, W: 200, H: 94})
	label := widgets.NewLabel("demoLabel", core.Rect{X: 40, Y: 150, W: 200, H: 32}, "Hi")
	u.Add(rect, label)
	if u.HandleMouse(MouseEvent{Pos: centerOf(rect)}) {
		t.Fatal("hover over passive widgets must not consume")
	}
	if rect.State != core.StateNormal || label.State != core.StateNormal {
		t.Fatalf("passive widgets must stay Normal, got %v/%v", rect.State, label.State)
	}
	pressed, released := clickAt(u, centerOf(rect))
	if pressed || released {
		t.Fatal("clicks on passive widgets must pass through to the game")
	}
	if rect.State != core.StateNormal {
		t.Fatalf("passive widgets must stay Normal after click, got %v", rect.State)
	}
}
