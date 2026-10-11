package render

import (
	"errors"
	"image/color"
	"os"
	"path/filepath"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// LoadImage decodes encoded image bytes (fileType is an extension such as
// ".png" or ".jpg") and uploads them as a smoothly filtered texture owned by
// the theme. Release it with UnloadImage; UnloadImages releases every one.
// It needs a ready window.
func (t *Theme) LoadImage(fileType string, data []byte) (skin.Texture, error) {
	if t == nil {
		return skin.Texture{}, errors.New("render: nil theme")
	}
	if len(data) == 0 {
		return skin.Texture{}, errors.New("render: empty image data")
	}
	backend := t.ensureTextureBackend()
	if !backend.ready() {
		return skin.Texture{}, errors.New("render: window not ready for image upload")
	}
	texture, err := backend.upload(fileType, data)
	if err != nil {
		return skin.Texture{}, err
	}
	backend.setSmooth(texture)
	t.images = append(t.images, texture)
	return texture, nil
}

// LoadImageFile reads path and loads it with LoadImage, taking the file type
// from the extension.
func (t *Theme) LoadImageFile(path string) (skin.Texture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return skin.Texture{}, err
	}
	return t.LoadImage(filepath.Ext(path), data)
}

// UnloadImage releases one texture returned by LoadImage. Unknown textures
// are ignored, so calling it twice is safe.
func (t *Theme) UnloadImage(texture skin.Texture) {
	if t == nil {
		return
	}
	for i, owned := range t.images {
		if owned == texture {
			t.ensureTextureBackend().unload(texture)
			t.images = append(t.images[:i], t.images[i+1:]...)
			return
		}
	}
}

// UnloadImages releases every texture returned by LoadImage. Call it while
// the graphics context is still active, before closing the window.
func (t *Theme) UnloadImages() {
	if t == nil {
		return
	}
	t.unloadTextures(t.images)
	t.images = nil
}

// ImageFitRects resolves the source sub-rectangle and destination rectangle
// for drawing source into bounds with fit. Contain shrinks the destination,
// cover crops the source, and stretch uses both whole. Empty inputs yield
// empty rectangles.
func ImageFitRects(source, bounds core.Rect, fit core.ImageFit) (src, dst core.Rect) {
	if source.W <= 0 || source.H <= 0 || bounds.W <= 0 || bounds.H <= 0 {
		return core.Rect{}, core.Rect{}
	}
	switch fit {
	case core.ImageStretch:
		return source, bounds
	case core.ImageCover:
		scale := max(bounds.W/source.W, bounds.H/source.H)
		visibleW, visibleH := bounds.W/scale, bounds.H/scale
		return core.Rect{
			X: source.X + (source.W-visibleW)/2, Y: source.Y + (source.H-visibleH)/2,
			W: visibleW, H: visibleH,
		}, bounds
	default:
		scale := min(bounds.W/source.W, bounds.H/source.H)
		w, h := source.W*scale, source.H*scale
		return source, core.Rect{X: bounds.X + (bounds.W-w)/2, Y: bounds.Y + (bounds.H-h)/2, W: w, H: h}
	}
}

// DrawImage draws texture into bounds. A zero source selects the whole
// texture. A zero texture ID draws nothing.
func (t *Theme) DrawImage(bounds core.Rect, texture skin.Texture, source core.Rect, tint core.Color, fit core.ImageFit) {
	if t == nil || texture.ID == 0 {
		return
	}
	if source.W <= 0 || source.H <= 0 {
		source = core.Rect{W: float32(texture.Width), H: float32(texture.Height)}
	}
	src, dst := ImageFitRects(source, bounds, fit)
	if dst.W <= 0 || dst.H <= 0 {
		return
	}
	descriptor := skin.SkinDescriptor{Texture: texture, AtlasRegion: src, Tint: tint, HasTexture: true}
	rgba := color.RGBA{R: tint.R, G: tint.G, B: tint.B, A: tint.A}
	dst = t.snap(dst)
	t.logDrawCall(core.WidgetImage, skin.PartBackground, core.StateNormal, bounds, dst, descriptor, rgba, false)
	t.renderDescriptorOrFallback(descriptor, dst, rgba, false, 0)
}
