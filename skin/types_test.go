package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestSkinDescriptorOverlayDropsTextureOnNone verifies an explicit none
// clears inherited textures and gradients while keeping other fields.
func TestSkinDescriptorOverlayDropsTextureOnNone(t *testing.T) {
	base := SkinDescriptor{
		Texture:            Texture{ID: 7, Width: 64, Height: 64},
		HasTexture:         true,
		Gradient:           LinearGradient{Direction: GradientToBottom},
		HasGradient:        true,
		BackgroundColor:    core.Color{R: 36, G: 53, B: 76, A: 255},
		HasBackgroundColor: true,
	}
	cleared := base.Overlay(SkinDescriptor{NoTexture: true})
	if cleared.HasTexture || cleared.HasGradient || !cleared.NoTexture {
		t.Fatalf("cleared overlay = %+v", cleared)
	}
	if !cleared.HasBackgroundColor {
		t.Fatalf("none must keep background color, got %+v", cleared)
	}
	kept := base.Overlay(SkinDescriptor{})
	if !kept.HasTexture || !kept.HasGradient {
		t.Fatalf("empty overlay must keep texture, got %+v", kept)
	}
}

// TestSkinDescriptorOverlayCarriesRadius verifies radius survives overlays.
func TestSkinDescriptorOverlayCarriesRadius(t *testing.T) {
	base := SkinDescriptor{Radius: 6, HasRadius: true}
	overlay := SkinDescriptor{}.Overlay(SkinDescriptor{Radius: 6, HasRadius: true})
	if !overlay.HasRadius || overlay.Radius != 6 {
		t.Fatalf("overlay radius = %+v", overlay)
	}
	kept := base.Overlay(SkinDescriptor{})
	if !kept.HasRadius || kept.Radius != 6 {
		t.Fatalf("empty overlay must keep radius, got %+v", kept)
	}
}
