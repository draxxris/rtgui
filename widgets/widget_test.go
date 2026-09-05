package widgets_test

import (
	"reflect"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/widgets"
)

// TestWidgetIdentityKindAndConfigurationAreEncapsulated checks immutable metadata and private fields.
func TestWidgetIdentityKindAndConfigurationAreEncapsulated(t *testing.T) {
	button := widgets.NewButton("okButton", core.Rect{W: 100, H: 30}, "OK")
	if button.Name() != "okButton" || button.Kind() != core.WidgetButton {
		t.Fatalf("identity = %q/%v", button.Name(), button.Kind())
	}
	widgetType := reflect.TypeOf(*button)
	for index := 0; index < widgetType.NumField(); index++ {
		if field := widgetType.Field(index); field.PkgPath == "" {
			t.Fatalf("Widget field %q is publicly mutable", field.Name)
		}
	}
	button.SetBounds(core.Rect{X: 4, Y: 5, W: 6, H: 7})
	button.SetEnabled(false)
	button.SetText("Changed")
	if button.Name() != "okButton" || button.Kind() != core.WidgetButton {
		t.Fatal("configuration mutation changed immutable identity or kind")
	}
	if snapshot := button.Snapshot(core.StateFocused); snapshot.Name != "okButton" || snapshot.Kind != core.WidgetButton || snapshot.State != core.StateFocused {
		t.Fatalf("Snapshot = %+v", snapshot)
	}
}

// TestWidgetDomainAccessors checks focused mutable domain operations.
func TestWidgetDomainAccessors(t *testing.T) {
	checkbox := widgets.NewCheckbox("agree", core.Rect{}, false)
	if !checkbox.SetChecked(true) || !checkbox.Checked() || checkbox.SetChecked(true) {
		t.Fatal("checkbox mutator must report only real changes")
	}

	slider := widgets.NewSlider("volume", core.Rect{}, 2)
	if slider.Value() != 1 || !slider.SetValue(-1) || slider.Value() != 0 {
		t.Fatalf("slider clamping failed: %v", slider.Value())
	}
	progress := widgets.NewProgressBar("progress", core.Rect{}, 0.25)
	if !progress.SetValue(0.75) || progress.Value() != 0.75 {
		t.Fatalf("progress value = %v", progress.Value())
	}

	panel := widgets.NewScrollPanel("scroll", core.Rect{})
	if !panel.ScrollBy(10, 20) || panel.Scroll() != (core.Vec2{X: 10, Y: 20}) {
		t.Fatalf("scroll = %+v", panel.Scroll())
	}
	if !panel.SetScroll(core.Vec2{X: 2, Y: 3}) || panel.Scroll() != (core.Vec2{X: 2, Y: 3}) {
		t.Fatalf("set scroll = %+v", panel.Scroll())
	}
}

// TestTextboxEditingDoesNotExposeBuffer checks bounded edits through widget methods.
func TestTextboxEditingDoesNotExposeBuffer(t *testing.T) {
	textbox := widgets.NewTextbox("field", core.Rect{}, 7)
	if !textbox.SetText("aé") || textbox.Text() != "aé" {
		t.Fatalf("textbox text = %q", textbox.Text())
	}
	if !textbox.TypeChar('😀') || textbox.Text() != "aé😀" {
		t.Fatalf("textbox append = %q", textbox.Text())
	}
	if textbox.TypeChar('x') {
		t.Fatal("full textbox append must be rejected")
	}
	if !textbox.Backspace() || textbox.Text() != "aé" {
		t.Fatalf("textbox backspace = %q", textbox.Text())
	}
}

// TestDropdownCopiesInputAndOutput protects dropdown slice ownership.
func TestDropdownCopiesInputAndOutput(t *testing.T) {
	items := []string{"Warrior", "Ranger", "Mage"}
	dropdown := widgets.NewDropdown("class", core.Rect{X: 10, Y: 20, W: 180, H: 42}, items, 0)
	items[0] = "mutated input"
	if selected, ok := dropdown.DropdownSelection(); !ok || selected != "Warrior" {
		t.Fatalf("selection = %q/%v", selected, ok)
	}
	snapshot := dropdown.DropdownItems()
	snapshot[1] = "mutated output"
	if got := dropdown.DropdownItems()[1]; got != "Ranger" {
		t.Fatalf("output mutation changed widget item to %q", got)
	}
	if !dropdown.SetDropdownIndex(2) || dropdown.DropdownIndex() != 2 {
		t.Fatalf("selection index = %d", dropdown.DropdownIndex())
	}
	if dropdown.SetDropdownIndex(9) || dropdown.DropdownIndex() != 2 {
		t.Fatal("invalid selection changed dropdown")
	}
}

// TestDropdownPopupGeometry checks library-owned popup row hit testing.
func TestDropdownPopupGeometry(t *testing.T) {
	dropdown := widgets.NewDropdown("class", core.Rect{X: 10, Y: 20, W: 180, H: 42}, []string{"Warrior", "Ranger", "Mage"}, 0)
	if popup := dropdown.DropdownPopupBounds(); popup != (core.Rect{X: 10, Y: 66, W: 180, H: 108}) {
		t.Fatalf("popup bounds = %+v", popup)
	}
	row, ok := dropdown.DropdownRowBounds(1)
	if !ok || row != (core.Rect{X: 10, Y: 102, W: 180, H: 36}) {
		t.Fatalf("second row = %+v/%v", row, ok)
	}
	if got := dropdown.DropdownIndexAt(core.Vec2{X: 20, Y: 110}); got != 1 {
		t.Fatalf("row index = %d", got)
	}
}

// TestWidgetUsesResolvedFrameBounds verifies hit testing, snapshots, and popup
// geometry consume the widget's arranged node directly.
func TestWidgetUsesResolvedFrameBounds(t *testing.T) {
	dropdown := arrangedDropdown(t)
	if dropdown.Frame() == nil {
		t.Fatal("visual widget has no layout frame")
	}
	want := core.Rect{X: 100, Y: 80, W: 180, H: 42}
	if dropdown.Bounds() != want || dropdown.Snapshot(core.StateNormal).Bounds != want {
		t.Fatalf("resolved widget bounds = %+v", dropdown.Bounds())
	}
	if !dropdown.HitTest(core.Vec2{X: 110, Y: 90}) || dropdown.HitTest(core.Vec2{X: 10, Y: 10}) {
		t.Fatal("hit testing did not use resolved bounds")
	}
	if popup := dropdown.DropdownPopupBounds(); popup.X != 100 || popup.Y != 126 {
		t.Fatalf("resolved popup bounds = %+v", popup)
	}
}

// arrangedDropdown creates one widget resolved through a parent layout tree.
func arrangedDropdown(t *testing.T) *widgets.Widget {
	t.Helper()
	root := layout.New("root", core.Rect{W: 400, H: 300})
	dropdown := widgets.NewDropdown("class", core.Rect{W: 180, H: 42}, []string{"A", "B"}, 0)
	if err := root.AddChild(dropdown.Frame()); err != nil {
		t.Fatal(err)
	}
	if err := dropdown.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: 100, Y: 80}); err != nil {
		t.Fatal(err)
	}
	if err := layout.ArrangeRoot(root, root.AuthoredBounds()); err != nil {
		t.Fatal(err)
	}
	return dropdown
}

// TestAbsoluteWidgetRetainsConstructorBounds verifies layout remains optional.
func TestAbsoluteWidgetRetainsConstructorBounds(t *testing.T) {
	bounds := core.Rect{X: 7, Y: 9, W: 80, H: 24}
	button := widgets.NewButton("absolute", bounds, "Absolute")
	if button.Bounds() != bounds || !button.HitTest(core.Vec2{X: 10, Y: 10}) {
		t.Fatalf("absolute bounds = %+v", button.Bounds())
	}
	updated := core.Rect{X: 20, Y: 30, W: 90, H: 28}
	button.SetBounds(updated)
	if button.Bounds() != updated || button.Snapshot(core.StateNormal).Bounds != updated {
		t.Fatalf("updated absolute bounds = %+v", button.Bounds())
	}
}
