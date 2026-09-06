package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestParseTabsMenuTooltipSelectors verifies the new kind and part vocabulary.
func TestParseTabsMenuTooltipSelectors(t *testing.T) {
	kinds := map[string]core.WidgetKind{
		"TabBar": core.WidgetTabBar, "Menu": core.WidgetMenu, "Tooltip": core.WidgetTooltip,
	}
	for name, kind := range kinds {
		rules, err := ParseCSS(name + " { padding: 1; }")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(rules) != 1 || rules[0].Kind != kind {
			t.Fatalf("%s parsed as %+v", name, rules)
		}
	}
	if _, err := ParseCSS("TabBar::tab { background-image: url(\"a.png\"); }"); err != nil {
		t.Fatalf("TabBar::tab: %v", err)
	}
	if _, err := ParseCSS("TabBar::tab:selected { background-image: url(\"a.png\"); }"); err != nil {
		t.Fatalf("TabBar::tab:selected: %v", err)
	}
	if _, err := ParseCSS("Menu::popup { background-image: url(\"a.png\"); padding: 2; }"); err != nil {
		t.Fatalf("Menu::popup: %v", err)
	}
	if _, err := ParseCSS("Menu::highlight { background-image: url(\"a.png\"); }"); err != nil {
		t.Fatalf("Menu::highlight: %v", err)
	}
	rejected := []string{
		"TabBar::track { background-image: url(\"a.png\"); }",
		"TabBar::tab { border-image-source: url(\"a.png\"); }",
		"TabBar::tab { padding: 1; }",
		"Menu::tab { background-image: url(\"a.png\"); }",
		"Tooltip::popup { background-image: url(\"a.png\"); }",
		"Menu::highlight { padding: 1; }",
	}
	for _, text := range rejected {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("rejected selector parsed: %q", text)
		}
	}
}
