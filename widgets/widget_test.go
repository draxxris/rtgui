package widgets

import (
	"testing"

	"rtgui/core"
	"rtgui/input"
)

func TestWidgetInteraction(t *testing.T) {
	capture := input.NewCapture()
	button := NewButton("okButton", core.Rect{W: 100, H: 30}, "OK")
	button.UpdateHover(core.Vec2{X: 50, Y: 15})
	if button.State != core.StateHovered {
		t.Fatalf("hover state %v", button.State)
	}
	if !button.Press(core.Vec2{X: 50, Y: 15}, capture) || !capture.IsCaptured() {
		t.Fatal("button press should capture")
	}
	if !button.Release(core.Vec2{X: 50, Y: 15}, capture) || capture.IsCaptured() {
		t.Fatal("button release should click and release capture")
	}
	button.Enabled = false
	button.UpdateHover(core.Vec2{X: 50, Y: 15})
	if button.State != core.StateDisabled || button.Press(core.Vec2{X: 50, Y: 15}, capture) {
		t.Fatal("disabled button should not press")
	}
}

func TestWidgetKindsAndEditing(t *testing.T) {
	checkbox := NewCheckbox("agreeCheckbox", core.Rect{W: 20, H: 20}, false)
	capture := input.NewCapture()
	checkbox.Press(core.Vec2{X: 10, Y: 10}, capture)
	checkbox.Release(core.Vec2{X: 10, Y: 10}, capture)
	if !checkbox.Checked {
		t.Fatal("checkbox should toggle")
	}

	textbox := NewTextbox("nameField", core.Rect{W: 200, H: 30}, 64)
	textbox.Focus()
	for _, ch := range []rune{'a', 'é', '😀', '中'} {
		textbox.TypeChar(ch)
	}
	if textbox.TextBuf.String() != "aé😀中" {
		t.Fatalf("textbox value %q", textbox.TextBuf.String())
	}
	textbox.Backspace()
	if textbox.TextBuf.String() != "aé😀" {
		t.Fatalf("unicode backspace %q", textbox.TextBuf.String())
	}

	slider := NewSlider("volumeSlider", core.Rect{W: 100, H: 10}, 0.5)
	slider.SetSlider(2)
	if slider.Value != 1 {
		t.Fatalf("slider upper clamp %v", slider.Value)
	}
	slider.SetSlider(-1)
	if slider.Value != 0 {
		t.Fatalf("slider lower clamp %v", slider.Value)
	}

	panel := NewScrollPanel("scrollPanel", core.Rect{W: 100, H: 100})
	panel.ScrollBy(10, 20)
	if panel.Scroll != (core.Vec2{X: 10, Y: 20}) {
		t.Fatalf("scroll offset %v", panel.Scroll)
	}
}

// TestWidgetStringIDs verifies the string-external numeric-internal identity.
// Every constructor stores Name, derives a deterministic nonzero ID, and
// propagates Name through Info for the renderer path.
func TestWidgetStringIDs(t *testing.T) {
	first := NewButton("okButton", core.Rect{W: 120, H: 40}, "OK")
	again := NewButton("okButton", core.Rect{W: 120, H: 40}, "OK")
	other := NewButton("cancelButton", core.Rect{W: 120, H: 40}, "Cancel")
	if first.Name != "okButton" {
		t.Fatalf("Name = %q, want okButton", first.Name)
	}
	if first.ID == 0 {
		t.Fatal("internal ID must never be 0")
	}
	if first.ID != again.ID {
		t.Fatalf("same name gave different IDs %d vs %d", first.ID, again.ID)
	}
	if first.ID == other.ID {
		t.Fatalf("distinct names collided on ID %d", first.ID)
	}
	if info := first.Info(); info.Name != "okButton" || info.ID != first.ID {
		t.Fatalf("Info() = %+v, want Name okButton ID %d", info, first.ID)
	}
	all := []*Widget{
		NewRectangle("rectID", core.Rect{W: 10, H: 10}),
		NewLabel("labelID", core.Rect{W: 10, H: 10}, "hi"),
		NewCheckbox("checkID", core.Rect{W: 20, H: 20}, false),
		NewTextbox("textID", core.Rect{W: 100, H: 30}, 64),
		NewSlider("sliderID", core.Rect{W: 100, H: 10}, 0.5),
		NewProgressBar("progressID", core.Rect{W: 100, H: 10}, 0.5),
		NewScrollPanel("scrollID", core.Rect{W: 100, H: 100}),
		NewDropdown("dropID", core.Rect{W: 100, H: 30}, []string{"a"}, 0),
		NewFrame("frameID", core.Rect{W: 100, H: 100}),
	}
	for _, w := range all {
		if w.Name == "" {
			t.Fatal("constructor left Name empty")
		}
		if w.ID == 0 {
			t.Fatalf("widget %q has zero internal ID", w.Name)
		}
		if w.Info().Name != w.Name {
			t.Fatalf("Info().Name = %q, want %q", w.Info().Name, w.Name)
		}
	}
}
