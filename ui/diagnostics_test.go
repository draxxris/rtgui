package ui

import (
	"strings"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// TestDiagnosticsReportInvalidOperations verifies that operations on invalid
// or mismatched widgets record actionable diagnostic messages.
func TestDiagnosticsReportInvalidOperations(t *testing.T) {
	u := New(800, 600)
	var reported []string
	u.SetDiagnosticHandler(func(msg string) {
		reported = append(reported, msg)
	})

	btn := widgets.NewButton("btn1", core.Rect{X: 10, Y: 10, W: 100, H: 30}, "Click")
	btn.SetEnabled(false)
	mustAdd(t, u, btn)

	// Activate on unknown widget
	if u.Activate("nonexistent") {
		t.Fatal("expected Activate on unknown widget to return false")
	}
	// Activate on disabled widget
	if u.Activate("btn1") {
		t.Fatal("expected Activate on disabled widget to return false")
	}
	// SelectTab on non-tab widget
	if u.SelectTab("btn1", 1) {
		t.Fatal("expected SelectTab on button to return false")
	}
	// TypeText on non-textbox widget
	if u.TypeText("btn1", "text") {
		t.Fatal("expected TypeText on button to return false")
	}
	// Focus on unknown widget
	if u.Focus("nonexistent") {
		t.Fatal("expected Focus on unknown widget to return false")
	}

	diags := u.Diagnostics()
	if len(diags) < 5 {
		t.Fatalf("expected at least 5 diagnostics, got %d: %v", len(diags), diags)
	}
	if len(reported) != len(diags) {
		t.Fatalf("expected handler calls (%d) to match diagnostics count (%d)", len(reported), len(diags))
	}

	u.ClearDiagnostics()
	if len(u.Diagnostics()) != 0 {
		t.Fatalf("expected ClearDiagnostics to empty the list, got %d", len(u.Diagnostics()))
	}
}

// TestDebugModeToggle verifies Theme and UI debug mode state.
func TestDebugModeToggle(t *testing.T) {
	u := New(800, 600)
	if u.DebugMode() {
		t.Fatal("default debug mode must be false")
	}
	u.SetDebugMode(true)
	if !u.DebugMode() {
		t.Fatal("expected debug mode to be true")
	}
	if !u.Theme().DebugMode() {
		t.Fatal("expected Theme debug mode to match UI debug mode")
	}
	u.SetDebugMode(false)
	if u.DebugMode() {
		t.Fatal("expected debug mode to be false")
	}
}

// TestRenderMissingSkinReportsDiagnostics verifies draw fallbacks trigger diagnostic messages.
func TestRenderMissingSkinReportsDiagnostics(t *testing.T) {
	u := New(800, 600)
	btn := widgets.NewButton("unskinned", core.Rect{X: 10, Y: 10, W: 100, H: 30}, "Test")
	mustAdd(t, u, btn)

	var reported []string
	u.SetDiagnosticHandler(func(msg string) {
		reported = append(reported, msg)
	})

	u.Draw()

	found := false
	for _, msg := range reported {
		if strings.Contains(msg, "missing skin") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing skin diagnostic in draw output, got: %v", reported)
	}
}
