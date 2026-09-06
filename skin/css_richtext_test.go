package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestParseRichTextSelectors verifies the message kind and part vocabulary.
func TestParseRichTextSelectors(t *testing.T) {
	rules, err := ParseCSS("RichText { padding: 6; }")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Kind != core.WidgetRichText {
		t.Fatalf("RichText parsed as %+v", rules)
	}
	if _, err := ParseCSS("RichText::highlight { background-image: url(\"a.png\"); }"); err != nil {
		t.Fatalf("RichText::highlight: %v", err)
	}
	rejected := []string{
		"RichText::tab { background-image: url(\"a.png\"); }",
		"RichText::link { background-image: url(\"a.png\"); }",
		"RichText::highlight { padding: 1; }",
	}
	for _, text := range rejected {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("rejected selector parsed: %q", text)
		}
	}
}
