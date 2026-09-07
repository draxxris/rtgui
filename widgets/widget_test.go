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

// TestDropdownPopupGeometry checks popup outer bounds and item counts.
// Row geometry is skin-aware and owned by render (DropdownPopupContent
// with DropdownPopupRow/Index), so this package asserts only the
// skin-free outer bounds and count used by those call sites.
func TestDropdownPopupGeometry(t *testing.T) {
	dropdown := widgets.NewDropdown("class", core.Rect{X: 10, Y: 20, W: 180, H: 42}, []string{"Warrior", "Ranger", "Mage"}, 0)
	if popup := dropdown.DropdownPopupBounds(); popup != (core.Rect{X: 10, Y: 66, W: 180, H: 108}) {
		t.Fatalf("popup bounds = %+v", popup)
	}
	if got := dropdown.DropdownItemCount(); got != 3 {
		t.Fatalf("item count = %d", got)
	}
	if got := widgets.NewDropdown("empty", core.Rect{}, nil, 0).DropdownItemCount(); got != 0 {
		t.Fatalf("empty item count = %d", got)
	}
	var nilWidget *widgets.Dropdown
	if nilWidget.DropdownItemCount() != 0 {
		t.Fatal("nil widget item count must be 0")
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
func arrangedDropdown(t *testing.T) *widgets.Dropdown {
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

// TestWidgetDirectCallbacksAndCanvas tests direct event handlers and custom canvas.
// TestWidgetDirectCallbacks verifies direct callback registration on button, slider, textbox, and tab bar.
func TestWidgetDirectCallbacks(t *testing.T) {
	btn := widgets.NewButton("btn", core.Rect{}, "OK")
	clicked := false
	btn.OnClick(func() { clicked = true })
	if btn.OnClickHandler() == nil {
		t.Fatal("expected OnClickHandler to be non-nil")
	}
	btn.OnClickHandler()()
	if !clicked {
		t.Fatal("expected direct click callback to execute")
	}

	slider := widgets.NewSlider("s", core.Rect{}, 0.5)
	var sliderVal float32
	slider.OnChange(func(v float32) { sliderVal = v })
	slider.OnChangeHandler()(0.8)
	if sliderVal != 0.8 {
		t.Fatalf("expected slider callback 0.8, got %v", sliderVal)
	}
	slider.SetFormat("%.1f")
	if slider.Format() != "%.1f" {
		t.Fatalf("expected format %q, got %q", "%.1f", slider.Format())
	}

	tb := widgets.NewTextbox("tb", core.Rect{}, 16)
	var textVal string
	tb.OnText(func(s string) { textVal = s })
	tb.OnTextHandler()("hello")
	if textVal != "hello" {
		t.Fatalf("expected text callback hello, got %q", textVal)
	}

	tab := widgets.NewTabBar("tabs", core.Rect{}, []string{"A", "B"}, 0)
	var tabIdx int
	tab.OnTabSelect(func(i int) { tabIdx = i })
	tab.OnTabSelectHandler()(1)
	if tabIdx != 1 {
		t.Fatalf("expected tab callback 1, got %d", tabIdx)
	}

	btn.SetTooltip("Button tip")
	if btn.Tooltip() != "Button tip" {
		t.Fatalf("expected tooltip %q, got %q", "Button tip", btn.Tooltip())
	}
}

// TestCanvasWidget verifies creation and invocation of canvas drawing functions.
func TestCanvasWidget(t *testing.T) {
	drawn := false
	canvas := widgets.NewCanvas("c", core.Rect{W: 50, H: 50}, func(b core.Rect) { drawn = true })
	if canvas.Kind() != core.WidgetCanvas || canvas.CanvasDraw() == nil {
		t.Fatal("expected WidgetCanvas with non-nil CanvasDraw")
	}
	canvas.CanvasDraw()(canvas.Bounds())
	if !drawn {
		t.Fatal("expected canvas draw function to run")
	}
}

// TestCheckboxWithLabelAndTextColor verifies label text, checked state, and custom text color.
func TestCheckboxWithLabelAndTextColor(t *testing.T) {
	cb := widgets.NewCheckboxWithLabel("cb", core.Rect{}, "Remember me", true)
	if cb.Text() != "Remember me" || !cb.Checked() {
		t.Fatalf("expected checkbox with label and checked=true")
	}
	cb.SetTextColor(core.Color{R: 255, G: 0, B: 0, A: 255})
	if c, ok := cb.TextColor(); !ok || c.R != 255 {
		t.Fatalf("expected text color configured")
	}
	snapshot := cb.Snapshot(core.StateNormal)
	if !snapshot.HasTextColor || snapshot.TextColor.R != 255 {
		t.Fatalf("expected snapshot to carry text color")
	}
}

// TestScrollPanelClampingAndContentDrawer tests max scroll limits and content drawers.
func TestScrollPanelClampingAndContentDrawer(t *testing.T) {
	panel := widgets.NewScrollPanel("scroll", core.Rect{W: 100, H: 100})
	panel.SetMaxScroll(core.Vec2{X: 0, Y: 150})
	if panel.MaxScroll() != (core.Vec2{X: 0, Y: 150}) {
		t.Fatalf("expected max scroll {0, 150}, got %+v", panel.MaxScroll())
	}
	panel.ScrollBy(0, 200)
	if panel.Scroll().Y != 150 {
		t.Fatalf("expected scroll Y clamped to 150, got %v", panel.Scroll().Y)
	}
	panel.ScrollBy(0, -300)
	if panel.Scroll().Y != 0 {
		t.Fatalf("expected scroll Y clamped to 0, got %v", panel.Scroll().Y)
	}

	drawerCalled := false
	panel.SetScrollContentDrawer(func(bounds core.Rect, offset core.Vec2) {
		drawerCalled = true
	})
	if panel.ScrollContentDrawer() == nil {
		t.Fatal("expected non-nil scroll content drawer")
	}
	panel.ScrollContentDrawer()(panel.Bounds(), panel.Scroll())
	if !drawerCalled {
		t.Fatal("expected scroll content drawer to run")
	}
}

// TestStyledLabelTypographyAndAlignment verifies typography options and alignment propagation.
func TestStyledLabelTypographyAndAlignment(t *testing.T) {
	label := widgets.NewStyledLabel("title", core.Rect{W: 200, H: 40}, "Gallery Title", 24, true, core.AlignCenter)
	if label.Kind() != core.WidgetLabel {
		t.Fatalf("expected WidgetLabel, got %v", label.Kind())
	}
	if label.FontSize() != 24 || !label.Italic() || label.Align() != core.AlignCenter {
		t.Fatalf("unexpected label properties: size=%v italic=%v align=%v", label.FontSize(), label.Italic(), label.Align())
	}

	snap := label.Snapshot(core.StateNormal)
	if snap.FontSize != 24 || !snap.Italic || snap.Align != core.AlignCenter {
		t.Fatalf("unexpected snapshot typography: %+v", snap)
	}

	label.SetFontSize(16).SetItalic(false).SetAlign(core.AlignRight)
	snap2 := label.Snapshot(core.StateNormal)
	if snap2.FontSize != 16 || snap2.Italic || snap2.Align != core.AlignRight {
		t.Fatalf("unexpected updated snapshot: %+v", snap2)
	}
}
