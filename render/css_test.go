package render

import (
	"testing"

	"rtgui/core"
	"rtgui/skin"
	"rtgui/transform"
)

// TestMergeSkinRules verifies later-wins-per-field and state-over-normal
// inheritance, including first-appearance key order.
func TestMergeSkinRules(t *testing.T) {
	rules := []skin.SkinRule{
		{Kind: core.WidgetButton, Part: skin.PartBorder, State: core.StateNormal,
			Image: "line.png", HasImage: true, Slice: 8, HasSlice: true,
			Padding: [4]float32{8, 8, 8, 8}, HasPadding: true},
		{Kind: core.WidgetButton, Part: skin.PartBorder, State: core.StateHovered,
			Image: "hover.png", HasImage: true},
		{Kind: core.WidgetButton, Part: skin.PartBorder, State: core.StateDisabled,
			Tint: core.Color{R: 185, G: 185, B: 195, A: 255}, HasTint: true},
	}
	merged, order := mergeSkinRules(rules)
	if len(order) != 3 {
		t.Fatalf("order=%v", order)
	}
	hover := merged[skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateHovered}]
	if hover.image != "hover.png" || !hover.hasSlice || hover.slice != 8 || !hover.hasPadding {
		t.Fatalf("hover must inherit slice+padding: %+v", hover)
	}
	disabled := merged[skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateDisabled}]
	if !disabled.hasImage || disabled.image != "line.png" || !disabled.hasTint {
		t.Fatalf("disabled must inherit image and keep tint: %+v", disabled)
	}
}

// TestLoadCSSFileHeadless verifies the loud early error without a window.
func TestLoadCSSFileHeadless(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	if err := theme.LoadCSSFile("nonexistent.css", ""); err == nil {
		t.Fatal("headless LoadCSSFile must fail without a window")
	}
	var nilTheme *Theme
	if err := nilTheme.LoadCSSFile("x.css", ""); err == nil {
		t.Fatal("nil theme must fail")
	}
}

// TestLoadRuleImages verifies missing files, tint-without-image, and real
// PNG decode using checked-in Kenney art (decode only, no GL upload).
func TestLoadRuleImages(t *testing.T) {
	base := "../testdata/skins"
	missing := map[skin.SkinKey]mergedRule{
		{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateNormal}: {
			image: "does-not-exist.png", hasImage: true,
		},
	}
	if _, err := loadRuleImages(base, missing, []skin.SkinKey{{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateNormal}}); err == nil {
		t.Fatal("missing image must fail")
	}
	bare := map[skin.SkinKey]mergedRule{
		{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateNormal}: {
			tint: core.Color{A: 255}, hasTint: true,
		},
	}
	if _, err := loadRuleImages(base, bare, []skin.SkinKey{{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateNormal}}); err == nil {
		t.Fatal("tint without image must fail")
	}
	real := map[skin.SkinKey]mergedRule{
		{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateNormal}: {
			image: "kenney/blue/button_rectangle_line.png", hasImage: true,
		},
		{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateHovered}: {
			image: "kenney/blue/check_square_grey.png", hasImage: true,
			tint: core.Color{R: 255, G: 255, B: 255, A: 255}, hasTint: true,
		},
	}
	keys := []skin.SkinKey{
		{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateNormal},
		{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateHovered},
	}
	images, err := loadRuleImages(base, real, keys)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 decoded images, got %d", len(images))
	}
}
