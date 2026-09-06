package render

import (
	"testing"
	"unicode/utf8"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

// TestTextboxContentUsesBackgroundAndBorder verifies skin insets drive layout.
func TestTextboxContentUsesBackgroundAndBorder(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	bounds := core.Rect{X: 10, Y: 20, W: 100, H: 40}
	if got := theme.TextboxContent(bounds, core.StateNormal); got != bounds {
		t.Fatalf("unskinned content = %+v", got)
	}
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTextbox, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		PaddingLeft: 4, PaddingTop: 3, PaddingRight: 5, PaddingBottom: 6,
	})
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTextbox, Part: skin.PartBorder, State: core.StateNormal}, skin.SkinDescriptor{
		HasNinePatch: true, NinePatch: skin.NinePatch{Left: 8, Top: 8, Right: 8, Bottom: 8},
	})
	want := core.Rect{X: 18, Y: 28, W: 84, H: 24}
	if got := theme.TextboxContent(bounds, core.StateNormal); got != want {
		t.Fatalf("skinned content = %+v want %+v", got, want)
	}
	var nilTheme *Theme
	if got := nilTheme.TextboxContent(bounds, core.StateNormal); got != bounds {
		t.Fatalf("nil theme content = %+v", got)
	}
}

// TestDrawTextboxShowsCaretAndSelection verifies caret, highlight, and text parts.
func TestDrawTextboxShowsCaretAndSelection(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder, _ := NewDrawRecorder(16)
	theme.SetDrawRecorder(recorder)
	info := core.WidgetInfo{Name: "field", Bounds: core.Rect{X: 10, Y: 10, W: 200, H: 40}, Kind: core.WidgetTextbox, State: core.StateFocused}
	theme.BeginFrame()
	theme.DrawTextbox(info, "abcd", 2, 1, 3, true)
	assertTextboxParts(t, recorder.Calls(), true, true, true, true)
	if got := recorder.LastWidgetInfo(); got != info {
		t.Fatalf("last widget = %+v", got)
	}
}

// TestDrawTextboxHidesIdle verifies idle boxes draw no highlight or caret.
func TestDrawTextboxHidesIdle(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder, _ := NewDrawRecorder(16)
	theme.SetDrawRecorder(recorder)
	info := core.WidgetInfo{Name: "field", Bounds: core.Rect{X: 10, Y: 10, W: 200, H: 40}, Kind: core.WidgetTextbox, State: core.StateFocused}
	theme.BeginFrame()
	theme.DrawTextbox(info, "abcd", 4, -1, -1, false)
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartOverlay || call.Part == skin.PartCaret {
			t.Fatalf("idle textbox must not draw selection/caret, got %+v", call)
		}
	}
	var nilTheme *Theme
	nilTheme.DrawTextbox(info, "abcd", 0, -1, -1, true)
}

// assertTextboxParts checks background, highlight, text, and caret presence.
func assertTextboxParts(t *testing.T, calls []DrawCall, wantBackground, wantOverlay, wantText, wantCaret bool) {
	t.Helper()
	seenBackground, seenOverlay, seenText, seenCaret := false, false, false, false
	for _, call := range calls {
		switch call.Part {
		case skin.PartBackground:
			seenBackground = true
		case skin.PartOverlay:
			seenOverlay = true
			if call.Fallback {
				t.Fatalf("selection must draw, got %+v", call)
			}
		case skin.PartText:
			seenText = true
		case skin.PartCaret:
			seenCaret = true
			if call.Fallback || call.Bounds.W != textboxCaretWidth {
				t.Fatalf("caret call = %+v", call)
			}
		}
	}
	if seenBackground != wantBackground || seenOverlay != wantOverlay || seenText != wantText || seenCaret != wantCaret {
		t.Fatalf("parts bg=%v sel=%v text=%v caret=%v calls=%v", seenBackground, seenOverlay, seenText, seenCaret, calls)
	}
}

// TestTextboxCaretIndexMatchesDraw verifies click mapping round-trips.
func TestTextboxCaretIndexMatchesDraw(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	bounds := core.Rect{X: 10, Y: 10, W: 200, H: 40}
	if got := theme.TextboxCaretIndex(bounds, core.StateNormal, "abcd", 0); got != 0 {
		t.Fatalf("left edge index = %d", got)
	}
	if got := theme.TextboxCaretIndex(bounds, core.StateNormal, "abcd", 1000); got != 4 {
		t.Fatalf("far right index = %d", got)
	}
	assertTextboxMonotonic(t, theme, bounds)
}

// assertTextboxMonotonic checks prefix advances grow and edges resolve.
func assertTextboxMonotonic(t *testing.T, theme *Theme, bounds core.Rect) {
	t.Helper()
	content := theme.TextboxContent(bounds, core.StateNormal)
	_, x0, _ := widgetTextOrigin(content)
	w1 := theme.measureTextboxPrefix("a", 22)
	w2 := theme.measureTextboxPrefix("ab", 22)
	if w2 <= w1 {
		t.Fatalf("prefix widths not monotonic: %v/%v", w1, w2)
	}
	mid := x0 + (w1+w2)/2
	if got := theme.TextboxCaretIndex(bounds, core.StateNormal, "abcd", mid); got < 1 || got > 2 {
		t.Fatalf("mid index = %d", got)
	}
	if got := theme.TextboxCaretIndex(bounds, core.StateNormal, "abcd", x0+w1-0.5); got != 1 {
		t.Fatalf("first rune edge = %d", got)
	}
}

// TestTextboxPrefixHandlesMultibyte verifies rune-based slicing.
func TestTextboxPrefixHandlesMultibyte(t *testing.T) {
	if got := textboxPrefix("aé😀", 2); got != "aé" {
		t.Fatalf("prefix = %q", got)
	}
	if got := utf8.RuneCountInString("aé😀"); got != 3 {
		t.Fatalf("count = %d", got)
	}
}
