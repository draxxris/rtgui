package ui

import (
	"errors"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

func mustAdd(t *testing.T, u *UI, list ...*widgets.Widget) {
	t.Helper()
	if err := u.Add(list...); err != nil {
		t.Fatalf("Add: %v", err)
	}
}

func centerOf(widget *widgets.Widget) core.Vec2 {
	bounds := widget.Bounds()
	return core.Vec2{X: bounds.X + bounds.W/2, Y: bounds.Y + bounds.H/2}
}

func clickAt(u *UI, pos core.Vec2) (bool, bool) {
	return u.HandleMouse(MouseEvent{Pos: pos, Pressed: true}), u.HandleMouse(MouseEvent{Pos: pos, Released: true})
}

// attachDrawRecorder adds bounded diagnostics to a test UI.
func attachDrawRecorder(t *testing.T, u *UI) *render.DrawRecorder {
	t.Helper()
	recorder, err := render.NewDrawRecorder(128)
	if err != nil {
		t.Fatalf("NewDrawRecorder: %v", err)
	}
	u.Theme().SetDrawRecorder(recorder)
	return recorder
}

// TestAddValidatesAtomicallyAndRejectsDuplicates checks the registry contract.
func TestAddValidatesAtomicallyAndRejectsDuplicates(t *testing.T) {
	u := New(800, 600)
	first := widgets.NewButton("first", core.Rect{W: 20, H: 20}, "first")
	if err := u.Add(first); err != nil {
		t.Fatal(err)
	}
	if err := u.Add(widgets.NewButton("first", core.Rect{}, "duplicate")); !errors.Is(err, ErrDuplicateWidget) {
		t.Fatalf("duplicate Add = %v", err)
	}
	valid := widgets.NewButton("valid", core.Rect{}, "valid")
	if err := u.Add(valid, nil); !errors.Is(err, ErrNilWidget) {
		t.Fatalf("invalid variadic Add = %v", err)
	}
	if u.Lookup("valid") != nil || len(u.order) != 1 {
		t.Fatal("invalid variadic Add partially mutated the registry")
	}
	if err := u.Add(widgets.NewButton("", core.Rect{}, "empty")); !errors.Is(err, ErrEmptyWidgetName) {
		t.Fatalf("empty-name Add = %v", err)
	}
	if err := u.Add(widgets.NewButton("same", core.Rect{}, "a"), widgets.NewButton("same", core.Rect{}, "b")); !errors.Is(err, ErrDuplicateWidget) {
		t.Fatalf("same-request duplicate Add = %v", err)
	}
	var nilUI *UI
	if err := nilUI.Add(first); !errors.Is(err, ErrNilUI) {
		t.Fatalf("nil UI Add = %v", err)
	}
}

// TestRemoveAndAddMovesWidgetToTop checks explicit replacement ordering.
func TestRemoveAndAddMovesWidgetToTop(t *testing.T) {
	u := New(100, 100)
	first := widgets.NewButton("first", core.Rect{W: 50, H: 50}, "first")
	second := widgets.NewButton("second", core.Rect{W: 50, H: 50}, "second")
	mustAdd(t, u, first, second)
	if !u.Remove("first") || u.Remove("missing") {
		t.Fatal("Remove result mismatch")
	}
	replacement := widgets.NewButton("first", core.Rect{W: 50, H: 50}, "replacement")
	mustAdd(t, u, replacement)
	if len(u.order) != 2 || u.order[0] != "second" || u.order[1] != "first" {
		t.Fatalf("remove-and-add order = %v", u.order)
	}
}

// TestTopmostOnlyHoverAndActivation checks overlap ownership.
func TestTopmostOnlyHoverAndActivation(t *testing.T) {
	u := New(100, 100)
	bottom := widgets.NewButton("bottom", core.Rect{W: 50, H: 50}, "bottom")
	top := widgets.NewButton("top", core.Rect{W: 50, H: 50}, "top")
	mustAdd(t, u, bottom, top)
	bottomCalls, topCalls := 0, 0
	u.OnClick("bottom", func() { bottomCalls++ })
	u.OnClick("top", func() { topCalls++ })
	if u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 10, Y: 10}}) {
		t.Fatal("hover alone must not consume input")
	}
	if u.Hovered() != top || u.visualState(top) != core.StateHovered || u.visualState(bottom) != core.StateNormal {
		t.Fatal("only the topmost eligible widget may hover")
	}
	pressed, released := clickAt(u, core.Vec2{X: 10, Y: 10})
	if !pressed || !released || topCalls != 1 || bottomCalls != 0 {
		t.Fatalf("topmost click = %v/%v calls=%d/%d", pressed, released, topCalls, bottomCalls)
	}
}

// TestFocusedDisableAndReenableUsesFreshState checks disabled reconciliation.
func TestFocusedDisableAndReenableUsesFreshState(t *testing.T) {
	u := New(100, 100)
	field := widgets.NewTextbox("field", core.Rect{W: 50, H: 20}, 16)
	mustAdd(t, u, field)
	if !u.Focus("field") || u.Focused() != field || u.visualState(field) != core.StateFocused {
		t.Fatal("semantic focus failed")
	}
	field.SetEnabled(false)
	u.Draw()
	if u.Focused() != nil || u.visualState(field) != core.StateDisabled {
		t.Fatal("draw did not reconcile disabled focus")
	}
	field.SetEnabled(true)
	u.Draw()
	if u.visualState(field) != core.StateNormal {
		t.Fatalf("re-enabled state = %v", u.visualState(field))
	}
}

// TestPressedDisableCancelsBeforeRelease checks press persistence and cancellation.
func TestPressedDisableCancelsBeforeRelease(t *testing.T) {
	u := New(100, 100)
	button := widgets.NewButton("button", core.Rect{W: 50, H: 20}, "button")
	other := widgets.NewButton("other", core.Rect{X: 60, W: 30, H: 20}, "other")
	mustAdd(t, u, button, other)
	calls := 0
	u.OnClick("button", func() { calls++ })
	if !u.HandleMouse(MouseEvent{Pos: centerOf(button), Pressed: true}) || u.Pressed() != button {
		t.Fatal("press did not establish owner")
	}
	if !u.HandleMouse(MouseEvent{Pos: centerOf(other), Pressed: true}) || u.Pressed() != button {
		t.Fatal("a second press replaced the active owner before release")
	}
	button.SetEnabled(false)
	if u.HandleMouse(MouseEvent{Pos: centerOf(button), Released: true}) {
		t.Fatal("release after disable must not consume a cancelled press")
	}
	if u.Pressed() != nil || calls != 0 {
		t.Fatal("disabled pressed widget remained active or fired")
	}
}

// TestVisualStatePriority checks disabled, pressed, focused, and hovered order.
func TestVisualStatePriority(t *testing.T) {
	u := New(100, 100)
	widget := widgets.NewTextbox("field", core.Rect{W: 20, H: 20}, 8)
	mustAdd(t, u, widget)
	u.hovered, u.focused, u.pressed = widget, widget, widget
	if state := u.visualState(widget); state != core.StatePressed {
		t.Fatalf("pressed priority = %v", state)
	}
	u.pressed = nil
	if state := u.visualState(widget); state != core.StateFocused {
		t.Fatalf("focused priority = %v", state)
	}
	u.focused = nil
	if state := u.visualState(widget); state != core.StateHovered {
		t.Fatalf("hovered priority = %v", state)
	}
	widget.SetEnabled(false)
	if state := u.visualState(widget); state != core.StateDisabled {
		t.Fatalf("disabled priority = %v", state)
	}
}

// TestRemovalClearsOwnersAndRetainsCallbacks checks independent lifetimes.
func TestRemovalClearsOwnersAndRetainsCallbacks(t *testing.T) {
	u := New(100, 100)
	button := widgets.NewButton("button", core.Rect{W: 50, H: 20}, "button")
	mustAdd(t, u, button)
	calls := 0
	u.OnClick("button", func() { calls++ })
	u.HandleMouse(MouseEvent{Pos: centerOf(button), Pressed: true})
	if u.Hovered() != button || u.Pressed() != button {
		t.Fatal("test did not establish hover and press")
	}
	if !u.Remove("button") || u.Hovered() != nil || u.Pressed() != nil || u.Focused() != nil {
		t.Fatal("Remove retained an active widget")
	}
	replacement := widgets.NewButton("button", core.Rect{W: 50, H: 20}, "replacement")
	mustAdd(t, u, replacement)
	clickAt(u, centerOf(replacement))
	if calls != 1 {
		t.Fatal("callback did not survive widget removal")
	}
	u.ClearWidgets()
	if u.Lookup("button") != nil || len(u.order) != 0 || u.Hovered() != nil || u.Pressed() != nil || u.Focused() != nil {
		t.Fatal("ClearWidgets retained widgets or owners")
	}
}

// TestReleaseOutsideConsumesWithoutActivation checks gesture completion.
func TestReleaseOutsideConsumesWithoutActivation(t *testing.T) {
	u := New(100, 100)
	button := widgets.NewButton("button", core.Rect{W: 20, H: 20}, "button")
	mustAdd(t, u, button)
	calls := 0
	u.OnClick("button", func() { calls++ })
	if !u.HandleMouse(MouseEvent{Pos: centerOf(button), Pressed: true}) {
		t.Fatal("press must be handled")
	}
	if !u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 90, Y: 90}, Released: true}) || calls != 0 {
		t.Fatal("outside release must consume without activation")
	}
}

// TestCheckboxMutatesBeforeSharedCallback checks callback ordering on both paths.
func TestCheckboxMutatesBeforeSharedCallback(t *testing.T) {
	u := New(100, 100)
	checkbox := widgets.NewCheckbox("check", core.Rect{W: 20, H: 20}, false)
	mustAdd(t, u, checkbox)
	seen := false
	u.OnClick("check", func() { seen = checkbox.Checked() })
	if !u.Activate("check") || !seen || !checkbox.Checked() {
		t.Fatal("semantic checkbox callback did not observe post-toggle value")
	}
	seen = true
	clickAt(u, centerOf(checkbox))
	if seen || checkbox.Checked() {
		t.Fatal("physical checkbox callback did not share post-toggle semantics")
	}
}

// TestSliderCallbacksOnlyAfterValueChanges checks change filtering.
func TestSliderCallbacksOnlyAfterValueChanges(t *testing.T) {
	u := New(100, 100)
	slider := widgets.NewSlider("slider", core.Rect{W: 100, H: 20}, 0.5)
	mustAdd(t, u, slider)
	var values []float32
	u.OnChange("slider", func(value float32) { values = append(values, value) })
	middle := core.Vec2{X: 50, Y: 10}
	u.HandleMouse(MouseEvent{Pos: middle, Pressed: true})
	u.HandleMouse(MouseEvent{Pos: middle, Down: true})
	if len(values) != 0 {
		t.Fatalf("unchanged slider fired %v", values)
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 75, Y: 10}, Down: true})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 75, Y: 10}, Down: true})
	if len(values) != 1 || values[0] != 0.75 || slider.Value() != 0.75 {
		t.Fatalf("changed slider callbacks=%v value=%v", values, slider.Value())
	}
}

// TestTextCallbacksStaySilentForRejectedEdits checks full and empty edits.
func TestTextCallbacksStaySilentForRejectedEdits(t *testing.T) {
	u := New(100, 100)
	field := widgets.NewTextbox("field", core.Rect{W: 50, H: 20}, 1)
	mustAdd(t, u, field)
	calls := 0
	u.OnText("field", func(string) { calls++ })
	if !u.TypeText("field", "a") || calls != 1 {
		t.Fatal("semantic fitting edit failed")
	}
	if !u.TypeText("field", "b") || calls != 1 || field.Text() != "a" {
		t.Fatal("full semantic edit must be handled without callback")
	}
	if !u.HandleKey(KeyEvent{Backspace: true}) || calls != 2 {
		t.Fatal("non-empty physical backspace failed")
	}
	if u.HandleKey(KeyEvent{Backspace: true}) || calls != 2 {
		t.Fatal("empty backspace fired callback or consumed")
	}
	if !u.TypeText("field", "") || calls != 2 || u.Focused() != field {
		t.Fatal("empty semantic typing must focus without callback")
	}
}

// TestSemanticOperationsRejectInvalidTargetsWithoutMutation checks validation.
func TestSemanticOperationsRejectInvalidTargetsWithoutMutation(t *testing.T) {
	u := New(100, 100)
	button := widgets.NewButton("button", core.Rect{}, "button")
	field := widgets.NewTextbox("field", core.Rect{}, 8)
	disabled := widgets.NewTextbox("disabled", core.Rect{}, 8)
	disabled.SetEnabled(false)
	mustAdd(t, u, button, field, disabled)
	if u.TypeText("button", "x") || u.TypeText("missing", "x") || u.TypeText("disabled", "x") || u.TypeText("field", string([]byte{0xff})) {
		t.Fatal("TypeText accepted a wrong, unknown, disabled, or invalid UTF-8 target/value")
	}
	if u.Focus("button") || u.Focus("missing") || u.Focus("disabled") {
		t.Fatal("Focus accepted a wrong, unknown, or disabled target")
	}
	if u.Activate("missing") || u.Activate("disabled") {
		t.Fatal("Activate accepted an unknown or disabled target")
	}
	if field.Text() != "" || u.Focused() != nil {
		t.Fatal("invalid semantics mutated UI")
	}
}

// TestSemanticActivationLeavesHoverUnchanged checks pointer-free control.
func TestSemanticActivationLeavesHoverUnchanged(t *testing.T) {
	u := New(100, 100)
	hovered := widgets.NewButton("hovered", core.Rect{W: 20, H: 20}, "hovered")
	activated := widgets.NewButton("activated", core.Rect{X: 40, W: 20, H: 20}, "activated")
	mustAdd(t, u, hovered, activated)
	u.HandleMouse(MouseEvent{Pos: centerOf(hovered)})
	if !u.Activate("activated") || u.Hovered() != hovered {
		t.Fatal("semantic activation changed hover ownership")
	}
}

// TestTwoUIsHaveNoSharedInteractionOrCallbacks checks instance isolation.
func TestTwoUIsHaveNoSharedInteractionOrCallbacks(t *testing.T) {
	firstUI, secondUI := New(100, 100), New(100, 100)
	first := widgets.NewCheckbox("same", core.Rect{W: 20, H: 20}, false)
	second := widgets.NewCheckbox("same", core.Rect{W: 20, H: 20}, false)
	mustAdd(t, firstUI, first)
	mustAdd(t, secondUI, second)
	calls := 0
	firstUI.OnClick("same", func() { calls++ })
	firstUI.HandleMouse(MouseEvent{Pos: centerOf(first)})
	if !firstUI.Activate("same") || !first.Checked() || second.Checked() || calls != 1 {
		t.Fatal("UI instances shared domain data or callbacks")
	}
	if secondUI.Hovered() != nil || firstUI.Hovered() != first {
		t.Fatal("UI instances shared hover ownership")
	}
}

// TestDropdownPopupSelectionAndRenderingRemainLibraryOwned protects the popup path.
func TestDropdownPopupSelectionAndRenderingRemainLibraryOwned(t *testing.T) {
	u := New(200, 200)
	dropdown := widgets.NewDropdown("class", core.Rect{X: 10, Y: 10, W: 100, H: 30}, []string{"A", "B"}, 0)
	mustAdd(t, u, dropdown)
	clickAt(u, centerOf(dropdown))
	if u.Focused() != dropdown {
		t.Fatal("dropdown click did not open popup")
	}
	content := u.Theme().DropdownPopupContent(dropdown.DropdownPopupBounds(), core.StatePressed)
	row, _ := render.DropdownPopupRow(content, dropdown.DropdownItemCount(), 1)
	point := core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
	if u.HandleMouse(MouseEvent{Pos: point}) {
		t.Fatal("popup hover alone must not consume")
	}
	recorder := attachDrawRecorder(t, u)
	u.Draw()
	if !hasDrawCall(recorder.Calls(), skin.PartOverlay, row, core.StateHovered) {
		t.Fatal("UI did not render hovered dropdown row")
	}
	selected := -1
	u.OnClick("class", func() { selected = dropdown.DropdownIndex() })
	if !u.HandleMouse(MouseEvent{Pos: point, Pressed: true}) || !u.HandleMouse(MouseEvent{Pos: point, Released: true}) {
		t.Fatal("popup row gesture was not consumed")
	}
	if selected != 1 || dropdown.DropdownIndex() != 1 || u.Focused() != nil {
		t.Fatalf("dropdown selection=%d/%d focused=%v", selected, dropdown.DropdownIndex(), u.Focused())
	}
}

// TestDrawWidgetsDefersPopupForAppLayers keeps the popup above app chrome.
// DrawWidgets must render widgets without popup parts in the same frame
// that a following DrawPopup extends, so apps can sandwich their own
// layers between widgets and the popup.
func TestDrawWidgetsDefersPopupForAppLayers(t *testing.T) {
	u := New(200, 200)
	dropdown := widgets.NewDropdown("class", core.Rect{X: 10, Y: 10, W: 100, H: 30}, []string{"A", "B"}, 0)
	mustAdd(t, u, dropdown)
	clickAt(u, centerOf(dropdown))
	content := u.Theme().DropdownPopupContent(dropdown.DropdownPopupBounds(), core.StatePressed)
	row, _ := render.DropdownPopupRow(content, dropdown.DropdownItemCount(), 1)
	point := core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
	u.HandleMouse(MouseEvent{Pos: point})
	recorder := attachDrawRecorder(t, u)
	u.DrawWidgets()
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartPopup || call.Part == skin.PartPopupBorder || call.Part == skin.PartOverlay {
			t.Fatalf("DrawWidgets emitted popup part %v", call.Part)
		}
	}
	if len(recorder.Calls()) == 0 {
		t.Fatal("DrawWidgets rendered no widget calls")
	}
	u.DrawPopup()
	calls := recorder.Calls()
	if !hasDrawCall(calls, skin.PartPopup, dropdown.DropdownPopupBounds(), core.StatePressed) {
		t.Fatal("DrawPopup did not render the popup above app layers")
	}
	if !hasDrawCall(calls, skin.PartOverlay, row, core.StateHovered) {
		t.Fatal("DrawPopup did not render the hovered dropdown row")
	}
	var nilUI *UI
	nilUI.DrawWidgets()
	nilUI.DrawPopup()
}

// TestDrawUsesComputedStateAndResetsRecorderFrame checks snapshot drawing.
func TestDrawUsesComputedStateAndResetsRecorderFrame(t *testing.T) {
	u := New(100, 100)
	button := widgets.NewButton("button", core.Rect{W: 20, H: 20}, "button")
	mustAdd(t, u, button)
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	recorder := attachDrawRecorder(t, u)
	u.Draw()
	if info := recorder.LastWidgetInfo(); info.Name != "button" || info.State != core.StateHovered {
		t.Fatalf("draw snapshot = %+v", info)
	}
	count := len(recorder.Calls())
	u.Theme().DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 1, H: 1}, core.StateNormal)
	if len(recorder.Calls()) != count+1 {
		t.Fatal("direct draw was not recorded")
	}
	u.Draw()
	if len(recorder.Calls()) != count {
		t.Fatal("UI.Draw did not begin a fresh recorder frame")
	}
}

// TestResolvedBoundsDriveInputAndDrawing verifies hit tests, slider mapping,
// snapshots, and border parts read directly from arranged widget frames.
func TestResolvedBoundsDriveInputAndDrawing(t *testing.T) {
	u := New(400, 300)
	root := layout.New("root", core.Rect{W: 400, H: 300})
	button := widgets.NewButton("button", core.Rect{W: 80, H: 30}, "button")
	slider := widgets.NewSlider("slider", core.Rect{W: 200, H: 20}, 0)
	mustAdd(t, u, button, slider)
	arrangeInputWidgets(t, root, button, slider)
	if u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 5, Y: 5}}) || u.Hovered() != nil {
		t.Fatal("input used stale constructor position")
	}
	if !u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 150, Y: 125}, Pressed: true}) || slider.Value() != 0.25 {
		t.Fatalf("slider mapping used wrong bounds: %v", slider.Value())
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 150, Y: 125}, Released: true})
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	recorder := attachDrawRecorder(t, u)
	u.Draw()
	want := core.Rect{X: 40, Y: 50, W: 80, H: 30}
	if info := recorder.LastWidgetInfo(); info.Name != "slider" || info.Bounds != (core.Rect{X: 100, Y: 120, W: 200, H: 20}) {
		t.Fatalf("last snapshot = %+v", info)
	}
	if !hasDrawCall(recorder.Calls(), skin.PartBorder, want, core.StateHovered) {
		t.Fatal("button border did not use resolved bounds")
	}
}

// arrangeInputWidgets resolves the controls used by the layout integration test.
func arrangeInputWidgets(t *testing.T, root *layout.Node, button, slider *widgets.Widget) {
	t.Helper()
	if err := root.AddChild(button.Frame()); err != nil {
		t.Fatal(err)
	}
	if err := root.AddChild(slider.Frame()); err != nil {
		t.Fatal(err)
	}
	if err := button.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: 40, Y: 50}); err != nil {
		t.Fatal(err)
	}
	if err := slider.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: 100, Y: 120}); err != nil {
		t.Fatal(err)
	}
	if err := layout.ArrangeRoot(root, root.AuthoredBounds()); err != nil {
		t.Fatal(err)
	}
}

// TestResizeWheelCallbacksAndNilSafety covers adjacent facade contracts.
func TestResizeWheelCallbacksAndNilSafety(t *testing.T) {
	u := New(100, 100)
	scroll := widgets.NewScrollPanel("scroll", core.Rect{W: 50, H: 50})
	mustAdd(t, u, scroll)
	if !u.HandleMouse(MouseEvent{Pos: centerOf(scroll), Wheel: 1}) || scroll.Scroll().Y == 0 {
		t.Fatal("wheel over scroll panel was not handled")
	}
	u.Resize(200, 200)
	if sx, sy := u.Scale(); sx != 2 || sy != 2 {
		t.Fatalf("Scale = %v/%v", sx, sy)
	}
	if point := u.ToLogical(core.Vec2{X: 100, Y: 50}); point != (core.Vec2{X: 50, Y: 25}) {
		t.Fatalf("ToLogical = %+v", point)
	}
	u.OnClick("future", func() {})
	u.OnClick("future", nil)
	if len(u.callbacks) != 0 {
		t.Fatal("empty callback record was retained")
	}
	var nilUI *UI
	if nilUI.HandleMouse(MouseEvent{}) || nilUI.HandleKey(KeyEvent{}) || nilUI.Activate("x") || nilUI.TypeText("x", "x") || nilUI.Focus("x") {
		t.Fatal("nil UI consumed input")
	}
	nilUI.ClearWidgets()
	nilUI.Draw()
	if nilUI.Theme() != nil || nilUI.Transform() != nil || nilUI.Lookup("x") != nil {
		t.Fatal("nil UI accessor returned data")
	}
}

// hasDrawCall searches a bounded recorder snapshot for one operation.
func hasDrawCall(calls []render.DrawCall, part skin.SkinPart, bounds core.Rect, state core.WidgetState) bool {
	for _, call := range calls {
		if call.Part == part && call.Bounds == bounds && call.State == state {
			return true
		}
	}
	return false
}

// TestDirectWidgetCallbacks verifies that handlers attached directly to widgets execute on activation.
func TestDirectWidgetCallbacks(t *testing.T) {
	u := New(200, 200)
	btn := widgets.NewButton("btn", core.Rect{W: 50, H: 30}, "OK")
	directClicked, stringClicked := false, false
	btn.OnClick(func() { directClicked = true })
	u.OnClick("btn", func() { stringClicked = true })
	mustAdd(t, u, btn)

	if !u.Activate("btn") || !directClicked || !stringClicked {
		t.Fatalf("directClicked=%v stringClicked=%v", directClicked, stringClicked)
	}

	slider := widgets.NewSlider("slider", core.Rect{Y: 40, W: 100, H: 20}, 0.2)
	var directVal float32
	slider.OnChange(func(v float32) { directVal = v })
	mustAdd(t, u, slider)
	clickAt(u, core.Vec2{X: 80, Y: 50})
	if directVal == 0 {
		t.Fatal("expected slider direct callback on interaction")
	}

	tab := widgets.NewTabBar("tabs", core.Rect{Y: 70, W: 100, H: 30}, []string{"A", "B"}, 0)
	var selectedIdx int
	tab.OnTabSelect(func(idx int) { selectedIdx = idx })
	mustAdd(t, u, tab)
	u.SelectTab("tabs", 1)
	if selectedIdx != 1 {
		t.Fatalf("expected tab direct callback 1, got %d", selectedIdx)
	}

	btn.SetTooltip("Direct tip")
	if tip, ok := u.TooltipText("btn"); !ok || tip != "Direct tip" {
		t.Fatalf("expected direct tooltip %q, got %q", "Direct tip", tip)
	}
}

// TestCanvasWidgetAndScrollPanelDrawer verifies that canvas and scroll content draw during UI.Draw.
func TestCanvasWidgetAndScrollPanelDrawer(t *testing.T) {
	u := New(200, 200)
	canvasDrawn, scrollDrawn := false, false
	canvas := widgets.NewCanvas("canvas", core.Rect{W: 50, H: 50}, func(b core.Rect) {
		canvasDrawn = true
	})
	scroll := widgets.NewScrollPanel("scroll", core.Rect{Y: 60, W: 80, H: 80})
	scroll.SetScrollContentDrawer(func(bounds core.Rect, offset core.Vec2) {
		scrollDrawn = true
	})
	mustAdd(t, u, canvas, scroll)

	u.Draw()
	if !canvasDrawn || !scrollDrawn {
		t.Fatalf("canvasDrawn=%v scrollDrawn=%v", canvasDrawn, scrollDrawn)
	}
}

// TestPollRaylibInputHeadless verifies that PollRaylibInput safely returns zero without a display.
func TestPollRaylibInputHeadless(t *testing.T) {
	u := New(100, 100)
	result := u.PollRaylibInput()
	if result.MouseHandled || result.KeyHandled {
		t.Fatalf("expected zero result in headless mode, got %+v", result)
	}
}
