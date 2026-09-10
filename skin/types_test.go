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
		Gradients:          [MaxGradientLayers]Gradient{{Direction: GradientToBottom, StopCount: 2}, {Kind: GradientRadial, CenterX: 0.3, CenterY: 0.2, StopCount: 2}},
		GradientCount:      2,
		BackgroundColor:    core.Color{R: 36, G: 53, B: 76, A: 255},
		HasBackgroundColor: true,
	}
	cleared := base.Overlay(SkinDescriptor{NoTexture: true})
	if cleared.HasTexture || cleared.HasGradient() || !cleared.NoTexture {
		t.Fatalf("cleared overlay = %+v", cleared)
	}
	if !cleared.HasBackgroundColor {
		t.Fatalf("none must keep background color, got %+v", cleared)
	}
	kept := base.Overlay(SkinDescriptor{})
	if !kept.HasTexture || kept.GradientCount != 2 {
		t.Fatalf("empty overlay must keep texture, got %+v", kept)
	}
	replaced := base.Overlay(SkinDescriptor{Gradients: [MaxGradientLayers]Gradient{{Direction: GradientToTop, StopCount: 2}}, GradientCount: 1})
	if replaced.GradientCount != 1 || replaced.Gradients[0].Direction != GradientToTop {
		t.Fatalf("gradient stack must replace wholesale, got %+v", replaced)
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

// TestSkinDescriptorOverlayTreatsBackgroundImageAsUnit verifies a gradient
// layer drops an inherited texture and vice versa, matching CSS where
// background-image is one property.
func TestSkinDescriptorOverlayTreatsBackgroundImageAsUnit(t *testing.T) {
	textured := SkinDescriptor{
		Texture:    Texture{ID: 7, Width: 64, Height: 64},
		HasTexture: true,
		Tint:       core.Color{R: 20, G: 28, B: 46, A: 255},
	}
	layered := SkinDescriptor{
		Gradients:     [MaxGradientLayers]Gradient{{Direction: GradientToBottom, StopCount: 2}},
		GradientCount: 1,
	}
	gradientOverTexture := textured.Overlay(layered)
	if gradientOverTexture.HasTexture || gradientOverTexture.GradientCount != 1 {
		t.Fatalf("gradient must drop inherited texture, got %+v", gradientOverTexture)
	}
	textureOverGradient := layered.Overlay(textured)
	if !textureOverGradient.HasTexture || textureOverGradient.HasGradient() {
		t.Fatalf("texture must drop inherited gradients, got %+v", textureOverGradient)
	}
	plain := SkinDescriptor{BackgroundColor: core.Color{R: 1, G: 2, B: 3, A: 255}, HasBackgroundColor: true}
	kept := textured.Overlay(plain)
	if !kept.HasTexture {
		t.Fatalf("color-only overlay must keep texture, got %+v", kept)
	}
}
