package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// TestImageFitRects checks contain, cover, and stretch geometry.
func TestImageFitRects(t *testing.T) {
	source := core.Rect{W: 200, H: 100}
	bounds := core.Rect{X: 10, Y: 20, W: 100, H: 100}
	cases := []struct {
		name     string
		fit      core.ImageFit
		src, dst core.Rect
	}{
		{"contain", core.ImageContain, source, core.Rect{X: 10, Y: 45, W: 100, H: 50}},
		{"cover", core.ImageCover, core.Rect{X: 50, Y: 0, W: 100, H: 100}, bounds},
		{"stretch", core.ImageStretch, source, bounds},
	}
	for _, c := range cases {
		src, dst := ImageFitRects(source, bounds, c.fit)
		if src != c.src || dst != c.dst {
			t.Errorf("%s: src=%v dst=%v, want src=%v dst=%v", c.name, src, dst, c.src, c.dst)
		}
	}
}

// TestImageFitRectsEmptyInputs checks that degenerate rectangles yield nothing.
func TestImageFitRectsEmptyInputs(t *testing.T) {
	for _, fit := range []core.ImageFit{core.ImageContain, core.ImageCover, core.ImageStretch} {
		if src, dst := ImageFitRects(core.Rect{}, core.Rect{W: 10, H: 10}, fit); src != (core.Rect{}) || dst != (core.Rect{}) {
			t.Errorf("fit %v empty source: %v %v", fit, src, dst)
		}
		if src, dst := ImageFitRects(core.Rect{W: 10, H: 10}, core.Rect{W: 0, H: 10}, fit); src != (core.Rect{}) || dst != (core.Rect{}) {
			t.Errorf("fit %v empty bounds: %v %v", fit, src, dst)
		}
	}
}

// TestLoadImageOwnsAndUnloadsTextures checks upload, smoothing, and release through the backend.
func TestLoadImageOwnsAndUnloadsTextures(t *testing.T) {
	backend := &fakeTextureBackend{isReady: true}
	theme := newFakeTheme(backend)
	first, err := theme.LoadImage(".png", []byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := theme.LoadImage(".jpg", []byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	if backend.smoothCalls != 2 || len(theme.images) != 2 {
		t.Fatalf("smooth=%d owned=%d", backend.smoothCalls, len(theme.images))
	}
	theme.UnloadImage(first)
	theme.UnloadImage(first)
	if len(backend.unloaded) != 1 || backend.unloaded[0] != first.ID || len(theme.images) != 1 {
		t.Fatalf("unloaded=%v owned=%d", backend.unloaded, len(theme.images))
	}
	theme.UnloadImages()
	if len(backend.unloaded) != 2 || backend.unloaded[1] != second.ID || len(theme.images) != 0 {
		t.Fatalf("unloaded=%v owned=%d", backend.unloaded, len(theme.images))
	}
}

// TestLoadImageRejectsBadInput checks the empty-data, no-window, and decode-failure errors.
func TestLoadImageRejectsBadInput(t *testing.T) {
	backend := &fakeTextureBackend{isReady: true, failUpload: 1}
	theme := newFakeTheme(backend)
	if _, err := theme.LoadImage(".png", nil); err == nil {
		t.Error("empty data must fail")
	}
	if _, err := theme.LoadImage(".png", []byte("x")); err == nil || len(theme.images) != 0 {
		t.Errorf("failed upload must not be tracked: %v", err)
	}
	backend.isReady = false
	if _, err := theme.LoadImage(".png", []byte("x")); err == nil {
		t.Error("missing window must fail")
	}
	if _, err := (*Theme)(nil).LoadImage(".png", []byte("x")); err == nil {
		t.Error("nil theme must fail")
	}
}

// TestDrawImageRecordsFittedDestination checks the recorded geometry without a window.
func TestDrawImageRecordsFittedDestination(t *testing.T) {
	theme := newFakeTheme(&fakeTextureBackend{})
	recorder := newTestRecorder(t, 8)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	texture := skin.Texture{ID: 3, Width: 200, Height: 100}
	white := core.Color{R: 255, G: 255, B: 255, A: 255}
	theme.DrawImage(core.Rect{W: 100, H: 100}, texture, core.Rect{}, white, core.ImageContain)
	theme.DrawImage(core.Rect{W: 100, H: 100}, skin.Texture{}, core.Rect{}, white, core.ImageContain)
	calls := recorder.Calls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1 (zero texture draws nothing)", len(calls))
	}
	if calls[0].Kind != core.WidgetImage || calls[0].Dest != (core.Rect{Y: 25, W: 100, H: 50}) || calls[0].Src != (core.Rect{W: 200, H: 100}) {
		t.Fatalf("call = %+v", calls[0])
	}
}
