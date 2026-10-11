package widgets_test

import (
	"errors"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// TestImageDefaultsAndSetters checks defaults and setter chaining.
func TestImageDefaultsAndSetters(t *testing.T) {
	img := widgets.NewImage("banner", core.Rect{W: 100, H: 50})
	white := core.Color{R: 255, G: 255, B: 255, A: 255}
	if img.Kind() != core.WidgetImage || img.Tint() != white || img.Fit() != core.ImageContain || img.Texture().ID != 0 {
		t.Fatalf("defaults: kind=%v tint=%v fit=%v texture=%v", img.Kind(), img.Tint(), img.Fit(), img.Texture())
	}
	texture := skin.Texture{ID: 7, Width: 64, Height: 32}
	if img.SetTexture(texture) != img {
		t.Fatal("SetTexture must return the image for chaining")
	}
	source := core.Rect{X: 8, Y: 4, W: 16, H: 16}
	tint := core.Color{R: 10, G: 20, B: 30, A: 40}
	img.SetSource(source).SetTint(tint).SetFit(core.ImageCover)
	if img.Source() != source || img.Tint() != tint || img.Fit() != core.ImageCover || img.Texture() != texture {
		t.Fatalf("setters: %v %v %v %v", img.Source(), img.Tint(), img.Fit(), img.Texture())
	}
}

// TestImageNilReceiverIsSafe checks that accessors tolerate a nil image.
func TestImageNilReceiverIsSafe(t *testing.T) {
	var img *widgets.Image
	if img.SetTexture(skin.Texture{ID: 1}) != nil || img.Texture().ID != 0 || img.Fit() != core.ImageContain {
		t.Fatal("nil image must report zero values")
	}
	if img.SetFit(core.ImageStretch) != nil {
		t.Fatal("nil setters must return nil")
	}
}

// TestAsImageChecksKind checks the checked cast helper.
func TestAsImageChecksKind(t *testing.T) {
	if _, err := widgets.AsImage(nil); !errors.Is(err, widgets.ErrNilWidget) {
		t.Fatalf("nil cast = %v", err)
	}
	if _, err := widgets.AsImage(widgets.NewButton("b", core.Rect{}, "b")); !errors.Is(err, widgets.ErrKindMismatch) {
		t.Fatalf("button cast = %v", err)
	}
	if img, err := widgets.AsImage(widgets.NewImage("i", core.Rect{})); err != nil || img == nil {
		t.Fatalf("image cast = %v, %v", img, err)
	}
}
