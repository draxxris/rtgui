package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

// BenchmarkDrawWidgetPart measures the normal no-recorder draw path.
func BenchmarkDrawWidgetPart(b *testing.B) {
	theme := NewTheme(transform.New(core.Viewport{Viewport: core.Rect{W: 800, H: 600}}))
	theme.SetDrawRecorder(nil)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, G: 255, B: 255, A: 255}, HasTexture: true,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 100, H: 30}, core.StateNormal)
	}
}

// BenchmarkNinePatch measures deterministic nine-patch geometry.
func BenchmarkNinePatch(b *testing.B) {
	config := NinePatchConfig{Left: 8, Top: 8, Right: 8, Bottom: 8}
	destination := core.Rect{W: 120, H: 40}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NinePatchRects(config, destination)
	}
}
