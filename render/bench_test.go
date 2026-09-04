package render

import (
	"testing"

	"rtgui/core"
	"rtgui/skin"
	"rtgui/transform"
)

func BenchmarkDrawWidgetPart(b *testing.B) {
	theme := NewTheme(transform.New(core.Viewport{Viewport: core.Rect{W: 800, H: 600}}))
	_ = theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, G: 255, B: 255, A: 255}, Alpha: 1, HasTexture: true,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 100, H: 30}, core.StateNormal)
	}
}

func BenchmarkNinePatch(b *testing.B) {
	source := core.Rect{W: 64, H: 32}
	config := NinePatchConfig{Source: source, Left: 8, Top: 8, Right: 8, Bottom: 8}
	destination := core.Rect{W: 120, H: 40}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NinePatchRects(source, config, destination)
	}
}
