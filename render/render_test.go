package render

import (
	"math"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

func approxEqual(a, b float32) bool {
	return a == b || math.Abs(float64(a-b)) < 1e-4
}

func newTestRecorder(t *testing.T, maxCalls int) *DrawRecorder {
	t.Helper()
	recorder, err := NewDrawRecorder(maxCalls)
	if err != nil {
		t.Fatalf("NewDrawRecorder(%d): %v", maxCalls, err)
	}
	return recorder
}

// TestNinePatchGeometry preserves the deterministic destination geometry.
func TestNinePatchGeometry(t *testing.T) {
	config := NinePatchConfig{Left: 8, Top: 8, Right: 8, Bottom: 8}
	for _, destination := range []core.Rect{
		{W: 10, H: 10},
		{W: 16, H: 16},
		{W: 64, H: 32},
		{X: 10, Y: 20, W: 200, H: 100},
		{W: -10, H: -5},
	} {
		rects := NinePatchRects(config, destination)
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
	if rects := NinePatchRects(config, core.Rect{W: 10, H: 10}); !approxEqual(rects[0].W, 5) || !approxEqual(rects[0].H, 5) {
		t.Fatalf("tiny destination did not scale borders: %v", rects[0])
	}

	source := core.Rect{W: 64, H: 32}
	sourceRects := NinePatchSourceRects(source, skin.NinePatch{Left: 8, Top: 8, Right: 8, Bottom: 8})
	if !approxEqual(sourceRects[4].W, 48) || !approxEqual(sourceRects[4].H, 16) {
		t.Fatalf("source center=%v", sourceRects[4])
	}
}

// TestThreePatchGeometry verifies top-middle-bottom 3-patch geometry calculations.
func TestThreePatchGeometry(t *testing.T) {
	config := ThreePatchConfig{Top: 8, Bottom: 8}
	for _, destination := range []core.Rect{
		{W: 16, H: 10},
		{W: 16, H: 16},
		{W: 16, H: 64},
		{X: 10, Y: 20, W: 16, H: 100},
		{W: -10, H: -5},
	} {
		rects := ThreePatchRects(config, destination)
		expectedHeight := destination.H
		if expectedHeight < 0 {
			expectedHeight = 0
		}
		height := rects[0].H + rects[1].H + rects[2].H
		if !approxEqual(height, expectedHeight) {
			t.Fatalf("destination=%v height sum=%v expected=%v", destination, height, expectedHeight)
		}
		for i, rect := range rects {
			if rect.W < 0 || rect.H < 0 {
				t.Fatalf("rect %d has negative size: %v", i, rect)
			}
		}
	}

	source := core.Rect{W: 16, H: 64}
	sourceRects := ThreePatchSourceRects(source, skin.ThreePatch{Top: 8, Bottom: 8})
	if !approxEqual(sourceRects[0].H, 8) || !approxEqual(sourceRects[1].H, 48) || !approxEqual(sourceRects[2].H, 8) {
		t.Fatalf("source caps mismatch: top=%v mid=%v bot=%v", sourceRects[0], sourceRects[1], sourceRects[2])
	}
}

// TestContentRect covers border and padding content insets.
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
	background := skin.SkinDescriptor{PaddingLeft: 2, PaddingTop: 12, PaddingRight: 4, PaddingBottom: 3}
	border := skin.SkinDescriptor{HasNinePatch: true, NinePatch: skin.NinePatch{Left: 8, Top: 6, Right: 10, Bottom: 7}}
	if got := ContentRect(bounds, background, border); got != want {
		t.Fatalf("merged content=%v want=%v", got, want)
	}
	if got := ContentRect(bounds); got != bounds {
		t.Fatalf("empty content=%v want=%v", got, bounds)
	}
}

// TestThemeDoesNotRecordWithoutRecorder verifies diagnostics are opt-in.
func TestThemeDoesNotRecordWithoutRecorder(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 100, H: 40}, core.StateNormal)
	if theme.recorder != nil {
		t.Fatal("theme without SetDrawRecorder retained a recorder")
	}
	recorder := newTestRecorder(t, 4)
	theme.SetDrawRecorder(recorder)
	if calls := recorder.Calls(); len(calls) != 0 {
		t.Fatalf("attaching a recorder retained old calls: %v", calls)
	}
}

// TestNewDrawRecorderRejectsNonpositiveLimits validates recorder construction.
func TestNewDrawRecorderRejectsNonpositiveLimits(t *testing.T) {
	for _, limit := range []int{0, -1} {
		recorder, err := NewDrawRecorder(limit)
		if recorder != nil || err == nil {
			t.Fatalf("NewDrawRecorder(%d) = %v, %v", limit, recorder, err)
		}
	}
}

// TestDrawRecorderBounds verifies fixed capacity and truncation reporting.
func TestDrawRecorderBounds(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder := newTestRecorder(t, 2)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	for i := 0; i < 3; i++ {
		theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{X: float32(i), W: 10, H: 10}, core.StateNormal)
	}
	calls := recorder.Calls()
	if len(calls) != 2 || cap(recorder.calls) != 2 || !recorder.Truncated() {
		t.Fatalf("bounded calls=%v cap=%d truncated=%v", calls, cap(recorder.calls), recorder.Truncated())
	}
}

// TestDrawRecorderSnapshotsAndFrames verifies copy-on-read and frame reset.
func TestDrawRecorderSnapshotsAndFrames(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder := newTestRecorder(t, 2)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 10, H: 10}, core.StateNormal)
	calls := recorder.Calls()
	calls[0].Bounds = core.Rect{}
	if got := recorder.Calls()[0].Bounds; got == (core.Rect{}) {
		t.Fatal("Calls returned a mutable recorder slice")
	}

	info := core.WidgetInfo{Name: "latest", Bounds: core.Rect{W: 10, H: 10}, Kind: core.WidgetButton}
	theme.DrawWidget(info, "", 0, false)
	if got := recorder.LastWidgetInfo(); got != info {
		t.Fatalf("last widget = %+v, want %+v", got, info)
	}
	theme.BeginFrame()
	if calls := recorder.Calls(); len(calls) != 0 {
		t.Fatalf("BeginFrame did not reset calls: %v", calls)
	}
	if recorder.Truncated() {
		t.Fatal("BeginFrame did not reset truncation")
	}
	if got := recorder.LastWidgetInfo(); got != (core.WidgetInfo{}) {
		t.Fatalf("BeginFrame did not reset last widget: %+v", got)
	}
}

// TestDrawRecorderDisables verifies nil recorder attachment stops diagnostics.
func TestDrawRecorderDisables(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder := newTestRecorder(t, 2)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 10, H: 10}, core.StateNormal)
	theme.SetDrawRecorder(nil)
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 20, H: 20}, core.StateNormal)
	if got := len(recorder.Calls()); got != 1 {
		t.Fatalf("recorder received calls after disable: got %d want 1", got)
	}
}

// TestDescriptorTintIsExact preserves transparent black descriptor tint.
func TestDescriptorTintIsExact(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder := newTestRecorder(t, 4)
	theme.SetDrawRecorder(recorder)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32},
		Tint:        core.Color{},
		HasTexture:  true,
	})
	theme.BeginFrame()
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 20, H: 20}, core.StateNormal)
	calls := recorder.Calls()
	if len(calls) != 1 || calls[0].Fallback || calls[0].Tint != (core.Color{}) {
		t.Fatalf("transparent-black descriptor call = %+v", calls)
	}
}

// TestCSSDefaultTintIsExplicitWhite checks the loader's no-tint default.
func TestCSSDefaultTintIsExplicitWhite(t *testing.T) {
	white := core.Color{R: 255, G: 255, B: 255, A: 255}
	if got := cssTint(mergedRule{}); got != white {
		t.Fatalf("untinted CSS descriptor = %+v, want %+v", got, white)
	}
	if got := cssTint(mergedRule{hasTint: true, tint: core.Color{}}); got != (core.Color{}) {
		t.Fatalf("transparent-black CSS tint changed to %+v", got)
	}
}

// TestThemeHeadlessAndIsolation covers fallback recording and theme isolation.
func TestThemeHeadlessAndIsolation(t *testing.T) {
	viewport := core.Viewport{Viewport: core.Rect{W: 800, H: 600}, LogicalSize: core.Vec2{X: 800, Y: 600}}
	theme := NewTheme(transform.New(viewport))
	recorder := newTestRecorder(t, 8)
	theme.SetDrawRecorder(recorder)
	bounds := core.Rect{X: 10, Y: 10, W: 100, H: 40}
	theme.BeginFrame()
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, bounds, core.StateNormal)
	calls := recorder.Calls()
	if len(calls) != 1 || !calls[0].Fallback || calls[0].Dest != bounds {
		t.Fatalf("fallback calls=%v", calls)
	}

	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, A: 255}, HasTexture: true,
	})
	theme.SetPixelSnap(true)
	theme.BeginFrame()
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{X: 10.2, Y: 10.7, W: 20.4, H: 20.4}, core.StateHovered)
	calls = recorder.Calls()
	if len(calls) != 1 || calls[0].Fallback || calls[0].Dest != (core.Rect{X: 10, Y: 11, W: 20, H: 20}) {
		t.Fatalf("fallback or snapping calls=%v", calls)
	}

	other := NewTheme(transform.New(viewport))
	otherRecorder := newTestRecorder(t, 2)
	other.SetDrawRecorder(otherRecorder)
	other.BeginFrame()
	other.DrawWidgetPart(core.WidgetButton, skin.PartBackground, bounds, core.StateNormal)
	if calls := otherRecorder.Calls(); len(calls) != 1 || !calls[0].Fallback {
		t.Fatalf("themes share skin state: %v", calls)
	}
}

// TestThemeWidgetVariants covers the whole-widget draw part sequences.
func TestThemeWidgetVariants(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{Viewport: core.Rect{W: 800, H: 600}}))
	recorder := newTestRecorder(t, 8)
	theme.SetDrawRecorder(recorder)
	button := core.WidgetInfo{Name: "okButton", Bounds: core.Rect{W: 100, H: 30}, Kind: core.WidgetButton, State: core.StateNormal}
	theme.BeginFrame()
	theme.DrawWidget(button, "OK", 0, false)
	if len(recorder.Calls()) < 2 {
		t.Fatalf("button draw calls=%v", recorder.Calls())
	}

	theme.BeginFrame()
	slider := core.WidgetInfo{Name: "volumeSlider", Bounds: core.Rect{W: 100, H: 20}, Kind: core.WidgetSlider, State: core.StateNormal}
	theme.DrawWidget(slider, "", 0.5, false)
	if calls := recorder.Calls(); len(calls) != 3 || calls[1].Part != skin.PartTrack || calls[2].Part != skin.PartThumb {
		t.Fatalf("slider draw calls=%v", calls)
	}

	theme.BeginFrame()
	progress := core.WidgetInfo{Name: "loadProgress", Bounds: core.Rect{W: 100, H: 20}, Kind: core.WidgetProgressBar, State: core.StateNormal}
	theme.DrawWidget(progress, "", 0.3, false)
	if calls := recorder.Calls(); len(calls) != 3 || calls[2].Part != skin.PartOverlay {
		t.Fatalf("progress draw calls=%v", calls)
	}
}

// TestThemeFontsHeadless verifies font state without a window: recording a
// present file succeeds with no GL, missing files and nil receivers fail,
// size-matched lookups fall back to the default font, and unload is idempotent.
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
	_ = theme.FontForSize(16)
	_ = theme.ItalicForSize(14)
	_ = theme.Font()
	_ = theme.ItalicFont()
	theme.UnloadFonts()
	theme.UnloadFonts()
	if theme.HasFont() || theme.HasItalicFont() {
		t.Fatal("UnloadFonts must forget recorded paths")
	}
	var nilTheme *Theme
	if nilTheme.HasFont() || nilTheme.HasItalicFont() {
		t.Fatal("nil theme must report no fonts")
	}
	nilTheme.UnloadFonts()
	if err := nilTheme.LoadFont("x.ttf"); err == nil {
		t.Fatal("nil theme LoadFont must fail")
	}
	_ = nilTheme.FontForSize(16)
}

// TestCheckboxIconParts verifies textured checkbox icons and no-skin fallback
// behavior through the recorder without requiring a graphics context.
func TestCheckboxIconParts(t *testing.T) {
	box := core.Rect{X: 10, Y: 10, W: 120, H: 40}
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder := newTestRecorder(t, 16)
	theme.SetDrawRecorder(recorder)

	unchecked := core.WidgetInfo{Name: "agreeBox", Bounds: box, Kind: core.WidgetCheckbox, State: core.StateNormal}
	theme.BeginFrame()
	theme.DrawWidget(unchecked, "", 0, false)
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartIcon || call.Part == skin.PartCheckmark {
			t.Fatalf("no-skin unchecked box must draw no icon, got %v", call.Part)
		}
	}

	icon := skin.SkinDescriptor{Texture: skin.Texture{ID: 7, Width: 32, Height: 32}, AtlasRegion: core.Rect{W: 32, H: 32}, Tint: core.Color{A: 255}, HasTexture: true}
	cross := skin.SkinDescriptor{Texture: skin.Texture{ID: 8, Width: 32, Height: 32}, AtlasRegion: core.Rect{W: 32, H: 32}, Tint: core.Color{A: 255}, HasTexture: true}
	key := skin.SkinKey{Widget: core.WidgetCheckbox, Part: skin.PartIcon, State: core.StateNormal}
	theme.SetSkinPart(key, icon)
	key = skin.SkinKey{Widget: core.WidgetCheckbox, Part: skin.PartCheckmark, State: core.StateNormal}
	theme.SetSkinPart(key, cross)

	theme.BeginFrame()
	theme.DrawWidget(unchecked, "", 0, false)
	foundIcon := false
	for _, call := range recorder.Calls() {
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

	theme.BeginFrame()
	checked := core.WidgetInfo{Name: "agreeBox", Bounds: box, Kind: core.WidgetCheckbox, State: core.StateNormal}
	theme.DrawWidget(checked, "", 0, true)
	foundCheck := false
	for _, call := range recorder.Calls() {
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

// TestEightPatchGeometry verifies border-only textures keep their geometry and
// skip only the center when the descriptor disables center filling.
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

	dest := NinePatchRects(NinePatchConfig{Left: 8, Top: 8, Right: 8, Bottom: 8}, core.Rect{X: 10, Y: 20, W: 200, H: 94})
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

// TestMissingSkinStaysInvisible verifies empty themes log fallback without art.
// Text still records; textured parts must not report success.
func TestMissingSkinStaysInvisible(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder := newTestRecorder(t, 32)
	theme.SetDrawRecorder(recorder)
	box := core.Rect{X: 10, Y: 10, W: 120, H: 40}
	theme.BeginFrame()
	theme.DrawWidget(core.WidgetInfo{Name: "b", Bounds: box, Kind: core.WidgetButton, State: core.StateNormal}, "OK", 0, false)
	theme.DrawWidget(core.WidgetInfo{Name: "c", Bounds: box, Kind: core.WidgetCheckbox, State: core.StateNormal}, "", 0, false)
	theme.DrawWidget(core.WidgetInfo{Name: "s", Bounds: box, Kind: core.WidgetSlider, State: core.StateNormal}, "", 0.5, false)
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartText {
			if call.Fallback {
				t.Fatalf("text must still draw: %+v", call)
			}
			continue
		}
		if !call.Fallback {
			t.Fatalf("missing skin must fall back: %+v", call)
		}
		if call.Part == skin.PartIcon {
			t.Fatalf("unchecked box must emit no icon: %+v", call)
		}
	}
	theme.BeginFrame()
	theme.DrawWidget(core.WidgetInfo{Name: "c", Bounds: box, Kind: core.WidgetCheckbox, State: core.StateNormal}, "", 0, true)
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartCheckmark && !call.Fallback {
			t.Fatalf("untextured checkmark must not succeed: %+v", call)
		}
	}
}

// TestDrawWidgetPartWithoutRecorderAllocatesNothing protects the hot draw path.
func TestDrawWidgetPartWithoutRecorderAllocatesNothing(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.SetDrawRecorder(nil)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, G: 255, B: 255, A: 255}, HasTexture: true,
	})
	allocations := testing.AllocsPerRun(100, func() {
		theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 100, H: 30}, core.StateNormal)
	})
	if allocations != 0 {
		t.Fatalf("DrawWidgetPart allocations = %v, want 0", allocations)
	}
}

// TestDrawWidgetPartColorAndGradientAllocatesNothing verifies zero allocations for color and gradient skins.
func TestDrawWidgetPartColorAndGradientAllocatesNothing(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.SetDrawRecorder(nil)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		BackgroundColor:   core.Color{R: 20, G: 40, B: 60, A: 200},
		HasBackgroundColor: true,
		Gradients: [skin.MaxGradientLayers]skin.LinearGradient{{
			Direction: skin.GradientToBottom,
			Stops: [skin.MaxGradientStops]skin.ColorStop{
				{Color: core.Color{R: 10, G: 20, B: 30, A: 128}, Position: 0},
				{Color: core.Color{R: 40, G: 50, B: 60, A: 128}, Position: 1},
			},
			StopCount: 2,
		}},
		GradientCount: 1,
	})
	allocations := testing.AllocsPerRun(100, func() {
		theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 100, H: 30}, core.StateNormal)
	})
	if allocations != 0 {
		t.Fatalf("color and gradient DrawWidgetPart allocations = %v, want 0", allocations)
	}
}

// TestDrawWidgetPartWithClassAllocatesNothing verifies zero allocations when drawing with a class.
func TestDrawWidgetPartWithClassAllocatesNothing(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.SetDrawRecorder(nil)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetAny, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		BackgroundColor:    core.Color{R: 10, G: 10, B: 10, A: 255},
		HasBackgroundColor: true,
	})
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		BackgroundColor:    core.Color{R: 20, G: 40, B: 60, A: 200},
		HasBackgroundColor: true,
	})
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Class: "danger", Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		BackgroundColor:    core.Color{R: 200, G: 20, B: 20, A: 255},
		HasBackgroundColor: true,
	})
	allocations := testing.AllocsPerRun(100, func() {
		theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 100, H: 30}, core.StateNormal, "danger")
	})
	if allocations != 0 {
		t.Fatalf("class DrawWidgetPart allocations = %v, want 0", allocations)
	}
}

// TestThemeDrawTextAndLayout verifies text layout calculation and alignment for labels and widgets.
func TestThemeDrawTextAndLayout(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	content := core.Rect{X: 10, Y: 10, W: 100, H: 40}

	// Unskinned Label with left alignment starts at content.X
	labelInfo := core.WidgetInfo{Kind: core.WidgetLabel, FontSize: 20, Align: core.AlignLeft}
	size, x, y := theme.textLayout(labelInfo, "Hello", content)
	if size != 20 || x != 10 || y != 20 {
		t.Fatalf("label left: got size=%v x=%v y=%v, want 20, 10, 20", size, x, y)
	}

	// Label with center alignment
	labelInfo.Align = core.AlignCenter
	_, centerX, _ := theme.textLayout(labelInfo, "Hello", content)
	if centerX <= 10 {
		t.Fatalf("label center: expected x > 10, got %v", centerX)
	}

	// Button with left alignment includes standard 6px inset
	buttonInfo := core.WidgetInfo{Kind: core.WidgetButton, FontSize: 20, Align: core.AlignLeft}
	_, btnX, _ := theme.textLayout(buttonInfo, "Hello", content)
	if btnX != 16 {
		t.Fatalf("button left: expected x=16, got %v", btnX)
	}

	// MeasureText returns a positive width for non-empty text
	w := theme.MeasureText("Hello World", 20, false)
	if w <= 0 {
		t.Fatalf("expected positive text width, got %v", w)
	}
}
