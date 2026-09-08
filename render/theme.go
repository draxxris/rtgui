package render

import (
	"errors"
	"image/color"
	"math"
	"os"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Theme owns render state for one UI. It has no package-global registry or
// viewport, so multiple windows can render independently.
type Theme struct {
	programmatic      *skin.Registry
	css               *skin.Registry
	ownedSkinTextures []skin.Texture
	textureBackend    textureBackend
	transform         *transform.Transform
	recorder          *DrawRecorder
	font              fontFace
	italic            fontFace
	debugMode         bool
	diagnosticHandler func(string)
	clips             []core.Rect
	textRevision      uint64
	warned            []skin.SkinKey
}

// fontFace owns one font path and its size-specific raster cache.
type fontFace struct {
	path      string
	available bool
	cache     map[int32]rl.Font
}

// NewTheme returns a theme using uiTransform, or a fresh empty transform.
func NewTheme(uiTransform *transform.Transform) *Theme {
	if uiTransform == nil {
		uiTransform = transform.New(core.Viewport{})
	}
	return &Theme{
		programmatic:   skin.NewRegistry(),
		css:            skin.NewRegistry(),
		textureBackend: raylibTextureBackend{},
		transform:      uiTransform,
	}
}

// SetDrawRecorder attaches recorder for subsequent diagnostic frames. Passing
// nil disables recording without retaining any draw history in the theme.
func (t *Theme) SetDrawRecorder(recorder *DrawRecorder) {
	if t != nil {
		t.recorder = recorder
	}
}

// BeginFrame resets the attached recorder for a new draw frame.
func (t *Theme) BeginFrame() {
	if t != nil {
		t.recorder.beginFrame()
	}
}

// SetDebugMode enables or disables visible placeholder rendering for missing skins.
func (t *Theme) SetDebugMode(enabled bool) {
	if t != nil {
		t.debugMode = enabled
	}
}

// DebugMode reports whether visible placeholder rendering for missing skins is enabled.
func (t *Theme) DebugMode() bool {
	return t != nil && t.debugMode
}

// SetDiagnosticHandler configures an optional callback invoked on render fallbacks or warnings.
func (t *Theme) SetDiagnosticHandler(handler func(string)) {
	if t != nil {
		t.diagnosticHandler = handler
	}
}

// SetSkinPart stores a borrowed descriptor in the programmatic layer.
func (t *Theme) SetSkinPart(key skin.SkinKey, descriptor skin.SkinDescriptor) {
	if t == nil {
		return
	}
	if t.programmatic == nil {
		t.programmatic = skin.NewRegistry()
	}
	t.programmatic.Set(key, descriptor)
	t.textRevision++
}

// TextRevision invalidates text caches after font or skin configuration changes.
func (t *Theme) TextRevision() uint64 { return t.textRevision }

// GetSkinPart returns an exact CSS descriptor before an exact programmatic descriptor.
func (t *Theme) GetSkinPart(key skin.SkinKey) (skin.SkinDescriptor, error) {
	if t == nil {
		return skin.SkinDescriptor{}, core.StatusMissingSkin
	}
	if descriptor, ok := t.css.Get(key); ok {
		return descriptor, nil
	}
	if descriptor, ok := t.programmatic.Get(key); ok {
		return descriptor, nil
	}
	return skin.SkinDescriptor{}, core.StatusMissingSkin
}

// Lookup resolves exact CSS, normal CSS, exact programmatic, then normal
// programmatic descriptors in that order. CSS normal wins over a programmatic
// exact match so unauthored states inherit their base rule instead of
// falling back to borrowed art.
func (t *Theme) Lookup(kind core.WidgetKind, part skin.SkinPart, state core.WidgetState) (skin.SkinDescriptor, bool) {
	if t == nil {
		return skin.SkinDescriptor{}, false
	}
	key := skin.SkinKey{Widget: kind, Part: part, State: state}
	if descriptor, ok := t.css.Get(key); ok {
		return descriptor, true
	}
	if state != core.StateNormal {
		normal := skin.SkinKey{Widget: kind, Part: part, State: core.StateNormal}
		if descriptor, ok := t.css.Get(normal); ok {
			return descriptor, true
		}
	}
	if descriptor, ok := t.programmatic.Get(key); ok {
		return descriptor, true
	}
	if state != core.StateNormal {
		key.State = core.StateNormal
		if descriptor, ok := t.programmatic.Get(key); ok {
			return descriptor, true
		}
	}
	return skin.SkinDescriptor{}, false
}

// UnloadSkin unloads only Theme-owned CSS textures and clears the CSS layer.
// When textures exist without a graphics context, it retains them for retry.
func (t *Theme) UnloadSkin() error {
	if t == nil {
		return nil
	}
	if len(t.ownedSkinTextures) > 0 && !t.ensureTextureBackend().ready() {
		return errors.New("render: window not ready for skin texture cleanup")
	}
	t.unloadTextures(t.ownedSkinTextures)
	t.ownedSkinTextures = nil
	if t.css != nil {
		t.css.Clear()
	}
	t.textRevision++
	return nil
}

// ClearSkin unloads the owned CSS layer and then clears borrowed descriptors
// without unloading their programmatic texture handles.
func (t *Theme) ClearSkin() error {
	if t == nil {
		return nil
	}
	if err := t.UnloadSkin(); err != nil {
		return err
	}
	if t.programmatic != nil {
		t.programmatic.Clear()
	}
	t.textRevision++
	return nil
}

// SetViewport replaces the logical-to-physical viewport transform.
func (t *Theme) SetViewport(viewport core.Viewport) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	return t.ensureTransform().SetViewport(viewport)
}

// GetViewport reports the theme's current viewport.
func (t *Theme) GetViewport() core.Viewport {
	if t == nil || t.transform == nil {
		return core.Viewport{}
	}
	return t.transform.Viewport
}

// SetPixelSnap enables or disables logical rectangle pixel snapping.
func (t *Theme) SetPixelSnap(enabled bool) {
	if t != nil {
		t.ensureTransform().PixelSnap = enabled
	}
}

// GetPixelSnap reports whether logical rectangle pixel snapping is enabled.
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
	if err := t.font.load(path); err != nil {
		return err
	}
	t.textRevision++
	return nil
}

// LoadItalicFont records a second TTF/OTF file for accent text. Its lifecycle
// and raster behavior match LoadFont.
func (t *Theme) LoadItalicFont(path string) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	if err := t.italic.load(path); err != nil {
		return err
	}
	t.textRevision++
	return nil
}

// HasFont reports whether a widget-text font file is recorded.
func (t *Theme) HasFont() bool { return t != nil && t.font.available }

// HasItalicFont reports whether an accent font file is recorded.
func (t *Theme) HasItalicFont() bool { return t != nil && t.italic.available }

// Font returns the widget-text font rasterized at 32px, or the raylib default
// when none is recorded. Prefer FontForSize for crisp text.
func (t *Theme) Font() rl.Font {
	if t != nil {
		return t.font.forSize(32)
	}
	return rl.GetFontDefault()
}

// ItalicFont returns the accent font rasterized at 32px, or the raylib default
// when none is recorded. Prefer ItalicForSize for crisp text.
func (t *Theme) ItalicFont() rl.Font {
	if t != nil {
		return t.italic.forSize(32)
	}
	return rl.GetFontDefault()
}

// FontForSize returns the widget-text font rasterized at the requested pixel
// size, building and caching the atlas on first use. Without a recorded font
// or ready window it returns the raylib default, so headless draws still work.
func (t *Theme) FontForSize(size float32) rl.Font {
	if t == nil {
		return rl.GetFontDefault()
	}
	return t.font.forSize(size)
}

// ItalicForSize returns the accent font rasterized at the requested size.
// Fallback semantics mirror FontForSize.
func (t *Theme) ItalicForSize(size float32) rl.Font {
	if t == nil {
		return rl.GetFontDefault()
	}
	return t.italic.forSize(size)
}

// UnloadFonts releases all rasterized font atlases and forgets recorded paths.
// It is safe to call repeatedly or when no font has been loaded.
func (t *Theme) UnloadFonts() {
	if t == nil {
		return
	}
	t.font.unload()
	t.italic.unload()
	t.textRevision++
}

// MeasureText measures the pixel width of value at size using the theme or italic font.
func (t *Theme) MeasureText(value string, size float32, italic bool) float32 {
	if value == "" || size <= 0 {
		return 0
	}
	if !rl.IsWindowReady() {
		return float32(len(value)) * (size * 0.5)
	}
	if t != nil && italic && t.HasItalicFont() {
		return rl.MeasureTextEx(t.ItalicForSize(size), value, size, size/10).X
	}
	if t != nil && t.HasFont() {
		return rl.MeasureTextEx(t.FontForSize(size), value, size, size/10).X
	}
	return float32(rl.MeasureText(value, int32(size)))
}

// DrawText renders text using the loaded font (or italic font) at the given size, position, and tint.
func (t *Theme) DrawText(value string, x, y, size float32, italic bool, tint color.RGBA) {
	if value == "" || size <= 0 || !rl.IsWindowReady() {
		return
	}
	if t != nil && italic && t.HasItalicFont() {
		rl.DrawTextEx(t.ItalicForSize(size), value, rl.NewVector2(x, y), size, size/10, tint)
		return
	}
	if t != nil && t.HasFont() {
		rl.DrawTextEx(t.FontForSize(size), value, rl.NewVector2(x, y), size, size/10, tint)
		return
	}
	rl.DrawText(value, int32(x), int32(y), int32(size), tint)
}

// SetTexture stores a borrowed raylib texture handle in a skin descriptor.
// Raylib is intentionally confined to this render package.
func (t *Theme) SetTexture(key skin.SkinKey, texture rl.Texture2D, region core.Rect, tint core.Color) {
	t.SetSkinPart(key, skin.SkinDescriptor{
		Texture:     fromRaylibTexture(texture),
		AtlasRegion: region,
		Tint:        tint,
		HasTexture:  true,
	})
}

// Transform exposes the theme's logical-to-physical transform.
func (t *Theme) Transform() *transform.Transform {
	if t == nil {
		return nil
	}
	return t.ensureTransform()
}

// load validates path, clears old raster data, and records the new face.
func (f *fontFace) load(path string) error {
	if _, err := os.Stat(path); err != nil {
		return errors.New("render: font not found")
	}
	f.clearCache()
	f.path = path
	f.available = true
	return nil
}

// forSize rasterizes the recorded face once per rounded positive pixel size.
func (f *fontFace) forSize(size float32) rl.Font {
	if !f.available {
		return rl.GetFontDefault()
	}
	if math.IsNaN(float64(size)) || size <= 0 {
		size = 1
	}
	if size > 256 {
		size = 256
	}
	pixels := int32(math.Round(float64(size)))
	if pixels < 1 {
		pixels = 1
	}
	if f.cache == nil {
		f.cache = make(map[int32]rl.Font)
	}
	if font, ok := f.cache[pixels]; ok {
		return font
	}
	if !rl.IsWindowReady() {
		return rl.GetFontDefault()
	}
	if len(f.cache) >= 32 {
		return f.nearestRaster(pixels)
	}
	font := rl.LoadFontEx(f.path, pixels, fontCodepoints(), 0)
	if !rl.IsFontValid(font) {
		return rl.GetFontDefault()
	}
	f.cache[pixels] = font
	return font
}

// nearestRaster bounds atlas retention without unloading queued GPU resources.
func (f *fontFace) nearestRaster(pixels int32) rl.Font {
	best, distance, bestSize := rl.GetFontDefault(), int32(1<<30), int32(1<<30)
	for size, font := range f.cache {
		delta := size - pixels
		if delta < 0 {
			delta = -delta
		}
		if delta < distance || delta == distance && size < bestSize {
			best, distance, bestSize = font, delta, size
		}
	}
	return best
}

// unload releases all cached rasters and clears the recorded source path.
func (f *fontFace) unload() {
	f.clearCache()
	f.path = ""
	f.available = false
}

// clearCache releases cached raylib rasters while preserving the source path.
func (f *fontFace) clearCache() {
	for _, font := range f.cache {
		rl.UnloadFont(font)
	}
	f.cache = nil
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

// ensureTransform creates the transform lazily for a zero-value theme.
func (t *Theme) ensureTransform() *transform.Transform {
	if t.transform == nil {
		t.transform = transform.New(core.Viewport{})
	}
	return t.transform
}

// fromRaylibTexture converts a raylib handle to skin's renderer-free value.
func fromRaylibTexture(texture rl.Texture2D) skin.Texture {
	return skin.Texture{ID: texture.ID, Width: texture.Width, Height: texture.Height, Mipmaps: texture.Mipmaps, Format: int32(texture.Format)}
}
