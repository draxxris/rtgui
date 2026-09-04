package transform

import (
	"testing"

	"rtgui/core"
)

func BenchmarkMapping(b *testing.B) {
	t := New(core.Viewport{Viewport: core.Rect{X: 10, Y: 20, W: 800, H: 600}})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.PhysicalToViewport(core.Vec2{X: 100, Y: 100})
		t.PixelSnap = true
		_ = t.Snap(100.6)
		t.PixelSnap = false
	}
}
