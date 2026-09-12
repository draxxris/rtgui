package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestTableCSSPartsStayDistinct checks the table-specific skin vocabulary.
func TestTableCSSPartsStayDistinct(t *testing.T) {
	rules, err := ParseCSS(`
		Table::header { background-color: #1e2636; color: #e8c878; }
		Table::stripe { background-color: #20242c; }
		Table::arrow { background-image-tint: #c9b896; color: #ffffff; }
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 {
		t.Fatalf("table parts = %d, want 3: %+v", len(rules), rules)
	}
	if rules[0].Kind != core.WidgetTable || rules[0].Part != PartHeader || !rules[0].HasBackgroundColor || !rules[0].HasTextColor {
		t.Fatalf("header rule = %+v", rules[0])
	}
	if rules[1].Kind != core.WidgetTable || rules[1].Part != PartStripe || !rules[1].HasBackgroundColor {
		t.Fatalf("stripe rule = %+v", rules[1])
	}
	if rules[2].Kind != core.WidgetTable || rules[2].Part != PartArrow || !rules[2].HasTint || !rules[2].HasTextColor {
		t.Fatalf("arrow rule = %+v", rules[2])
	}
	if _, err := ParseCSS(`Table::arrow { background-image: url("arrow.png"); }`); err == nil {
		t.Fatal("Table::arrow accepted image art")
	}
}
