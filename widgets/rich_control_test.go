package widgets_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// TestBaseRichTextFallback checks rich wins for Text with plain fallback.
func TestBaseRichTextFallback(t *testing.T) {
	button := widgets.NewButton("ok", core.Rect{}, "OK")
	if button.HasRichText() {
		t.Fatal("new button has rich text")
	}
	if !button.SetRichSegments([]core.RichSegment{{Icon: "iron-plate", HasIcon: true}, {Text: " OK"}}) {
		t.Fatal("rich set not reported")
	}
	if !button.HasRichText() || button.Text() != " OK" {
		t.Fatalf("rich fallback text = %q", button.Text())
	}
	if button.SetRichSegments([]core.RichSegment{{Icon: "iron-plate", HasIcon: true}, {Text: " OK"}}) {
		t.Fatal("identical rich reported a change")
	}
	if !button.SetText("Plain") || button.HasRichText() || button.Text() != "Plain" {
		t.Fatalf("SetText did not clear rich: %q/%v", button.Text(), button.HasRichText())
	}
}

// TestDropdownRichRows checks per-row rich overlay with plain fallback.
func TestDropdownRichRows(t *testing.T) {
	dropdown := widgets.NewDropdown("class", core.Rect{}, []string{"Warrior", "Mage"}, 0)
	if dropdown.HasRichDropdownItems() {
		t.Fatal("new dropdown has rich rows")
	}
	if !dropdown.SetRichDropdownItem(0, []core.RichSegment{{Icon: "iron-plate", HasIcon: true}, {Text: " Warrior"}}) {
		t.Fatal("rich row set not reported")
	}
	segments, ok := dropdown.RichDropdownItem(0)
	if !ok || len(segments) != 2 || !segments[0].HasIcon {
		t.Fatalf("rich row = %+v/%v", segments, ok)
	}
	if _, ok := dropdown.RichDropdownItem(1); ok {
		t.Fatal("plain row reported rich")
	}
	if dropdown.SetRichDropdownItem(9, []core.RichSegment{{Text: "x"}}) {
		t.Fatal("out-of-range rich row accepted")
	}
	if !dropdown.ClearRichDropdownItems() || dropdown.HasRichDropdownItems() {
		t.Fatal("rich rows not cleared")
	}
}

// TestDropdownRichSingleSource pins rows as the single rich authority.
// Replacing items resets row and base runs so no stale display survives.
func TestDropdownRichSingleSource(t *testing.T) {
	dropdown := widgets.NewDropdown("class", core.Rect{}, []string{"Warrior", "Mage"}, 0)
	dropdown.SetRichSegments([]core.RichSegment{{Text: "stale"}})
	dropdown.SetRichDropdownItem(1, []core.RichSegment{{Text: "Mage!"}})
	if !dropdown.HasRichText() || !dropdown.HasRichDropdownItems() {
		t.Fatal("rich state not set")
	}
	if !dropdown.SetDropdownItems([]string{"Rogue"}) {
		t.Fatal("item replacement not reported")
	}
	if dropdown.HasRichDropdownItems() || dropdown.HasRichText() {
		t.Fatal("stale rich state survived item replacement")
	}
	if _, ok := dropdown.RichDropdownItem(0); ok {
		t.Fatal("stale row survived item replacement")
	}
}
