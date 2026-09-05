package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestRegistryStateFallback checks exact, fallback, replace, and clear operations.
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
	replacement := NewRegistry()
	replacement.Set(SkinKey{Widget: core.WidgetButton, Part: PartBackground, State: core.StateHovered}, SkinDescriptor{Tint: core.Color{G: 9}})
	registry.Replace(replacement)
	if _, ok := registry.Get(key); ok {
		t.Fatal("whole-map replacement retained an old exact key")
	}
	if got, ok := registry.Get(SkinKey{Widget: core.WidgetButton, Part: PartBackground, State: core.StateHovered}); !ok || got.Tint.G != 9 {
		t.Fatalf("replacement exact lookup = %+v/%v", got, ok)
	}
	replacement.Clear()
	if _, ok := registry.Get(SkinKey{Widget: core.WidgetButton, Part: PartBackground, State: core.StateHovered}); !ok {
		t.Fatal("registry replacement did not copy the source map")
	}
	registry.Clear()
	if _, ok := registry.Lookup(core.WidgetButton, PartBackground, core.StateNormal); ok {
		t.Fatal("cleared registry still returned a descriptor")
	}
}
