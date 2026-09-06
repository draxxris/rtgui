package text

import "testing"

// TestBufferCaretMovesByRune verifies arrow-style motion and clamping.
func TestBufferCaretMovesByRune(t *testing.T) {
	b := NewBuffer(16, "aé😀")
	if got := b.Caret(); got != 3 {
		t.Fatalf("initial caret = %d, want end 3", got)
	}
	if !b.MoveCaret(-1, false) || b.Caret() != 2 {
		t.Fatalf("move left caret = %d", b.Caret())
	}
	if !b.MoveCaretTo(0, false) || b.Caret() != 0 {
		t.Fatalf("move home caret = %d", b.Caret())
	}
	if b.MoveCaret(-1, false) {
		t.Fatal("move past start must report no change")
	}
	if !b.MoveCaretTo(3, false) || b.Caret() != 3 {
		t.Fatalf("move end caret = %d", b.Caret())
	}
	if b.MoveCaret(1, false) {
		t.Fatal("move past end must report no change")
	}
	if !b.SetCaret(1) || b.Caret() != 1 {
		t.Fatalf("set caret = %d", b.Caret())
	}
	if b.SetCaret(1) {
		t.Fatal("unchanged SetCaret must report no change")
	}
}

// TestBufferInsertsAtCaret verifies mid-text insertion and capacity reuse.
func TestBufferInsertsAtCaret(t *testing.T) {
	b := NewBuffer(16, "ac")
	if !b.SetCaret(1) {
		t.Fatal("SetCaret to middle failed")
	}
	if !b.AppendRune('B') || b.String() != "aBc" || b.Caret() != 2 {
		t.Fatalf("insert at caret = %q/%d", b.String(), b.Caret())
	}
	if !b.Backspace() || b.String() != "ac" || b.Caret() != 1 {
		t.Fatalf("backspace at caret = %q/%d", b.String(), b.Caret())
	}
	if !b.Delete() || b.String() != "a" || b.Caret() != 1 {
		t.Fatalf("forward delete = %q/%d", b.String(), b.Caret())
	}
	if b.Delete() {
		t.Fatal("delete at end must report no change")
	}
}

// TestBufferSelectionReplacesOnType verifies select-all plus typing flow.
func TestBufferSelectionReplacesOnType(t *testing.T) {
	b := NewBuffer(16, "hello")
	if !b.SelectAll() || !b.HasSelection() {
		t.Fatal("SelectAll failed")
	}
	if start, end := b.Selection(); start != 0 || end != 5 {
		t.Fatalf("selection = %d/%d", start, end)
	}
	if got := b.SelectedText(); got != "hello" {
		t.Fatalf("selected text = %q", got)
	}
	if !b.AppendRune('X') || b.String() != "X" || b.Caret() != 1 || b.HasSelection() {
		t.Fatalf("type over selection = %q/%d sel=%v", b.String(), b.Caret(), b.HasSelection())
	}
}

// TestBufferShiftExtendsSelection verifies anchor handling and collapse.
func TestBufferShiftExtendsSelection(t *testing.T) {
	b := NewBuffer(16, "abcd")
	if !b.MoveCaretTo(1, false) {
		t.Fatal("initial move failed")
	}
	if !b.MoveCaret(2, true) || !b.HasSelection() {
		t.Fatal("shift extend failed")
	}
	if start, end := b.Selection(); start != 1 || end != 3 {
		t.Fatalf("extended selection = %d/%d", start, end)
	}
	if !b.MoveCaret(-2, true) || b.HasSelection() {
		t.Fatalf("shift collapse must clear, sel=%v caret=%d", b.HasSelection(), b.Caret())
	}
	if !b.SelectAll() || !b.ClearSelection() || b.HasSelection() {
		t.Fatal("clear selection failed")
	}
	if b.ClearSelection() || b.DeleteSelection() {
		t.Fatal("idle clear/delete must report no change")
	}
}

// TestBufferInsertStringHonorsLimit verifies paste truncation and selection free.
func TestBufferInsertStringHonorsLimit(t *testing.T) {
	b := NewBuffer(4, "ab")
	b.SelectAll()
	// Deleting "ab" frees 2 bytes; "wxyz" needs 4, so only "wx" fits after
	// removal plus the freed bytes allow a partial insert with mutation.
	if !b.InsertString("wxyz") {
		t.Fatal("paste over selection must mutate")
	}
	if got := b.String(); got != "wxyz"[:len(got)] || len(got) == 0 {
		t.Fatalf("truncated paste = %q", got)
	}
	full := NewBuffer(2, "hi")
	if full.InsertString("XY") {
		t.Fatal("paste into full buffer without selection must fail")
	}
	if got := full.String(); got != "hi" {
		t.Fatalf("full paste changed value to %q", got)
	}
}

// TestBufferSetResetsCaret verifies replacement moves caret to end.
func TestBufferSetResetsCaret(t *testing.T) {
	b := NewBuffer(16, "hello")
	b.SelectAll()
	if !b.Set("hi") || b.Caret() != 2 || b.HasSelection() {
		t.Fatalf("set reset = %d sel=%v", b.Caret(), b.HasSelection())
	}
}
