package widgets_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// TestTextboxCaretMoves verifies widget caret motion and selection bounds.
func TestTextboxCaretMoves(t *testing.T) {
	field := widgets.NewTextbox("field", core.Rect{}, 16)
	if !field.SetText("abcd") || field.Caret() != 4 {
		t.Fatalf("set caret = %d", field.Caret())
	}
	if !field.MoveCaret(-2, false) || field.Caret() != 2 {
		t.Fatalf("move caret = %d", field.Caret())
	}
	if !field.MoveCaretTo(1, true) || !field.HasSelection() {
		t.Fatal("shift move must select")
	}
	if start, end := field.Selection(); start != 1 || end != 2 {
		t.Fatalf("selection = %d/%d", start, end)
	}
	if got := field.SelectedText(); got != "b" {
		t.Fatalf("selected = %q", got)
	}
	if field.RuneCount() != 4 {
		t.Fatalf("rune count = %d", field.RuneCount())
	}
}

// TestTextboxSelectionEdits verifies selection deletion and insertion.
func TestTextboxSelectionEdits(t *testing.T) {
	field := widgets.NewTextbox("field", core.Rect{}, 16)
	field.SetText("abcd")
	field.MoveCaret(-2, false)
	field.MoveCaretTo(1, true)
	if !field.DeleteSelection() || field.Text() != "acd" {
		t.Fatalf("delete selection = %q", field.Text())
	}
	if field.SetCaret(1) {
		t.Fatal("redundant SetCaret must report no change")
	}
	if !field.InsertString("XY") || field.Text() != "aXYcd" {
		t.Fatalf("insert string = %q", field.Text())
	}
	if !field.SetCaret(0) {
		t.Fatal("home failed")
	}
	if !field.Delete() || field.Text() != "XYcd" {
		t.Fatalf("forward delete = %q", field.Text())
	}
	if !field.SelectAll() || !field.ClearSelection() || field.HasSelection() {
		t.Fatal("select/clear failed")
	}
}

// TestTextboxCaretNilZero verifies nil zero values.
func TestTextboxCaretNilZero(t *testing.T) {
	var nilWidget *widgets.Textbox
	if nilWidget.Caret() != 0 {
		t.Fatal("nil caret must be zero")
	}
	if nilWidget.HasSelection() {
		t.Fatal("nil selection must be empty")
	}
	if nilWidget.RuneCount() != 0 {
		t.Fatal("nil count must be zero")
	}
	if _, _ = nilWidget.Selection(); nilWidget.SelectedText() != "" {
		t.Fatal("nil selection text must be empty")
	}
}

// TestTextboxCaretNilEdits verifies nil and wrong-kind edits fail.
func TestTextboxCaretNilEdits(t *testing.T) {
	var nilWidget *widgets.Textbox
	if nilWidget.SetCaret(1) {
		t.Fatal("nil SetCaret must fail")
	}
	if nilWidget.MoveCaret(1, false) {
		t.Fatal("nil MoveCaret must fail")
	}
	if nilWidget.SelectAll() {
		t.Fatal("nil SelectAll must fail")
	}
	if nilWidget.Delete() {
		t.Fatal("nil Delete must fail")
	}
	if nilWidget.InsertString("x") {
		t.Fatal("nil insert must fail")
	}
	button := widgets.NewButton("button", core.Rect{}, "OK")
	if _, err := widgets.AsTextbox(button); err == nil {
		t.Fatal("expected ErrKindMismatch when casting button to Textbox")
	}
}
