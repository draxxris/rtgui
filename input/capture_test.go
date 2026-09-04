package input

import "testing"

func TestCapture(t *testing.T) {
	capture := NewCapture()
	if capture.IsCaptured() {
		t.Fatal("capture should start inactive")
	}
	if err := capture.Set(42); err != nil {
		t.Fatalf("set capture: %v", err)
	}
	if !capture.IsCaptured() || capture.ID() != 42 {
		t.Fatalf("capture state: active=%v id=%d", capture.IsCaptured(), capture.ID())
	}
	capture.Release()
	if capture.IsCaptured() || capture.ID() != 0 {
		t.Fatal("capture should be released")
	}
}
