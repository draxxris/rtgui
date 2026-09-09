package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestParseListSelectors verifies the List kind and its allowed parts.
func TestParseListSelectors(t *testing.T) {
	rules, err := ParseCSS("List { padding: 1; }")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 1 || rules[0].Kind != core.WidgetList {
		t.Fatalf("List parsed as %+v", rules)
	}
	allowed := []string{
		"List::highlight { background-image: url(\"a.png\"); }",
		"List::highlight:selected { background-image: url(\"a.png\"); }",
		"List::track { border-image-source: url(\"a.png\"); }",
		"List::thumb { background-image: url(\"a.png\"); }",
		"List.category { padding: 2; }",
		"List.category::highlight { background-image: url(\"a.png\"); }",
	}
	for _, text := range allowed {
		if _, err := ParseCSS(text); err != nil {
			t.Fatalf("allowed selector rejected %q: %v", text, err)
		}
	}
	// Chevrons are geometry-only like textbox carets, so List::arrow fails
	// loudly instead of authoring an ignored texture.
	rejected := []string{
		"List::tab { background-image: url(\"a.png\"); }",
		"List::popup { background-image: url(\"a.png\"); }",
		"List::arrow { background-image: url(\"a.png\"); }",
		"List::highlight { padding: 1; }",
		"List::highlight { border-image-source: url(\"a.png\"); }",
	}
	for _, text := range rejected {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("rejected selector parsed: %q", text)
		}
	}
}
