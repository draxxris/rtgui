package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// TestScrollContent verifies skin-aware content area computation for ScrollPanel.
func TestScrollContent(t *testing.T) {
	bounds := core.Rect{X: 10, Y: 20, W: 200, H: 100}

	var nilTheme *Theme
	if got := nilTheme.ScrollContent(bounds, core.StateNormal); got != bounds {
		t.Fatalf("nil theme ScrollContent=%v, want %v", got, bounds)
	}

	theme := NewTheme(nil)
	if got := theme.ScrollContent(bounds, core.StateNormal); got != bounds {
		t.Fatalf("unskinned ScrollContent=%v, want %v", got, bounds)
	}

	// Register border nine-patch with 8px slices and background padding of 10px.
	borderDesc := skin.SkinDescriptor{
		NinePatch:    skin.NinePatch{Left: 8, Top: 8, Right: 8, Bottom: 8},
		HasNinePatch: true,
	}
	bgDesc := skin.SkinDescriptor{
		PaddingLeft:   10,
		PaddingTop:    10,
		PaddingRight:  10,
		PaddingBottom: 10,
	}
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetScrollPanel, Part: skin.PartBorder, State: core.StateNormal}, borderDesc)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetScrollPanel, Part: skin.PartBackground, State: core.StateNormal}, bgDesc)

	// max(8, 10) = 10 on all sides.
	expected := core.Rect{X: 20, Y: 30, W: 180, H: 80}
	if got := theme.ScrollContent(bounds, core.StateNormal); got != expected {
		t.Fatalf("skinned ScrollContent=%v, want %v", got, expected)
	}
}
