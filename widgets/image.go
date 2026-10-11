package widgets

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// Image displays a texture scaled into its bounds. It is non-interactive and
// borrows its texture: load one with render.Theme.LoadImage and release it
// with render.Theme.UnloadImage.
type Image struct {
	base
	texture skin.Texture
	source  core.Rect
	tint    core.Color
	fit     core.ImageFit
}

// NewImage returns an enabled image widget with no texture, white tint, and
// contain fit.
func NewImage(name string, bounds core.Rect) *Image {
	return &Image{
		base: newBase(name, core.WidgetImage, bounds),
		tint: core.Color{R: 255, G: 255, B: 255, A: 255},
	}
}

// Texture returns the displayed texture; a zero ID means none is set.
func (i *Image) Texture() skin.Texture {
	if i == nil {
		return skin.Texture{}
	}
	return i.texture
}

// SetTexture replaces the displayed texture. A zero texture hides the image.
func (i *Image) SetTexture(texture skin.Texture) *Image {
	if i != nil {
		i.texture = texture
	}
	return i
}

// Source returns the texture sub-rectangle in pixels; a zero rectangle means
// the whole texture.
func (i *Image) Source() core.Rect {
	if i == nil {
		return core.Rect{}
	}
	return i.source
}

// SetSource selects a texture sub-rectangle in pixels, for example one frame
// of an atlas. A zero rectangle selects the whole texture.
func (i *Image) SetSource(source core.Rect) *Image {
	if i != nil {
		i.source = source
	}
	return i
}

// Tint returns the color multiplied with the texture.
func (i *Image) Tint() core.Color {
	if i == nil {
		return core.Color{}
	}
	return i.tint
}

// SetTint sets the color multiplied with the texture. Opaque white leaves the
// texture unchanged.
func (i *Image) SetTint(tint core.Color) *Image {
	if i != nil {
		i.tint = tint
	}
	return i
}

// Fit returns how the image is scaled into the bounds.
func (i *Image) Fit() core.ImageFit {
	if i == nil {
		return core.ImageContain
	}
	return i.fit
}

// SetFit selects how the image is scaled into the bounds.
func (i *Image) SetFit(fit core.ImageFit) *Image {
	if i != nil {
		i.fit = fit
	}
	return i
}

// SetTooltip attaches a hover tooltip string directly to the image.
func (i *Image) SetTooltip(text string) *Image {
	i.base.SetTooltip(text)
	return i
}
