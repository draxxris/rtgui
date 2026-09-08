package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

// TestEffectiveBackgroundRadius verifies radius gating and CSS clamping.
func TestEffectiveBackgroundRadius(t *testing.T) {
	dest := core.Rect{W: 100, H: 40}
	plain := skin.SkinDescriptor{}
	if got := effectiveBackgroundRadius(plain, dest); got != 0 {
		t.Fatalf("unauthored radius = %v", got)
	}
	rounded := skin.SkinDescriptor{Radius: 6, HasRadius: true}
	if got := effectiveBackgroundRadius(rounded, dest); got != 6 {
		t.Fatalf("radius = %v", got)
	}
	huge := skin.SkinDescriptor{Radius: 100, HasRadius: true}
	if got := effectiveBackgroundRadius(huge, dest); got != 20 {
		t.Fatalf("clamped radius = %v, want 20", got)
	}
	if got := effectiveBackgroundRadius(rounded, core.Rect{}); got != 0 {
		t.Fatalf("degenerate dest radius = %v", got)
	}
	zero := skin.SkinDescriptor{Radius: 0, HasRadius: true}
	if got := effectiveBackgroundRadius(zero, dest); got != 0 {
		t.Fatalf("zero radius = %v", got)
	}
}

// TestRoundedBandsTileDest verifies bands partition dest with no gaps.
func TestRoundedBandsTileDest(t *testing.T) {
	dest := core.Rect{X: 10, Y: 20, W: 100, H: 40}
	const radius = 6
	count := roundedBandCount(radius)
	if count != 13 {
		t.Fatalf("band count = %d, want 13", count)
	}
	bands := make([]core.Rect, count)
	for i := range bands {
		bands[i] = roundedBand(dest, radius, i, count)
	}
	assertBandsStackVertically(t, dest, bands)
	assertBandsCentered(t, dest, bands, count)
}

// assertBandsStackVertically verifies bands tile dest top to bottom with no gaps.
func assertBandsStackVertically(t *testing.T, dest core.Rect, bands []core.Rect) {
	t.Helper()
	if bands[0].Y != dest.Y {
		t.Fatalf("first band y = %v", bands[0].Y)
	}
	height := float32(0)
	for i, band := range bands {
		if i > 0 && band.Y != bands[i-1].Y+bands[i-1].H {
			t.Fatalf("band %d starts at %v after %+v", i, band.Y, bands[i-1])
		}
		height += band.H
	}
	if height != dest.H {
		t.Fatalf("band heights sum to %v, want %v", height, dest.H)
	}
	middle := bands[len(bands)/2]
	if middle.X != dest.X || middle.W != dest.W {
		t.Fatalf("middle band keeps full width, got %+v", middle)
	}
}

// assertBandsCentered verifies corner rows stay centered with insets shrinking
// toward the middle and mirroring top to bottom.
func assertBandsCentered(t *testing.T, dest core.Rect, bands []core.Rect, count int) {
	t.Helper()
	previous := float32(math.MaxFloat32)
	for i := range count / 2 {
		inset := bands[i].X - dest.X
		right := dest.X + dest.W - (bands[i].X + bands[i].W)
		if math.Abs(float64(inset-right)) > 0.001 {
			t.Fatalf("band %d off-center: %+v", i, bands[i])
		}
		if i == 0 && inset <= 0 {
			t.Fatalf("outer row must cut the corner, got %+v", bands[i])
		}
		if inset >= previous {
			t.Fatalf("insets must shrink toward the middle, band %d = %+v", i, bands[i])
		}
		previous = inset
		mirror := bands[count-1-i]
		if mirror.X != bands[i].X || mirror.W != bands[i].W {
			t.Fatalf("band %d mirrors %+v, got %+v", i, bands[i], mirror)
		}
	}
}

// TestRoundedRowInsetHugsArc verifies the staircase stays inside the circle.
func TestRoundedRowInsetHugsArc(t *testing.T) {
	const radius = 8
	for row := range 8 {
		inset := roundedRowInset(radius, row)
		// The cut corner sampled at the row middle must sit on the arc.
		x := radius - inset
		dy := radius - float32(row) - 0.5
		if got := x*x + dy*dy; math.Abs(float64(got-radius*radius)) > 1.01 {
			t.Fatalf("row %d misses the arc: %v", row, got)
		}
	}
}

// TestBilinearColorSamplesQuad verifies corner and midpoint sampling.
func TestBilinearColorSamplesQuad(t *testing.T) {
	topLeft := color.RGBA{R: 255, A: 255}
	bottomLeft := color.RGBA{G: 255, A: 255}
	bottomRight := color.RGBA{B: 255, A: 255}
	topRight := color.RGBA{R: 255, G: 255, A: 255}
	if got := bilinearColor(topLeft, bottomLeft, bottomRight, topRight, 0, 0); got != topLeft {
		t.Fatalf("u0v0 = %+v", got)
	}
	if got := bilinearColor(topLeft, bottomLeft, bottomRight, topRight, 0, 1); got != bottomLeft {
		t.Fatalf("u0v1 = %+v", got)
	}
	if got := bilinearColor(topLeft, bottomLeft, bottomRight, topRight, 1, 1); got != bottomRight {
		t.Fatalf("u1v1 = %+v", got)
	}
	if got := bilinearColor(topLeft, bottomLeft, bottomRight, topRight, 1, 0); got != topRight {
		t.Fatalf("u1v0 = %+v", got)
	}
	mid := bilinearColor(topLeft, bottomLeft, bottomRight, topRight, 0.5, 0.5)
	want := color.RGBA{R: 128, G: 128, B: 64, A: 255}
	if mid != want {
		t.Fatalf("midpoint = %+v, want %+v", mid, want)
	}
}

// TestMergeSkinRulesRadiusInheritance verifies state radius fallback.
func TestMergeSkinRulesRadiusInheritance(t *testing.T) {
	rules := []skin.SkinRule{
		{Kind: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal,
			Image: "bg.png", HasImage: true, Radius: 6, HasRadius: true},
		{Kind: core.WidgetButton, Part: skin.PartBackground, State: core.StateHovered,
			Image: "bg.png", HasImage: true},
		{Kind: core.WidgetButton, Part: skin.PartBackground, State: core.StatePressed,
			Image: "bg.png", HasImage: true, Radius: 2, HasRadius: true},
	}
	merged, _ := mergeSkinRules(rules)
	hover := merged[skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateHovered}]
	if !hover.hasRadius || hover.radius != 6 {
		t.Fatalf("hover must inherit radius, got %+v", hover)
	}
	pressed := merged[skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StatePressed}]
	if !pressed.hasRadius || pressed.radius != 2 {
		t.Fatalf("pressed keeps its radius, got %+v", pressed)
	}
}

// TestLoadCSSBorderRadius verifies the file path stores radius on descriptors.
func TestLoadCSSBorderRadius(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "pixel.png", color.RGBA{R: 200, G: 100, A: 255})
	cssPath := writeCSS(t, directory, `
Button { background-image: url(pixel.png); border-radius: 6; }
Button:hover { background-image-tint: #aabbccdd; }
Button:active { border-radius: 2; }
`)
	theme := newFakeTheme(&fakeTextureBackend{isReady: true})
	mustLoadCSS(t, theme, cssPath)
	normal, err := theme.GetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal})
	if err != nil {
		t.Fatal(err)
	}
	if !normal.HasRadius || normal.Radius != 6 {
		t.Fatalf("normal radius = %+v", normal)
	}
	hover, err := theme.GetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateHovered})
	if err != nil {
		t.Fatal(err)
	}
	if !hover.HasRadius || hover.Radius != 6 {
		t.Fatalf("hover must inherit radius, got %+v", hover)
	}
	pressed, err := theme.GetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StatePressed})
	if err != nil {
		t.Fatal(err)
	}
	if !pressed.HasRadius || pressed.Radius != 2 {
		t.Fatalf("pressed keeps its radius, got %+v", pressed)
	}
}

// TestDrawWidgetPartRoundedBackgroundLogsSingleCall verifies the recorder still
// sees one background call while bands draw underneath it.
func TestDrawWidgetPartRoundedBackgroundLogsSingleCall(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		BackgroundColor: core.Color{R: 10, G: 20, B: 30, A: 255}, HasBackgroundColor: true,
		Radius: 6, HasRadius: true,
	})
	recorder, _ := NewDrawRecorder(8)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{X: 5, Y: 5, W: 100, H: 40}, core.StateNormal)
	calls := recorder.Calls()
	if len(calls) != 1 || calls[0].Part != skin.PartBackground || calls[0].Fallback {
		t.Fatalf("rounded background calls = %+v", calls)
	}
}
