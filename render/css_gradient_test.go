package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

// gradientTestStops builds two edge-to-edge stops from channel values.
func gradientTestStops(a, b uint8) [skin.MaxGradientStops]skin.ColorStop {
	return [skin.MaxGradientStops]skin.ColorStop{
		{Color: core.Color{R: a, G: a, B: a, A: 255}, Position: 0},
		{Color: core.Color{R: b, G: b, B: b, A: 255}, Position: 1},
	}
}

// gradientTestRule builds a rule carrying the given gradient stack.
func gradientTestRule(layers ...skin.LinearGradient) skin.SkinRule {
	rule := skin.SkinRule{Kind: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateNormal}
	for i, layer := range layers {
		rule.Gradients[i] = layer
	}
	rule.GradientCount = len(layers)
	return rule
}

// TestCSSGradientStackMergesByState verifies layered fills merge by state.
func TestCSSGradientStackMergesByState(t *testing.T) {
	base := gradientTestRule(
		skin.LinearGradient{Kind: skin.GradientRadial, CenterX: 0.3, CenterY: 0.2, Stops: gradientTestStops(30, 40), StopCount: 2},
		skin.LinearGradient{Direction: skin.GradientToBottom, Stops: gradientTestStops(10, 20), StopCount: 2},
	)
	hovered := skin.SkinRule{Kind: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateHovered,
		BackgroundColor: core.Color{R: 1, G: 2, B: 3, A: 255}, HasBackgroundColor: true}
	pressed := gradientTestRule(
		skin.LinearGradient{Direction: skin.GradientToTop, Stops: gradientTestStops(50, 60), StopCount: 2},
	)
	pressed.State = core.StatePressed
	merged, _ := mergeSkinRules([]skin.SkinRule{base, hovered, pressed})
	hover := merged[skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateHovered}]
	if hover.gradientCount != 2 {
		t.Fatalf("hover must inherit both layers, got %+v", hover)
	}
	if hover.gradients[0].CenterX != 0.3 {
		t.Fatalf("hover top center = %+v", hover.gradients[0])
	}
	got := merged[skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StatePressed}]
	if got.gradientCount != 1 || got.gradients[0].Direction != skin.GradientToTop {
		t.Fatalf("single layer must replace the stack, got %+v", got)
	}
}

// TestCSSMixedTextureStackReplacesInheritedStackAsUnit verifies a mixed
// normal background is inherited together, while explicit URL-only and
// gradient-only states retain the existing replacement behavior.
func TestCSSMixedTextureStackReplacesInheritedStackAsUnit(t *testing.T) {
	base := gradientTestRule(
		skin.LinearGradient{Kind: skin.GradientRadial, Stops: gradientTestStops(30, 40), StopCount: 2},
		skin.LinearGradient{Direction: skin.GradientToBottom, Stops: gradientTestStops(10, 20), StopCount: 2},
	)
	base.Image, base.HasImage = "surface.png", true
	hovered := skin.SkinRule{Kind: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateHovered,
		BackgroundColor: core.Color{R: 1, G: 2, B: 3, A: 255}, HasBackgroundColor: true}
	urlOnly := skin.SkinRule{Kind: core.WidgetTooltip, Part: skin.PartBackground, State: core.StatePressed,
		Image: "pressed.png", HasImage: true}
	gradientOnly := gradientTestRule(skin.LinearGradient{Direction: skin.GradientToTop, Stops: gradientTestStops(50, 60), StopCount: 2})
	gradientOnly.State = core.StateFocused
	merged, _ := mergeSkinRules([]skin.SkinRule{base, hovered, urlOnly, gradientOnly})
	hover := merged[skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateHovered}]
	if !hover.hasImage || hover.image != "surface.png" || hover.gradientCount != 2 {
		t.Fatalf("mixed stack inheritance = %+v", hover)
	}
	pressed := merged[skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StatePressed}]
	if !pressed.hasImage || pressed.image != "pressed.png" || pressed.gradientCount != 0 {
		t.Fatalf("URL-only replacement = %+v", pressed)
	}
	focused := merged[skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateFocused}]
	if focused.hasImage || focused.gradientCount != 1 || focused.gradients[0].Direction != skin.GradientToTop {
		t.Fatalf("gradient-only replacement = %+v", focused)
	}
}

// TestCSSGradientNoneClearsStack verifies none drops every layer.
func TestCSSGradientNoneClearsStack(t *testing.T) {
	rules, err := skin.ParseCSS(`
		Tooltip {
			background-image: radial-gradient(circle at 30% 20%, #3a5a7a80, #00000000 60%), linear-gradient(to bottom, #2b3d54, #0b1524);
		}
		Tooltip.item {
			background-image: none;
			background-color: #24354c;
		}
	`)
	if err != nil {
		t.Fatal(err)
	}
	merged, _ := mergeSkinRules(rules)
	key := skin.SkinKey{Widget: core.WidgetTooltip, Class: "item", Part: skin.PartBackground, State: core.StateNormal}
	entry := merged[key]
	if !entry.noTexture || entry.gradientCount != 0 {
		t.Fatalf("none must clear the stack, got %+v", entry)
	}
	base := merged[skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateNormal}]
	if base.gradientCount != 2 {
		t.Fatalf("base must hold both layers, got %+v", base)
	}
	theme := newFakeTheme(&fakeTextureBackend{isReady: false})
	desc, err := theme.buildCSSDescriptor(key, entry, ".", map[string]skin.Texture{})
	if err != nil {
		t.Fatal(err)
	}
	if desc.HasGradient() || !desc.HasBackgroundColor {
		t.Fatalf("descriptor must drop layers but keep color, got %+v", desc)
	}
}

// TestDrawGradientStackAllocatesNothing verifies fast-path layers allocate nothing.
func TestDrawGradientStackAllocatesNothing(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.SetDrawRecorder(nil)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		Gradients: [skin.MaxGradientLayers]skin.LinearGradient{
			{Direction: skin.GradientToBottom, Stops: gradientTestStops(20, 40), StopCount: 2},
			{Direction: skin.GradientToRight, Stops: gradientTestStops(60, 80), StopCount: 2},
		},
		GradientCount: 2,
	})
	allocations := testing.AllocsPerRun(100, func() {
		theme.DrawWidgetPart(core.WidgetTooltip, skin.PartBackground, core.Rect{W: 100, H: 30}, core.StateNormal)
	})
	if allocations != 0 {
		t.Fatalf("gradient stack allocations = %v, want 0", allocations)
	}
}

// TestLookupClassedGradientDropsBaseTexture verifies the item tooltip shell
// carries its gradient stack without the base textured background. A prior
// revision composited both layers and the texture buried the gradients.
func TestLookupClassedGradientDropsBaseTexture(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		Texture: skin.Texture{ID: 7, Width: 64, Height: 64}, HasTexture: true,
		Tint: core.Color{R: 20, G: 28, B: 46, A: 255},
	})
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTooltip, Class: "item", Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		Gradients: [skin.MaxGradientLayers]skin.LinearGradient{
			{Kind: skin.GradientRadial, CenterX: 0.3, CenterY: 0.2, Stops: gradientTestStops(30, 40), StopCount: 2},
			{Direction: skin.GradientToBottom, Stops: gradientTestStops(10, 20), StopCount: 2},
		},
		GradientCount: 2,
	})
	resolved, ok := theme.Lookup(core.WidgetTooltip, skin.PartBackground, core.StateNormal, "item")
	if !ok || resolved.GradientCount != 2 || resolved.HasTexture {
		t.Fatalf("classed lookup must hold only the stack, got ok=%v %+v", ok, resolved)
	}
}
