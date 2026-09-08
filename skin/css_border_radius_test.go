package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestParseCSSBorderRadius verifies radius values land on the background entry.
func TestParseCSSBorderRadius(t *testing.T) {
	rules, err := ParseCSS(`Textbox {
		background-image: url("bg.png");
		border-image-source: url("ring.png");
		border-image-slice: 8;
		border-radius: 6px;
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected background+border entries, got %+v", rules)
	}
	var bg, border *SkinRule
	for i := range rules {
		switch rules[i].Part {
		case PartBackground:
			bg = &rules[i]
		case PartBorder:
			border = &rules[i]
		}
	}
	if bg == nil || !bg.HasRadius || bg.Radius != 6 {
		t.Fatalf("background radius = %+v", bg)
	}
	if border == nil || border.HasRadius {
		t.Fatalf("border must not carry a radius, got %+v", border)
	}
}

// TestParseCSSBorderRadiusForms verifies bare, px-suffixed, and zero radii.
func TestParseCSSBorderRadiusForms(t *testing.T) {
	for _, text := range []string{
		`Button { border-radius: 0; }`,
		`Button { border-radius: 8; }`,
		`Button { border-radius: 8px; }`,
		`TabBar::tab { border-radius: 4; }`,
		`Dropdown::popup { border-radius: 4; }`,
	} {
		rules, err := ParseCSS(text)
		if err != nil {
			t.Fatalf("%s: %v", text, err)
		}
		if len(rules) != 1 || !rules[0].HasRadius {
			t.Fatalf("%s parsed as %+v", text, rules)
		}
	}
	if rules, _ := ParseCSS(`Button { border-radius: 0; }`); rules[0].Radius != 0 {
		t.Fatalf("zero radius = %+v", rules[0])
	}
}

// TestParseCSSBorderRadiusErrors verifies negative and non-numeric radii fail.
func TestParseCSSBorderRadiusErrors(t *testing.T) {
	for _, text := range []string{
		`Button { border-radius: -1; }`,
		`Button { border-radius: lots; }`,
		`Button { border-radius: 1 2; }`,
	} {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("expected error for %q", text)
		}
	}
}

// TestSkinRuleRadiusRouting verifies whole-widget radius targets the background.
func TestSkinRuleRadiusRouting(t *testing.T) {
	rules, err := ParseCSS(`Button { border-radius: 5; }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Kind != core.WidgetButton || rules[0].Part != PartBackground || rules[0].Radius != 5 {
		t.Fatalf("radius routing = %+v", rules)
	}
}
