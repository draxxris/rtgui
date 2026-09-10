package core

import "testing"

// TestRichTooltipDataEqualClass verifies the shell class joins equality.
func TestRichTooltipDataEqualClass(t *testing.T) {
	base := RichTooltip{Title: "Thunderfury", Class: "item"}
	if !RichTooltipDataEqual(base, RichTooltip{Title: "Thunderfury", Class: "item"}) {
		t.Fatal("matching classes must compare equal")
	}
	if RichTooltipDataEqual(base, RichTooltip{Title: "Thunderfury"}) {
		t.Fatal("class mismatch must compare unequal")
	}
	if RichTooltipDataEqual(base, RichTooltip{Title: "Thunderfury", Class: "other"}) {
		t.Fatal("different classes must compare unequal")
	}
}
