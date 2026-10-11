package render

import (
	"slices"
	"testing"
)

// TestGlyphCodepointsDefaultsShrinkAtLargeSizes checks the size-dependent default coverage.
func TestGlyphCodepointsDefaultsShrinkAtLargeSizes(t *testing.T) {
	small := glyphCodepoints(nil, 16)
	for _, r := range []rune{'A', 'é', 'Ā', 'Ω', 'Ж', '€', '—'} {
		if !slices.Contains(small, r) {
			t.Errorf("16px coverage is missing %q", r)
		}
	}
	large := glyphCodepoints(nil, defaultGlyphSizeLimit+1)
	if !slices.Contains(large, 'A') || !slices.Contains(large, '—') {
		t.Error("large sizes must keep ASCII and dashes")
	}
	if slices.Contains(large, 'Ж') || len(large) >= len(small) {
		t.Errorf("large sizes must use the compact set: %d vs %d glyphs", len(large), len(small))
	}
}

// TestGlyphCodepointsCustomRangesApplyAtEverySize checks that explicit ranges are not shrunk.
func TestGlyphCodepointsCustomRangesApplyAtEverySize(t *testing.T) {
	ranges := []GlyphRange{{0x41, 0x43}, {0x3040, 0x3042}}
	for _, size := range []int32{12, 200} {
		got := glyphCodepoints(ranges, size)
		if !slices.Equal(got, []rune{'A', 'B', 'C', 0x3040, 0x3041, 0x3042}) {
			t.Errorf("size %d = %U", size, got)
		}
	}
}

// TestSetGlyphRangesValidates checks rejection of malformed ranges and reset to defaults.
func TestSetGlyphRangesValidates(t *testing.T) {
	theme := NewTheme(nil)
	bad := [][]GlyphRange{
		{{0x50, 0x40}},
		{{0, 0x10}},
		{{0x41, 0x110000}},
		{{0xD700, 0xD900}},
		{{0x20, 0x20 + maxGlyphCodepoints}},
	}
	for _, ranges := range bad {
		if err := theme.SetGlyphRanges(ranges); err == nil {
			t.Errorf("%v must be rejected", ranges)
		}
	}
	revision := theme.textRevision
	if err := theme.SetGlyphRanges([]GlyphRange{{0x41, 0x5A}}); err != nil || theme.glyphRanges == nil || theme.textRevision == revision {
		t.Fatalf("valid ranges: err=%v ranges=%v", err, theme.glyphRanges)
	}
	if err := theme.SetGlyphRanges(nil); err != nil || theme.glyphRanges != nil {
		t.Fatalf("reset: err=%v ranges=%v", err, theme.glyphRanges)
	}
	if err := (*Theme)(nil).SetGlyphRanges(nil); err == nil {
		t.Error("nil theme must fail")
	}
}
