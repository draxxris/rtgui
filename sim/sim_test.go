package sim

import (
	"errors"
	"testing"

	"rtgui/core"
	"rtgui/input"
	"rtgui/render"
	"rtgui/transform"
	"rtgui/widgets"
)

// TestStringIDModel verifies the string-external numeric-internal identity.
// Constructors must store Name, derive a deterministic nonzero ID, and
// propagate Name through Info.
func TestStringIDModel(t *testing.T) {
	first := widgets.NewButton("okButton", core.Rect{W: 120, H: 40}, "OK")
	second := widgets.NewButton("okButton", core.Rect{W: 120, H: 40}, "OK")
	other := widgets.NewButton("cancelButton", core.Rect{W: 120, H: 40}, "Cancel")
	if first.Name != "okButton" {
		t.Fatalf("Name = %q, want okButton", first.Name)
	}
	if first.ID == 0 {
		t.Fatal("internal ID must never be 0")
	}
	if first.ID != second.ID {
		t.Fatalf("same name gave different IDs %d vs %d", first.ID, second.ID)
	}
	if first.ID == other.ID {
		t.Fatalf("distinct names collided on ID %d", first.ID)
	}
	info := first.Info()
	if info.Name != "okButton" {
		t.Fatalf("Info().Name = %q, want okButton", info.Name)
	}
	if info.ID != first.ID {
		t.Fatalf("Info().ID = %d, want %d", info.ID, first.ID)
	}
	// All constructors store the external name with a nonzero internal ID.
	all := []*widgets.Widget{
		widgets.NewRectangle("rect1", core.Rect{W: 10, H: 10}),
		widgets.NewLabel("label1", core.Rect{W: 10, H: 10}, "hi"),
		widgets.NewCheckbox("check1", core.Rect{W: 20, H: 20}, false),
		widgets.NewTextbox("text1", core.Rect{W: 100, H: 30}, 64),
		widgets.NewSlider("slider1", core.Rect{W: 100, H: 10}, 0.5),
		widgets.NewProgressBar("progress1", core.Rect{W: 100, H: 10}, 0.5),
		widgets.NewScrollPanel("scroll1", core.Rect{W: 100, H: 100}),
		widgets.NewDropdown("drop1", core.Rect{W: 100, H: 30}, []string{"a"}, 0),
		widgets.NewFrame("frame1", core.Rect{W: 100, H: 100}),
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

// TestRegisterValidation covers empty-name, duplicate, and nil rejection.
// No path may panic and duplicates must not clobber the original entry.
func TestRegisterValidation(t *testing.T) {
	stage := NewStage()
	if err := stage.Register(widgets.NewButton("okButton", core.Rect{W: 100, H: 30}, "OK")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	dup := widgets.NewButton("okButton", core.Rect{W: 100, H: 30}, "Other")
	if err := stage.Register(dup); !errors.Is(err, ErrDuplicateWidget) {
		t.Fatalf("duplicate Register = %v, want ErrDuplicateWidget", err)
	}
	if err := stage.Register(widgets.NewButton("", core.Rect{W: 10, H: 10}, "x")); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("empty-name Register did not return ErrEmptyName")
	}
	if err := stage.Register(nil); !errors.Is(err, ErrNilWidget) {
		t.Fatalf("nil Register = %v, want ErrNilWidget", err)
	}
	var nilStage *Stage
	if err := nilStage.Register(widgets.NewButton("x", core.Rect{W: 10, H: 10}, "x")); !errors.Is(err, ErrNilStage) {
		t.Fatalf("nil-stage Register = %v, want ErrNilStage", err)
	}
	if err := stage.Register(&widgets.Widget{}); !errors.Is(err, ErrEmptyName) {
		t.Fatalf("zero widget Register did not return ErrEmptyName")
	}
}

// TestDuplicatePreservesOriginal proves a duplicate registration keeps the
// first widget reachable via Click instead of clobbering it.
func TestDuplicatePreservesOriginal(t *testing.T) {
	stage := NewStage()
	first := widgets.NewButton("dupButton", core.Rect{W: 100, H: 30}, "first")
	if err := stage.Register(first); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Same name but disabled: if it clobbered, Click would return false.
	shadow := widgets.NewButton("dupButton", core.Rect{W: 100, H: 30}, "second")
	shadow.Enabled = false
	if err := stage.Register(shadow); !errors.Is(err, ErrDuplicateWidget) {
		t.Fatalf("duplicate Register = %v, want ErrDuplicateWidget", err)
	}
	if !stage.Click("dupButton") {
		t.Fatal("Click after duplicate should still reach the original widget")
	}
	if shadow.State == core.StateHovered || shadow.State == core.StatePressed {
		t.Fatal("shadow widget must not have been driven by Click")
	}
}

// TestClickButton verifies a button click returns true and releases capture.
func TestClickButton(t *testing.T) {
	capture := input.NewCapture()
	stage := NewStageWithCapture(capture)
	button := widgets.NewButton("okButton", core.Rect{W: 100, H: 30}, "OK")
	if err := stage.Register(button); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !stage.Click("okButton") {
		t.Fatal("Click on button should return true")
	}
	if capture.IsCaptured() {
		t.Fatal("Click must release capture")
	}
	if button.State != core.StateHovered {
		t.Fatalf("button state after click = %v, want hovered", button.State)
	}
}

// TestClickCheckbox verifies Click toggles checkbox state via Release.
func TestClickCheckbox(t *testing.T) {
	stage := NewStage()
	box := widgets.NewCheckbox("agreeBox", core.Rect{W: 20, H: 20}, false)
	if err := stage.Register(box); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !stage.Click("agreeBox") {
		t.Fatal("Click on checkbox should return true")
	}
	if !box.Checked {
		t.Fatal("checkbox should toggle on click")
	}
	if !stage.Click("agreeBox") {
		t.Fatal("second Click should also return true")
	}
	if box.Checked {
		t.Fatal("checkbox should toggle back on second click")
	}
}

// TestClickDisabledAndUnknown covers the false paths and proves capture is
// never left held after a miss.
func TestClickDisabledAndUnknown(t *testing.T) {
	capture := input.NewCapture()
	stage := NewStageWithCapture(capture)
	disabled := widgets.NewButton("offButton", core.Rect{W: 100, H: 30}, "off")
	disabled.Enabled = false
	if err := stage.Register(disabled); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if stage.Click("offButton") {
		t.Fatal("Click on disabled widget must return false")
	}
	if stage.Click("missingButton") {
		t.Fatal("Click on unknown ID must return false")
	}
	if stage.Click("") {
		t.Fatal("Click on empty name must return false")
	}
	if capture.IsCaptured() {
		t.Fatal("failed Click must not leave capture held")
	}
	// A later valid click still works, proving no stuck state.
	ok := widgets.NewButton("onButton", core.Rect{W: 100, H: 30}, "on")
	if err := stage.Register(ok); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !stage.Click("onButton") {
		t.Fatal("valid Click after misses should return true")
	}
	if capture.IsCaptured() {
		t.Fatal("valid Click must release capture")
	}
}

// TestClickUsesCenter proves Click drives the widget-center point by using
// offset bounds where the origin would miss.
func TestClickUsesCenter(t *testing.T) {
	stage := NewStage()
	button := widgets.NewButton("farButton", core.Rect{X: 200, Y: 150, W: 120, H: 40}, "far")
	if err := stage.Register(button); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Origin (0,0) is far outside these bounds; success implies center math.
	if !stage.Click("farButton") {
		t.Fatal("Click should hit the widget center for offset bounds")
	}
	center := core.Vec2{X: 200 + 120.0/2, Y: 150 + 40.0/2}
	if !button.Bounds.Contains(center) {
		t.Fatalf("test center %v outside bounds %v", center, button.Bounds)
	}
}

// TestTypeAppendUTF8 verifies focus-first append semantics for ASCII and
// multi-byte runes via the existing TypeChar path.
func TestTypeAppendUTF8(t *testing.T) {
	stage := NewStage()
	field := widgets.NewTextbox("myTextField", core.Rect{W: 200, H: 30}, 256)
	if err := stage.Register(field); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !stage.Type("myTextField", "aé😀中") {
		t.Fatal("Type should return true for a textbox")
	}
	if got := field.TextBuf.String(); got != "aé😀中" {
		t.Fatalf("buffer = %q, want %q", got, "aé😀中")
	}
	if !stage.Type("myTextField", "hello") {
		t.Fatal("second Type should also return true")
	}
	if got, want := field.TextBuf.String(), "aé😀中hello"; got != want {
		t.Fatalf("append buffer = %q, want %q", got, want)
	}
	if !field.Focused {
		t.Fatal("Type must focus its target")
	}
}

// TestTypeWrongKindDisabledUnknown covers the Type false paths without
// mutating any buffer.
func TestTypeWrongKindDisabledUnknown(t *testing.T) {
	stage := NewStage()
	button := widgets.NewButton("okButton", core.Rect{W: 100, H: 30}, "OK")
	box := widgets.NewCheckbox("agreeBox", core.Rect{W: 20, H: 20}, false)
	off := widgets.NewTextbox("offField", core.Rect{W: 200, H: 30}, 64)
	off.Enabled = false
	for _, w := range []*widgets.Widget{button, box, off} {
		if err := stage.Register(w); err != nil {
			t.Fatalf("Register %q: %v", w.Name, err)
		}
	}
	if stage.Type("okButton", "hi") {
		t.Fatal("Type on a button must return false")
	}
	if stage.Type("agreeBox", "hi") {
		t.Fatal("Type on a checkbox must return false")
	}
	if stage.Type("offField", "hi") {
		t.Fatal("Type on a disabled textbox must return false")
	}
	if got := off.TextBuf.String(); got != "" {
		t.Fatalf("disabled Type mutated buffer to %q", got)
	}
	if stage.Type("missingField", "hi") {
		t.Fatal("Type on unknown ID must return false")
	}
	if stage.Type("", "hi") {
		t.Fatal("Type on empty name must return false")
	}
}

// TestTypeFocusBehavior verifies Type focuses its target, blurs the
// previous target, and treats empty text as a focusing no-op success.
func TestTypeFocusBehavior(t *testing.T) {
	stage := NewStage()
	first := widgets.NewTextbox("firstField", core.Rect{W: 200, H: 30}, 64)
	second := widgets.NewTextbox("secondField", core.Rect{W: 200, H: 30}, 64)
	if err := stage.Register(first); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := stage.Register(second); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !stage.Type("firstField", "ab") {
		t.Fatal("Type on first field should succeed")
	}
	if !first.Focused {
		t.Fatal("first field should be focused after Type")
	}
	if !stage.Type("secondField", "cd") {
		t.Fatal("Type on second field should succeed")
	}
	if !second.Focused {
		t.Fatal("second field should be focused after Type")
	}
	if first.Focused {
		t.Fatal("first field should blur when focus moves")
	}
	if got := first.TextBuf.String(); got != "ab" {
		t.Fatalf("first buffer = %q, want ab", got)
	}
	// Empty text still focuses and succeeds without mutating the buffer.
	third := widgets.NewTextbox("thirdField", core.Rect{W: 200, H: 30}, 64)
	if err := stage.Register(third); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !stage.Type("thirdField", "") {
		t.Fatal("empty-string Type must return true")
	}
	if !third.Focused {
		t.Fatal("empty-string Type must still focus the target")
	}
	if got := third.TextBuf.String(); got != "" {
		t.Fatalf("empty-string Type mutated buffer to %q", got)
	}
	if second.Focused {
		t.Fatal("previous field should blur even on empty-string Type")
	}
}

// TestNilSafety verifies nil stages, captures, and names never panic.
func TestNilSafety(t *testing.T) {
	var nilStage *Stage
	if nilStage.Click("anything") {
		t.Fatal("nil-stage Click must return false")
	}
	if nilStage.Type("anything", "text") {
		t.Fatal("nil-stage Type must return false")
	}
	if err := nilStage.Register(widgets.NewButton("x", core.Rect{W: 10, H: 10}, "x")); err == nil {
		t.Fatal("nil-stage Register must return an error")
	}
	// Zero-value Stage and nil-capture Stage remain usable.
	var zero Stage
	if err := zero.Register(widgets.NewButton("zeroButton", core.Rect{W: 100, H: 30}, "z")); err != nil {
		t.Fatalf("zero-value Register: %v", err)
	}
	if !zero.Click("zeroButton") {
		t.Fatal("zero-value Click should succeed")
	}
	nilCapture := NewStageWithCapture(nil)
	if err := nilCapture.Register(widgets.NewTextbox("nilCapField", core.Rect{W: 200, H: 30}, 64)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !nilCapture.Type("nilCapField", "hi") {
		t.Fatal("nil-capture Stage Type should succeed")
	}
}

// TestMiniStageIntegration drives a button, textbox, and checkbox together
// and proves the renderer path still works with string IDs.
func TestMiniStageIntegration(t *testing.T) {
	stage := NewStage()
	button := widgets.NewButton("okButton", core.Rect{X: 10, Y: 10, W: 120, H: 40}, "OK")
	field := widgets.NewTextbox("myTextField", core.Rect{X: 10, Y: 60, W: 200, H: 30}, 256)
	box := widgets.NewCheckbox("agreeBox", core.Rect{X: 10, Y: 100, W: 20, H: 20}, false)
	for _, w := range []*widgets.Widget{button, field, box} {
		if err := stage.Register(w); err != nil {
			t.Fatalf("Register %q: %v", w.Name, err)
		}
	}
	if !stage.Click("okButton") {
		t.Fatal("Click okButton should succeed")
	}
	if !stage.Click("agreeBox") || !box.Checked {
		t.Fatal("Click agreeBox should check the box")
	}
	if !stage.Type("myTextField", "hello world") {
		t.Fatal("Type hello world should succeed")
	}
	if got := field.TextBuf.String(); got != "hello world" {
		t.Fatalf("buffer = %q, want hello world", got)
	}
	// Renderer path: each widget snapshot draws headlessly with its Name.
	theme := render.NewTheme(transform.New(core.Viewport{}))
	theme.ClearDrawLog()
	for _, w := range []*widgets.Widget{button, field, box} {
		info := w.Info()
		if info.Name != w.Name {
			t.Fatalf("Info().Name = %q, want %q", info.Name, w.Name)
		}
		text := w.Text
		if w.TextBuf != nil {
			text = w.TextBuf.String()
		}
		if err := theme.DrawWidget(info, text, w.Value, w.Checked); err != nil {
			t.Fatalf("DrawWidget %q: %v", w.Name, err)
		}
		if got := theme.LastWidgetInfo(); got.Name != w.Name {
			t.Fatalf("LastWidgetInfo().Name = %q, want %q", got.Name, w.Name)
		}
	}
	if len(theme.DrawLog()) == 0 {
		t.Fatal("expected draw calls for the mini stage")
	}
}
