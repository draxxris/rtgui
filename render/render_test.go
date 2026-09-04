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

// TestThemeFontsHeadless verifies font state without a window: recording a
// present file succeeds with no GL, missing files and nil receivers fail,
// size-matched lookups fall back to the default font, and unload is safe.
func TestThemeFontsHeadless(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	if theme.HasFont() || theme.HasItalicFont() {
		t.Fatal("fresh theme must report no fonts")
	}
	if err := theme.LoadFont("../testdata/fonts/Grenze-Light.ttf"); err != nil {
		t.Fatalf("LoadFont must record a present file without a window: %v", err)
	}
	if err := theme.LoadItalicFont("../testdata/fonts/Grenze-LightItalic.ttf"); err != nil {
		t.Fatalf("LoadItalicFont must record a present file without a window: %v", err)
	}
	if !theme.HasFont() || !theme.HasItalicFont() {
		t.Fatal("recorded fonts must report present")
	}
	if err := theme.LoadFont("testdata/fonts/does-not-exist.ttf"); err == nil {
		t.Fatal("LoadFont of a missing file must fail")
	}
	if err := theme.LoadItalicFont("testdata/fonts/does-not-exist.ttf"); err == nil {
		t.Fatal("LoadItalicFont of a missing file must fail")
	}
	// No window is ready in tests, so rasterization falls back to default.
	_ = theme.FontForSize(16)
	_ = theme.ItalicForSize(14)
	_ = theme.Font()
	_ = theme.ItalicFont()
	theme.UnloadFonts() // must not panic with nothing rasterized
	if theme.HasFont() || theme.HasItalicFont() {
		t.Fatal("UnloadFonts must forget recorded paths")
	}
	var nilTheme *Theme
	if nilTheme.HasFont() || nilTheme.HasItalicFont() {
		t.Fatal("nil theme must report no fonts")
	}
	nilTheme.UnloadFonts() // must not panic
	if err := nilTheme.LoadFont("x.ttf"); err == nil {
		t.Fatal("nil theme LoadFont must fail")
	}
	_ = nilTheme.FontForSize(16)
}

// TestCheckboxIconParts verifies the Kenney-style checkbox art: a textured
// PartIcon renders the empty box when unchecked, a textured PartCheckmark the
// cross when checked, and missing skins keep the old behavior (nothing when
// unchecked, geometry check when checked).
func TestCheckboxIconParts(t *testing.T) {
	box := core.Rect{X: 10, Y: 10, W: 120, H: 40}
	theme := NewTheme(transform.New(core.Viewport{}))

	unchecked := core.WidgetInfo{Name: "agreeBox", Bounds: box, Kind: core.WidgetCheckbox, State: core.StateNormal}
	_ = theme.DrawWidget(unchecked, "", 0, false)
	for _, call := range theme.DrawLog() {
		if call.Part == skin.PartIcon || call.Part == skin.PartCheckmark {
			t.Fatalf("no-skin unchecked box must draw no icon, got %v", call.Part)
		}
	}

	icon := skin.SkinDescriptor{Texture: skin.Texture{ID: 7, Width: 32, Height: 32}, AtlasRegion: core.Rect{W: 32, H: 32}, Tint: core.Color{A: 255}, Alpha: 1, HasTexture: true}
	cross := skin.SkinDescriptor{Texture: skin.Texture{ID: 8, Width: 32, Height: 32}, AtlasRegion: core.Rect{W: 32, H: 32}, Tint: core.Color{A: 255}, Alpha: 1, HasTexture: true}
	key := skin.SkinKey{Widget: core.WidgetCheckbox, Part: skin.PartIcon, State: core.StateNormal}
	if err := theme.SetSkinPart(key, icon); err != nil {
		t.Fatal(err)
	}
	key = skin.SkinKey{Widget: core.WidgetCheckbox, Part: skin.PartCheckmark, State: core.StateNormal}
	if err := theme.SetSkinPart(key, cross); err != nil {
		t.Fatal(err)
	}

	theme.ClearDrawLog()
	_ = theme.DrawWidget(unchecked, "", 0, false)
	foundIcon := false
	for _, call := range theme.DrawLog() {
		if call.Part == skin.PartIcon {
			foundIcon = true
		}
		if call.Part == skin.PartCheckmark {
			t.Fatal("unchecked box must not draw the checkmark")
		}
	}
	if !foundIcon {
		t.Fatal("unchecked box must draw the empty-box icon")
	}

	theme.ClearDrawLog()
	checked := core.WidgetInfo{Name: "agreeBox", Bounds: box, Kind: core.WidgetCheckbox, State: core.StateNormal}
	_ = theme.DrawWidget(checked, "", 0, true)
	foundCheck := false
	for _, call := range theme.DrawLog() {
		if call.Part == skin.PartCheckmark {
			foundCheck = true
		}
		if call.Part == skin.PartIcon {
			t.Fatal("checked box must not draw the empty-box icon")
		}
	}
	if !foundCheck {
		t.Fatal("checked box must draw the checkmark")
	}
}

// TestEightPatchGeometry verifies border-only textures (opaque edges, empty
// center, e.g. the Kenney 64x64 grey panel ring with 8px edges): the edge
// rects keep full border thickness, stretch along their axis, and index 4 is
// the only patch skipped when CenterFill is false.
func TestEightPatchGeometry(t *testing.T) {
	patch := skin.NinePatch{Left: 8, Top: 8, Right: 8, Bottom: 8}
	src := core.Rect{W: 64, H: 64}
	source := NinePatchSourceRects(src, patch)
	if source[0] != (core.Rect{X: 0, Y: 0, W: 8, H: 8}) {
		t.Fatalf("top-left source=%v", source[0])
	}
	if source[4] != (core.Rect{X: 8, Y: 8, W: 48, H: 48}) {
		t.Fatalf("center source=%v", source[4])
	}
	if source[8] != (core.Rect{X: 56, Y: 56, W: 8, H: 8}) {
		t.Fatalf("bottom-right source=%v", source[8])
	}

	dest := NinePatchRects(src, NinePatchConfig{Source: src, Left: 8, Top: 8, Right: 8, Bottom: 8}, core.Rect{X: 10, Y: 20, W: 200, H: 94})
	if dest[0] != (core.Rect{X: 10, Y: 20, W: 8, H: 8}) {
		t.Fatalf("top-left dest=%v", dest[0])
	}
	if dest[1] != (core.Rect{X: 18, Y: 20, W: 184, H: 8}) {
		t.Fatalf("top edge dest=%v", dest[1])
	}
	if dest[3] != (core.Rect{X: 10, Y: 28, W: 8, H: 78}) {
		t.Fatalf("left edge dest=%v", dest[3])
	}
	if dest[4] != (core.Rect{X: 18, Y: 28, W: 184, H: 78}) {
		t.Fatalf("center dest=%v", dest[4])
	}

	skipped := 0
	for i := 0; i < 9; i++ {
		if skipCenter(false, i) {
			skipped++
			if i != 4 {
				t.Fatalf("only index 4 may skip, got %d", i)
			}
		}
		if skipCenter(true, i) {
			t.Fatalf("CenterFill must draw patch %d", i)
		}
	}
	if skipped != 1 {
		t.Fatalf("8-patch must skip exactly one patch, skipped %d", skipped)
	}
}
