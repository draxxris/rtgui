package render

import (
	"errors"
	"math"
	"os"

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
	fontPath       string
	hasFont        bool
	fonts          map[int32]rl.Font
	italicPath     string
	hasItalic      bool
	italics        map[int32]rl.Font
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

// LoadFont records a TTF/OTF file as the widget-text font. Rasters are built
// lazily per pixel size on first use (see FontForSize), so glyphs are always
// drawn 1:1 instead of scaled down from a single atlas, which blurs small
// text. Only the file-exists check runs here; no window is required.
func (t *Theme) LoadFont(path string) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	if _, err := os.Stat(path); err != nil {
		return errors.New("render: font not found")
	}
	t.clearFontCache()
	t.fontPath, t.hasFont = path, true
	return nil
}

// LoadItalicFont records a second TTF/OTF file for accent text (hints, status).
// Semantics mirror LoadFont; galleries use it for captions and the status line.
func (t *Theme) LoadItalicFont(path string) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	if _, err := os.Stat(path); err != nil {
		return errors.New("render: font not found")
	}
	t.clearItalicCache()
	t.italicPath, t.hasItalic = path, true
	return nil
}

// HasFont reports whether a widget-text font file is recorded.
func (t *Theme) HasFont() bool { return t != nil && t.hasFont }

// HasItalicFont reports whether the accent font file is recorded.
func (t *Theme) HasItalicFont() bool { return t != nil && t.hasItalic }

// Font returns the widget-text font rasterized at 32px, or the raylib
// default when none is recorded. Prefer FontForSize for crisp text.
func (t *Theme) Font() rl.Font {
	if t != nil && t.hasFont {
		return t.FontForSize(32)
	}
	return rl.GetFontDefault()
}

// ItalicFont returns the accent font rasterized at 32px, or the raylib
// default when none is recorded. Prefer ItalicForSize for crisp text.
func (t *Theme) ItalicFont() rl.Font {
	if t != nil && t.hasItalic {
		return t.ItalicForSize(32)
	}
	return rl.GetFontDefault()
}

// FontForSize returns the widget-text font rasterized at the requested pixel
// size, building and caching the atlas on first use. Without a recorded font
// or ready window it returns the raylib default, so headless draws still log.
func (t *Theme) FontForSize(size float32) rl.Font {
	if t == nil || !t.hasFont {
		return rl.GetFontDefault()
	}
	return t.cachedFont(t.fontPath, &t.fonts, size)
}

// ItalicForSize returns the accent font rasterized at the requested size.
// Fallback semantics mirror FontForSize.
func (t *Theme) ItalicForSize(size float32) rl.Font {
	if t == nil || !t.hasItalic {
		return rl.GetFontDefault()
	}
	return t.cachedFont(t.italicPath, &t.italics, size)
}

// cachedFont rasterizes path at the rounded size once and caches the atlas.
// Sizes round to whole pixels with a floor of 1; unready windows and invalid
// rasters fall back to the default font without caching failures.
func (t *Theme) cachedFont(path string, cache *map[int32]rl.Font, size float32) rl.Font {
	pixels := int32(math.Round(float64(size)))
	if pixels < 1 {
		pixels = 1
	}
	if *cache == nil {
		*cache = make(map[int32]rl.Font)
	}
	if font, ok := (*cache)[pixels]; ok {
		return font
	}
	if !rl.IsWindowReady() {
		return rl.GetFontDefault()
	}
	font := rl.LoadFontEx(path, pixels, fontCodepoints(), 0)
	if !rl.IsFontValid(font) {
		return rl.GetFontDefault()
	}
	(*cache)[pixels] = font
	return font
}

// fontCodepoints lists the glyphs rasterized into theme font atlases: printable
// ASCII plus the punctuation the gallery uses (em/en dashes, bullet, ellipsis).
// Glyphs outside this set render as missing-glyph marks.
func fontCodepoints() []rune {
	points := make([]rune, 0, 100)
	for r := rune(32); r <= rune(126); r++ {
		points = append(points, r)
	}
	return append(points, '\u2013', '\u2014', '\u2022', '\u2026')
}

// UnloadFonts releases rasterized font atlases and forgets recorded paths.
// Safe to call without loaded fonts.
func (t *Theme) UnloadFonts() {
	if t == nil {
		return
	}
	t.clearFontCache()
	t.clearItalicCache()
	t.fontPath, t.hasFont = "", false
	t.italicPath, t.hasItalic = "", false
}

// clearFontCache releases rasterized widget-text atlases, keeping the path.
func (t *Theme) clearFontCache() {
	for _, font := range t.fonts {
		rl.UnloadFont(font)
	}
	t.fonts = nil
}

// clearItalicCache releases rasterized accent atlases, keeping the path.
func (t *Theme) clearItalicCache() {
	for _, font := range t.italics {
		rl.UnloadFont(font)
	}
	t.italics = nil
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
