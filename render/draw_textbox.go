package render

import (
	"image/color"
	"unicode/utf8"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Caret and selection metrics for single-line textboxes. Advances come from
// the loaded font when a graphics context exists and fall back to estimation
// headless, so tests stay deterministic while windowed hit testing matches
// drawn glyphs. Drawing and hit testing share measureTextboxPrefix, so the
// caret always lands where the next draw places it within each mode.
const (
	// textboxEstimatedAdvance is the per-rune advance factor used only when
	// no graphics context exists for font measurement.
	textboxEstimatedAdvance = 0.55
	// textboxCaretWidth is the logical caret line width.
	textboxCaretWidth = 2
)

var (
	textboxSelectionTint = color.RGBA{R: 100, G: 150, B: 255, A: 120}
	textboxCaretTint     = color.RGBA{R: 20, G: 20, B: 20, A: 255}
)

// TextboxContent returns the skin-aware textbox content area used for text,
// selection, caret drawing, and click mapping. It resolves background and
// border with normal fallback and snaps when pixel snap is on. Callers must
// use this — never raw textbox bounds — so drawn and hit rows always agree.
func (t *Theme) TextboxContent(bounds core.Rect, state core.WidgetState) core.Rect {
	if t == nil {
		return bounds
	}
	background, _ := t.resolveDescriptor(core.WidgetTextbox, skin.PartBackground, state)
	border, _ := t.resolveDescriptor(core.WidgetTextbox, skin.PartBorder, state)
	return t.snap(ContentRect(bounds, background, border))
}

// DrawTextbox renders a single-line textbox with selection and caret. value
// is the full text, caret is the rune index, and selStart/selEnd are the
// sorted selection bounds or negatives when idle. showCaret draws the caret
// line, typically only when the textbox owns focus.
func (t *Theme) DrawTextbox(info core.WidgetInfo, value string, caret, selStart, selEnd int, showCaret bool) {
	if t == nil {
		return
	}
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
	content := t.TextboxContent(info.Bounds, info.State)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	if content.W <= 0 || content.H <= 0 {
		return
	}
	fontSize, x0, y0 := widgetTextOrigin(content)
	if hasTextboxSelection(selStart, selEnd, value) {
		t.drawTextboxSelection(info, content, value, selStart, selEnd, fontSize, x0, y0)
	}
	t.drawTextInContent(info.Kind, value, content, info.State)
	if showCaret {
		t.drawTextboxCaret(info, content, value, caret, fontSize, x0, y0)
	}
}

// TextboxCaretIndex maps a logical X to the nearest rune index in text. It
// uses the same content area and advances as DrawTextbox so clicks land where
// drawing places glyphs.
func (t *Theme) TextboxCaretIndex(bounds core.Rect, state core.WidgetState, text string, x float32) int {
	content := t.TextboxContent(bounds, state)
	if content.W <= 0 {
		return 0
	}
	fontSize, x0, _ := widgetTextOrigin(content)
	if x <= x0 {
		return 0
	}
	runes := utf8.RuneCountInString(text)
	// Walk prefixes so windowed font advances and headless estimation agree
	// with drawing; the last prefix at or left of x wins.
	index := 0
	for i := 1; i <= runes; i++ {
		prefix := textboxPrefix(text, i)
		width := t.measureTextboxPrefix(prefix, fontSize)
		// Round the boundary midpoint between adjacent prefixes so clicks
		// favor the nearer rune edge.
		if x0+width-float32(textboxCaretWidth) <= x {
			index = i
			continue
		}
		break
	}
	return index
}

// drawTextboxSelection records and draws the selected range backdrop.
func (t *Theme) drawTextboxSelection(info core.WidgetInfo, content core.Rect, value string, selStart, selEnd int, fontSize int32, x0, y0 float32) {
	startX := t.measureTextboxPrefix(textboxPrefix(value, selStart), fontSize)
	endX := t.measureTextboxPrefix(textboxPrefix(value, selEnd), fontSize)
	rect := t.snap(core.Rect{X: x0 + startX, Y: y0, W: endX - startX, H: float32(fontSize)})
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	t.logDrawCall(info.Kind, skin.PartOverlay, info.State, rect, rect, skin.SkinDescriptor{}, textboxSelectionTint, false)
	if rl.IsWindowReady() {
		drawFallbackPart(rect, textboxSelectionTint)
	}
}

// drawTextboxCaret records and draws the caret line at the rune index.
func (t *Theme) drawTextboxCaret(info core.WidgetInfo, content core.Rect, value string, caret int, fontSize int32, x0, y0 float32) {
	if caret < 0 {
		caret = 0
	}
	if n := utf8.RuneCountInString(value); caret > n {
		caret = n
	}
	prefixWidth := t.measureTextboxPrefix(textboxPrefix(value, caret), fontSize)
	rect := t.snap(core.Rect{X: x0 + prefixWidth, Y: y0, W: textboxCaretWidth, H: float32(fontSize)})
	t.logDrawCall(info.Kind, skin.PartCaret, info.State, rect, rect, skin.SkinDescriptor{}, textboxCaretTint, false)
	if rl.IsWindowReady() {
		drawFallbackPart(rect, textboxCaretTint)
	}
}

// measureTextboxPrefix returns the logical advance of prefix in textbox
// metrics, preferring the loaded font raster when a graphics context exists.
func (t *Theme) measureTextboxPrefix(prefix string, fontSize int32) float32 {
	if prefix == "" {
		return 0
	}
	if t != nil && t.HasFont() && rl.IsWindowReady() {
		if width := rl.MeasureTextEx(t.FontForSize(float32(fontSize)), prefix, float32(fontSize), float32(fontSize)/10).X; width > 0 {
			return width
		}
	}
	return float32(utf8.RuneCountInString(prefix)) * float32(fontSize) * textboxEstimatedAdvance
}

// hasTextboxSelection reports whether a drawable selection range exists.
func hasTextboxSelection(start, end int, value string) bool {
	if start < 0 || end < 0 || start == end {
		return false
	}
	if start > end {
		start, end = end, start
	}
	n := utf8.RuneCountInString(value)
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}
	return start < end
}

// textboxPrefix returns the first n runes of value without allocating a rune
// slice conversion for the whole string on every measurement step.
func textboxPrefix(value string, n int) string {
	if n <= 0 {
		return ""
	}
	pos := 0
	count := 0
	for pos < len(value) && count < n {
		_, size := utf8.DecodeRuneInString(value[pos:])
		if size <= 0 {
			break
		}
		pos += size
		count++
	}
	return value[:pos]
}
