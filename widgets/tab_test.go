package widgets_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// TestTabBarCopiesLabels checks label copy semantics for inputs and outputs.
func TestTabBarCopiesLabels(t *testing.T) {
	labels := []string{"Inventory", "Skills", "Map"}
	bar := widgets.NewTabBar("tabs", core.Rect{X: 10, Y: 10, W: 300, H: 36}, labels, 0)
	labels[0] = "mutated input"
	if selected, ok := bar.TabSelection(); !ok || selected != "Inventory" {
		t.Fatalf("selection = %q/%v", selected, ok)
	}
	snapshot := bar.TabLabels()
	snapshot[1] = "mutated output"
	if got := bar.TabLabels()[1]; got != "Skills" {
		t.Fatalf("output mutation changed tab label to %q", got)
	}
}

// TestTabBarGuardsSelection checks selection bounds and shrinking labels.
func TestTabBarGuardsSelection(t *testing.T) {
	bar := widgets.NewTabBar("tabs", core.Rect{}, []string{"A", "B", "C"}, 0)
	if !bar.SetSelectedTab(2) || bar.SelectedTab() != 2 {
		t.Fatalf("selected tab = %d", bar.SelectedTab())
	}
	if bar.SetSelectedTab(2) || bar.SetSelectedTab(9) || bar.SelectedTab() != 2 {
		t.Fatal("duplicate or invalid selection changed tab bar")
	}
	if !bar.SetTabLabels([]string{"A"}) || bar.SelectedTab() != -1 || bar.TabCount() != 1 {
		t.Fatalf("shrunk labels = %v/%d", bar.TabLabels(), bar.SelectedTab())
	}
	if got := widgets.NewTabBar("empty", core.Rect{}, nil, 0).TabCount(); got != 0 {
		t.Fatalf("empty tab count = %d", got)
	}
	var nilWidget *widgets.Widget
	if nilWidget.TabCount() != 0 || nilWidget.SelectedTab() != -1 {
		t.Fatal("nil widget tab accessors must be zero")
	}
	if _, ok := nilWidget.TabSelection(); ok {
		t.Fatal("nil widget tab selection must be unset")
	}
}
