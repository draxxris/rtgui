package render

import (
	"errors"
	"fmt"
)

// GlyphRange is an inclusive span of Unicode code points rasterized into font
// atlases.
type GlyphRange struct {
	First, Last rune
}

const (
	// defaultGlyphSizeLimit is the largest pixel size that rasterizes the full
	// default coverage. Larger sizes use the compact base set so a 256px atlas
	// does not grow to hundreds of megabytes.
	defaultGlyphSizeLimit = 48
	// maxGlyphCodepoints bounds the glyph count accepted by SetGlyphRanges.
	maxGlyphCodepoints = 16384
	maxRune            = 0x10FFFF
)

// baseGlyphRanges is the compact set used above defaultGlyphSizeLimit:
// printable ASCII plus en dash, em dash, bullet, and ellipsis.
var baseGlyphRanges = []GlyphRange{
	{0x20, 0x7E}, {0x2013, 0x2014}, {0x2022, 0x2022}, {0x2026, 0x2026},
}

// defaultGlyphRanges covers Basic Latin, Latin-1 Supplement, Latin Extended-A,
// Greek, Cyrillic, general punctuation, and currency symbols.
var defaultGlyphRanges = []GlyphRange{
	{0x20, 0x7E}, {0xA0, 0xFF}, {0x100, 0x17F}, {0x370, 0x3FF}, {0x400, 0x4FF},
	{0x2010, 0x2027}, {0x2030, 0x205E}, {0x20A0, 0x20BF},
}

// SetGlyphRanges replaces the code points rasterized into every font atlas and
// discards cached rasters so they rebuild on next use. Ranges are used at every
// pixel size, so large ranges (CJK, for example) cost atlas memory at large
// sizes. An empty slice restores the defaults. The loaded font must contain the
// glyphs; missing ones render as the font's missing-glyph mark.
func (t *Theme) SetGlyphRanges(ranges []GlyphRange) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	if err := validateGlyphRanges(ranges); err != nil {
		return err
	}
	t.glyphRanges = append([]GlyphRange(nil), ranges...)
	if len(ranges) == 0 {
		t.glyphRanges = nil
	}
	t.font.clearCache()
	t.italic.clearCache()
	for _, face := range t.namedFonts {
		if face != nil {
			face.clearCache()
		}
	}
	t.textRevision++
	return nil
}

// validateGlyphRanges rejects inverted, out-of-range, surrogate, and oversized
// range lists.
func validateGlyphRanges(ranges []GlyphRange) error {
	total := 0
	for _, r := range ranges {
		switch {
		case r.First < 1 || r.Last < r.First || r.Last > maxRune:
			return fmt.Errorf("render: invalid glyph range %U-%U", r.First, r.Last)
		case r.First <= 0xDFFF && r.Last >= 0xD800:
			return fmt.Errorf("render: glyph range %U-%U includes surrogates", r.First, r.Last)
		}
		total += int(r.Last-r.First) + 1
		if total > maxGlyphCodepoints {
			return fmt.Errorf("render: more than %d glyphs requested", maxGlyphCodepoints)
		}
	}
	return nil
}

// glyphCodepoints lists the code points rasterized for one pixel size. Custom
// ranges apply at every size; the defaults shrink to the base set above
// defaultGlyphSizeLimit.
func glyphCodepoints(ranges []GlyphRange, pixels int32) []rune {
	if len(ranges) == 0 {
		ranges = defaultGlyphRanges
		if pixels > defaultGlyphSizeLimit {
			ranges = baseGlyphRanges
		}
	}
	total := 0
	for _, r := range ranges {
		total += int(r.Last-r.First) + 1
	}
	points := make([]rune, 0, total)
	for _, r := range ranges {
		for c := r.First; c <= r.Last; c++ {
			points = append(points, c)
		}
	}
	return points
}
