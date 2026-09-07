package sim

import (
	"errors"
	"reflect"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/ui"
	"github.com/draxxris/rtgui/widgets"
)

// newStageUI builds a registered UI and its borrowed semantic adapter.
func newStageUI(t *testing.T, list ...widgets.Widget) (*Stage, *ui.UI) {
	t.Helper()
	facade := ui.New(200, 200)
	if err := facade.Add(list...); err != nil {
		t.Fatalf("Add: %v", err)
	}
	stage, err := NewStage(facade)
	if err != nil {
		t.Fatalf("NewStage: %v", err)
	}
	return stage, facade
}

// TestStageContainsOnlyExistingUI verifies construction and the minimal Stage shape.
func TestStageContainsOnlyExistingUI(t *testing.T) {
	if stage, err := NewStage(nil); !errors.Is(err, ErrNilUI) || stage != nil {
		t.Fatalf("NewStage(nil) = %#v, %v", stage, err)
	}
	stageType := reflect.TypeOf(Stage{})
	if stageType.NumField() != 1 || stageType.Field(0).Type != reflect.TypeOf((*ui.UI)(nil)) {
		t.Fatalf("Stage fields = %v", stageType.NumField())
	}
}

// TestSimulationCallbacksMatchPhysicalInput compares shared activation semantics.
func TestSimulationCallbacksMatchPhysicalInput(t *testing.T) {
	simulated := widgets.NewCheckbox("check", core.Rect{W: 20, H: 20}, false)
	physical := widgets.NewCheckbox("check", core.Rect{W: 20, H: 20}, false)
	stage, simulatedUI := newStageUI(t, simulated)
	_, physicalUI := newStageUI(t, physical)
	var simulatedSeen, physicalSeen bool
	simulatedUI.OnClick("check", func() { simulatedSeen = simulated.Checked() })
	physicalUI.OnClick("check", func() { physicalSeen = physical.Checked() })
	if !stage.Click("check") {
		t.Fatal("simulated activation failed")
	}
	point := core.Vec2{X: 10, Y: 10}
	if !physicalUI.HandleMouse(ui.MouseEvent{Pos: point, Pressed: true}) || !physicalUI.HandleMouse(ui.MouseEvent{Pos: point, Released: true}) {
		t.Fatal("physical activation failed")
	}
	if !simulated.Checked() || !physical.Checked() || !simulatedSeen || !physicalSeen {
		t.Fatal("simulation and physical callbacks observed different checkbox state")
	}
}

// TestSimulationTypingUsesUICallbacksAndFocus checks shared text transitions.
func TestSimulationTypingUsesUICallbacksAndFocus(t *testing.T) {
	field := widgets.NewTextbox("field", core.Rect{W: 100, H: 20}, 16)
	stage, facade := newStageUI(t, field)
	var values []string
	facade.OnText("field", func(value string) { values = append(values, value) })
	if !stage.Type("field", "aé😀") || field.Text() != "aé😀" || facade.Focused() != field {
		t.Fatalf("typed text=%q focused=%v", field.Text(), facade.Focused())
	}
	if len(values) != 1 || values[0] != "aé😀" {
		t.Fatalf("text callbacks = %v", values)
	}
	if !stage.Type("field", "") || len(values) != 1 {
		t.Fatal("empty simulated type must focus without callback")
	}
	if !stage.Focus("field") || facade.Focused() != field {
		t.Fatal("simulated focus failed")
	}
}

// TestSimulationRejectsInvalidTargets checks semantic validation and nil safety.
func TestSimulationRejectsInvalidTargets(t *testing.T) {
	button := widgets.NewButton("button", core.Rect{}, "button")
	disabled := widgets.NewTextbox("disabled", core.Rect{}, 8)
	disabled.SetEnabled(false)
	stage, facade := newStageUI(t, button, disabled)
	if stage.Click("missing") || stage.Type("button", "x") || stage.Type("disabled", "x") || stage.Focus("button") {
		t.Fatal("simulation accepted an invalid target")
	}
	if disabled.Text() != "" || facade.Focused() != nil {
		t.Fatal("invalid simulation mutated UI")
	}
	var nilStage *Stage
	if nilStage.Click("x") || nilStage.Type("x", "x") || nilStage.Focus("x") {
		t.Fatal("nil Stage handled an operation")
	}
}

// TestTwoSimulationInstancesShareNoState verifies instance ownership.
func TestTwoSimulationInstancesShareNoState(t *testing.T) {
	firstWidget := widgets.NewCheckbox("same", core.Rect{}, false)
	secondWidget := widgets.NewCheckbox("same", core.Rect{}, false)
	firstStage, firstUI := newStageUI(t, firstWidget)
	secondStage, secondUI := newStageUI(t, secondWidget)
	firstCalls, secondCalls := 0, 0
	firstUI.OnClick("same", func() { firstCalls++ })
	secondUI.OnClick("same", func() { secondCalls++ })
	if !firstStage.Click("same") {
		t.Fatal("first stage activation failed")
	}
	if !firstWidget.Checked() || secondWidget.Checked() || firstCalls != 1 || secondCalls != 0 {
		t.Fatal("first stage leaked state into second")
	}
	if !secondStage.Click("same") || !secondWidget.Checked() || secondCalls != 1 {
		t.Fatal("second stage activation failed independently")
	}
}

// TestSimulationFrameFocusAndHotkeys checks scoped hotkeys through the stage.
func TestSimulationFrameFocusAndHotkeys(t *testing.T) {
	frame := widgets.NewFrame("demoFrame", core.Rect{X: 10, Y: 10, W: 100, H: 100})
	stage, facade := newStageUI(t, frame)
	fires := 0
	facade.OnHotkey("move", 'R', ui.HotkeyOpts{Scope: "demoFrame", Consume: true}, func() { fires++ })
	if stage.PressHotkey('R') || fires != 0 {
		t.Fatal("out-of-scope hotkey must pass through without firing")
	}
	if !stage.FocusFrame("demoFrame") || facade.ActiveFrame() != frame {
		t.Fatal("simulated frame focus failed")
	}
	if !stage.PressHotkey('r') || fires != 1 {
		t.Fatal("in-scope hotkey must fire case-insensitively and consume")
	}
	var nilStage *Stage
	if nilStage.PressHotkey('R') || nilStage.FocusFrame("demoFrame") {
		t.Fatal("nil Stage handled frame operations")
	}
	if stage.FocusFrame("missing") {
		t.Fatal("invalid frame focus accepted")
	}
}
