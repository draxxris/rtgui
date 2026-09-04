package skin

import (
	"testing"

	"rtgui/core"
)

func TestRegistryStateFallback(t *testing.T) {
	registry := NewRegistry()
	key := SkinKey{Widget: core.WidgetButton, Part: PartBackground, State: core.StateNormal}
	descriptor := SkinDescriptor{AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, A: 255}}
	registry.Set(key, descriptor)
	if got, ok := registry.Get(key); !ok || got.AtlasRegion.W != 64 {
		t.Fatalf("exact lookup: %+v %v", got, ok)
	}
	if got, ok := registry.Lookup(core.WidgetButton, PartBackground, core.StateHovered); !ok || got != descriptor {
		t.Fatalf("normal-state fallback: %+v %v", got, ok)
	}
	registry.Clear()
	if _, ok := registry.Lookup(core.WidgetButton, PartBackground, core.StateNormal); ok {
		t.Fatal("cleared registry still returned a descriptor")
	}
}
