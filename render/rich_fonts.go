package render

import (
	"errors"
	"unicode/utf8"

	"github.com/draxxris/rtgui/core"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Named rich-text faces and styled metrics. Go code selects faces by name;
// the gallery registers ValleySans to prove the path. Bold without a bold
// asset renders as faux-bold (double draw with a one-pixel offset).
const (
	// RichMinFontSize clamps per-segment sizes so chat rows stay usable.
	RichMinFontSize = 8
	// RichMaxFontSize clamps per-segment sizes inside the 256px raster cap.
	RichMaxFontSize = 64
)

// RegisterFont records a named TTF/OTF face for rich segments.
// Names match exactly; empty names or missing files report an error.
// Replacing a name unloads its old rasters first so GPU memory never leaks.
// Rasters build lazily per size like the default faces.
func (t *Theme) RegisterFont(name, path string) error {
	if t == nil {
		return nil
	}
	if name == "" {
		return errors.New("render: empty font name")
	}
	face := &fontFace{}
	if err := face.load(path); err != nil {
		return err
	}
	if t.namedFonts == nil {
		t.namedFonts = make(map[string]*fontFace)
	}
	if old := t.namedFonts[name]; old != nil {
		old.unload()
	}
	t.namedFonts[name] = face
	t.textRevision++
	return nil
}

// UnregisterFont unloads one named face and forgets its path.
func (t *Theme) UnregisterFont(name string) {
	if t == nil || t.namedFonts == nil {
		return
	}
	if face := t.namedFonts[name]; face != nil {
		face.unload()
	}
	delete(t.namedFonts, name)
	t.textRevision++
}

// HasNamedFont reports whether a named face is registered.
func (t *Theme) HasNamedFont(name string) bool {
	return t != nil && name != "" && t.namedFonts != nil && t.namedFonts[name] != nil
}

// richFont resolves the raster for a span style without allocating.
// Unknown names fall back to the default face; italic is theme-level only.
func (t *Theme) richFont(size float32, font string) rl.Font {
	size = clampRichFontSize(size)
	if t != nil && font != "" && t.namedFonts != nil {
		if face := t.namedFonts[font]; face != nil {
			return face.forSize(size)
		}
	}
	if t != nil {
		return t.FontForSize(size)
	}
	return rl.GetFontDefault()
}

// measureRichWordStyled returns the advance of word in its span style.
func (t *Theme) measureRichWordStyled(word, font string, size float32) float32 {
	if word == "" {
		return 0
	}
	size = clampRichFontSize(size)
	if t != nil && t.HasFont() && rl.IsWindowReady() {
		if width := rl.MeasureTextEx(t.richFont(size, font), word, size, richTextSpacing).X; width > 0 {
			return width
		}
	}
	if t != nil && rl.IsWindowReady() {
		return float32(rl.MeasureText(word, int32(size)))
	}
	return float32(utf8.RuneCountInString(word)) * size * richEstimatedAdvance
}

// richSpaceFor returns the space advance scaled to a span size.
func (t *Theme) richSpaceFor(space, size float32) float32 {
	if RichFontSize <= 0 {
		return space
	}
	return space * clampRichFontSize(size) / float32(RichFontSize)
}

// richRowHeight returns the row advance for a span size.
func richRowHeight(size float32) float32 {
	height := clampRichFontSize(size) * float32(RichLineHeight) / float32(RichFontSize)
	if height < float32(RichLineHeight) {
		return float32(RichLineHeight)
	}
	return height
}

// clampRichFontSize confines per-segment sizes to the raster budget.
func clampRichFontSize(size float32) float32 {
	if size < float32(RichMinFontSize) {
		return float32(RichMinFontSize)
	}
	if size > float32(RichMaxFontSize) {
		return float32(RichMaxFontSize)
	}
	return size
}

// richSpanSize resolves a segment's effective size for layout and draw.
func richSpanSize(segment core.RichSegment) float32 {
	if segment.HasFontSize {
		return clampRichFontSize(segment.FontSize)
	}
	return float32(RichFontSize)
}

// richSpanFont resolves a segment's face name for layout and draw.
func richSpanFont(segment core.RichSegment) string {
	if segment.HasFont {
		return segment.Font
	}
	return ""
}
