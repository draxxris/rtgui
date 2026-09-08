package skin

import (
	"testing"
)

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
