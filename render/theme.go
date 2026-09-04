package render

import (
	"errors"

	rl "github.com/gen2brain/raylib-go/raylib"
	"rtgui/core"
	"rtgui/skin"
	"rtgui/transform"
)

// Theme owns render state for one UI. It has no package-global registry or
// viewport, so multiple windows can render independently.
type Theme struct {
	registry       *skin.Registry
	transform      *transform.Transform
	drawLog        []DrawCall
	lastWidgetInfo core.WidgetInfo
}

func NewTheme(uiTransform *transform.Transform) *Theme {
	if uiTransform == nil {
		uiTransform = transform.New(core.Viewport{})
	}
	return &Theme{registry: skin.NewRegistry(), transform: uiTransform}
}

func (t *Theme) SetSkinPart(key skin.SkinKey, descriptor skin.SkinDescriptor) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	if t.registry == nil {
		t.registry = skin.NewRegistry()
	}
	t.registry.Set(key, descriptor)
	return nil
}

func (t *Theme) GetSkinPart(key skin.SkinKey) (skin.SkinDescriptor, error) {
	if t == nil || t.registry == nil {
		return skin.SkinDescriptor{}, core.StatusMissingSkin
	}
	if descriptor, ok := t.registry.Get(key); ok {
		return descriptor, nil
	}
	return skin.SkinDescriptor{}, core.StatusMissingSkin
}

func (t *Theme) Lookup(kind core.WidgetKind, part skin.SkinPart, state core.WidgetState) (skin.SkinDescriptor, bool) {
	if t == nil || t.registry == nil {
		return skin.SkinDescriptor{}, false
	}
	return t.registry.Lookup(kind, part, state)
}

func (t *Theme) ClearSkin() {
	if t != nil && t.registry != nil {
		t.registry.Clear()
	}
}

func (t *Theme) SetViewport(viewport core.Viewport) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	return t.ensureTransform().SetViewport(viewport)
}

func (t *Theme) GetViewport() core.Viewport {
	if t == nil || t.transform == nil {
		return core.Viewport{}
	}
	return t.transform.Viewport
}

func (t *Theme) SetPixelSnap(enabled bool) {
	if t != nil {
		t.ensureTransform().PixelSnap = enabled
	}
}

func (t *Theme) GetPixelSnap() bool {
	return t != nil && t.transform != nil && t.transform.PixelSnap
}

// SetTexture stores a raylib texture handle in a skin descriptor. Raylib is
// intentionally confined to this render package.
func (t *Theme) SetTexture(key skin.SkinKey, texture rl.Texture2D, region core.Rect, tint core.Color) error {
	return t.SetSkinPart(key, skin.SkinDescriptor{
		Texture:     fromRaylibTexture(texture),
		AtlasRegion: region,
		Tint:        tint,
		Alpha:       1,
		HasTexture:  true,
	})
}

func (t *Theme) Transform() *transform.Transform {
	if t == nil {
		return nil
	}
	return t.ensureTransform()
}

func (t *Theme) ensureTransform() *transform.Transform {
	if t.transform == nil {
		t.transform = transform.New(core.Viewport{})
	}
	return t.transform
}

func (t *Theme) ClearDrawLog() {
	if t != nil {
		t.drawLog = nil
	}
}

func (t *Theme) DrawLog() []DrawCall {
	if t == nil {
		return nil
	}
	return append([]DrawCall(nil), t.drawLog...)
}

func (t *Theme) LastWidgetInfo() core.WidgetInfo {
	if t == nil {
		return core.WidgetInfo{}
	}
	return t.lastWidgetInfo
}

func fromRaylibTexture(texture rl.Texture2D) skin.Texture {
	return skin.Texture{ID: texture.ID, Width: texture.Width, Height: texture.Height, Mipmaps: texture.Mipmaps, Format: int32(texture.Format)}
}
