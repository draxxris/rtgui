package render

import (
	"math"
	"testing"

	"rtgui/core"
	"rtgui/skin"
	"rtgui/transform"
)

func approxEqual(a, b float32) bool {
	return a == b || math.Abs(float64(a-b)) < 1e-4
}

func TestNinePatchGeometry(t *testing.T) {
	source := core.Rect{W: 64, H: 32}
	config := NinePatchConfig{Source: source, Left: 8, Top: 8, Right: 8, Bottom: 8}
	for _, destination := range []core.Rect{
		{W: 10, H: 10},
		{W: 16, H: 16},
		{W: 64, H: 32},
		{X: 10, Y: 20, W: 200, H: 100},
		{W: -10, H: -5},
	} {
		rects := NinePatchRects(source, config, destination)
		expectedWidth, expectedHeight := destination.W, destination.H
		if expectedWidth < 0 {
			expectedWidth = 0
		}
		if expectedHeight < 0 {
			expectedHeight = 0
		}
		width := rects[0].W + rects[1].W + rects[2].W
		height := rects[0].H + rects[3].H + rects[6].H
		if !approxEqual(width, expectedWidth) || !approxEqual(height, expectedHeight) {
			t.Fatalf("destination=%v sums=%v,%v", destination, width, height)
		}
		for i, rect := range rects {
			if rect.W < 0 || rect.H < 0 {
				t.Fatalf("rect %d has negative size: %v", i, rect)
			}
		}
	}
	if rects := NinePatchRects(source, config, core.Rect{W: 10, H: 10}); !approxEqual(rects[0].W, 5) || !approxEqual(rects[0].H, 5) {
		t.Fatalf("tiny destination did not scale borders: %v", rects[0])
	}

	sourceRects := NinePatchSourceRects(source, skin.NinePatch{Left: 8, Top: 8, Right: 8, Bottom: 8})
	if !approxEqual(sourceRects[4].W, 48) || !approxEqual(sourceRects[4].H, 16) {
		t.Fatalf("source center=%v", sourceRects[4])
	}
}

func TestContentRect(t *testing.T) {
	bounds := core.Rect{X: 10, Y: 20, W: 100, H: 60}
	descriptor := skin.SkinDescriptor{
		HasNinePatch: true,
		NinePatch:    skin.NinePatch{Left: 8, Top: 6, Right: 10, Bottom: 7},
		PaddingLeft:  2, PaddingTop: 12, PaddingRight: 4, PaddingBottom: 3,
	}
	want := core.Rect{X: 18, Y: 32, W: 82, H: 41}
	if got := ContentRect(bounds, descriptor); got != want {
		t.Fatalf("content=%v want=%v", got, want)
	}
	descriptor.PaddingLeft = 20
	if got := ContentRect(bounds, descriptor); got.X != 30 || got.W != 70 {
		t.Fatalf("explicit padding did not win: %v", got)
	}
	if got := ContentRect(core.Rect{W: -1, H: -1}, descriptor); got.W != 0 || got.H != 0 {
		t.Fatalf("negative bounds=%v", got)
	}
}

func TestThemeHeadlessAndIsolation(t *testing.T) {
	viewport := core.Viewport{Viewport: core.Rect{W: 800, H: 600}, LogicalSize: core.Vec2{X: 800, Y: 600}}
	theme := NewTheme(transform.New(viewport))
	theme.ClearDrawLog()
	bounds := core.Rect{X: 10, Y: 10, W: 100, H: 40}
	if err := theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, bounds, core.StateNormal); err != nil {
		t.Fatal(err)
	}
	calls := theme.DrawLog()
	if len(calls) != 1 || !calls[0].Fallback || calls[0].Dest != bounds {
		t.Fatalf("fallback calls=%v", calls)
	}

	if err := theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, A: 255}, Alpha: 1, HasTexture: true,
	}); err != nil {
		t.Fatal(err)
	}
	theme.SetPixelSnap(true)
	theme.ClearDrawLog()
	_ = theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{X: 10.2, Y: 10.7, W: 20.4, H: 20.4}, core.StateHovered)
	calls = theme.DrawLog()
	if len(calls) != 1 || calls[0].Fallback || calls[0].Dest != (core.Rect{X: 10, Y: 11, W: 20, H: 20}) {
		t.Fatalf("fallback or snapping calls=%v", calls)
	}

	other := NewTheme(transform.New(viewport))
	other.ClearDrawLog()
	_ = other.DrawWidgetPart(core.WidgetButton, skin.PartBackground, bounds, core.StateNormal)
	if calls := other.DrawLog(); len(calls) != 1 || !calls[0].Fallback {
		t.Fatalf("themes share skin state: %v", calls)
	}
}

func TestThemeWidgetVariants(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{Viewport: core.Rect{W: 800, H: 600}}))
	theme.ClearDrawLog()
	button := core.WidgetInfo{ID: 1, Name: "okButton", Bounds: core.Rect{W: 100, H: 30}, Kind: core.WidgetButton, State: core.StateNormal}
	if err := theme.DrawWidget(button, "OK", 0, false); err != nil {
		t.Fatal(err)
	}
	if len(theme.DrawLog()) < 2 {
		t.Fatalf("button draw log=%v", theme.DrawLog())
	}

	theme.ClearDrawLog()
	slider := core.WidgetInfo{Name: "volumeSlider", Bounds: core.Rect{W: 100, H: 20}, Kind: core.WidgetSlider, State: core.StateNormal}
	_ = theme.DrawWidget(slider, "", 0.5, false)
	if calls := theme.DrawLog(); len(calls) != 3 || calls[1].Part != skin.PartTrack || calls[2].Part != skin.PartThumb {
		t.Fatalf("slider draw log=%v", calls)
	}

	theme.ClearDrawLog()
	progress := core.WidgetInfo{Name: "loadProgress", Bounds: core.Rect{W: 100, H: 20}, Kind: core.WidgetProgressBar, State: core.StateNormal}
	_ = theme.DrawWidget(progress, "", 0.3, false)
	if calls := theme.DrawLog(); len(calls) != 3 || calls[2].Part != skin.PartOverlay {
		t.Fatalf("progress draw log=%v", calls)
	}
}
