package render

import (
	"strings"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Rich-text metrics for v1. Wrapping is word-based; advances come from the
// loaded font when a graphics context exists and fall back to estimation
// headless, so tests stay deterministic while windowed hit testing matches
// drawn glyphs. Font size is fixed; per-segment sizes are deferred.
const (
	// RichFontSize is the fixed message text size in logical pixels.
	// Requested ~1.5x nominal (see drawTextInContent) for a true ~13px EM.
	RichFontSize = 20
	// RichLineHeight is the fixed vertical advance per wrapped line.
	RichLineHeight = 24
	// richTextSpacing is the inter-glyph spacing shared by measurement and
	// drawing so laid-out fragments match drawn words.
	richTextSpacing = 2.0
	// richEstimatedAdvance is the per-rune advance factor used only when no
	// graphics context exists for font measurement.
	richEstimatedAdvance = 0.55
)

// RichSpanLayout is one laid-out single-line fragment of a message segment.
// A segment split by wrapping yields one fragment per wrapped row; all share
// the segment index so hover, press, and tooltip state stay per segment.
type RichSpanLayout struct {
	// Segment is the index into the laid-out segment slice.
	Segment int
	// Bounds is the logical fragment bounds used for drawing and hit testing.
	Bounds core.Rect
	// Text is the fragment word run drawn inside Bounds.
	Text string
	// Link is the fragment's link; its kind is LinkNone for plain text.
	Link core.Link
}

// Linked reports whether the fragment belongs to a linked segment.
func (s RichSpanLayout) Linked() bool { return s.Link.Kind != core.LinkNone }

// RichContent returns the skin-aware message content area used for wrapping,
// drawing, and hit testing. It resolves RichText background and border with
// normal fallback and snaps when pixel snap is on. Callers must use this —
// never raw message bounds — so wrapped, drawn, and hit rows always agree.
// It is safe on a nil theme, where it returns the normalized bounds.
func (t *Theme) RichContent(bounds core.Rect, state core.WidgetState) core.Rect {
	if t == nil {
		return bounds
	}
	background, _ := t.resolveDescriptor(core.WidgetRichText, skin.PartBackground, state)
	border, _ := t.resolveDescriptor(core.WidgetRichText, skin.PartBorder, state)
	return t.snap(ContentRect(bounds, background, border))
}

// LayoutRichSpans wraps segments into single-line fragments inside the
// skin-aware content area. Vertically overflowing lines are clipped so hit
// testing never reaches invisible text.
func (t *Theme) LayoutRichSpans(bounds core.Rect, segments []core.RichSegment, state core.WidgetState) []RichSpanLayout {
	return t.layoutRichSpans(t.RichContent(bounds, state), segments)
}

// RichContentHeight returns the outer height for width that fits every
// wrapped line plus skin insets, so chat frames can size to their content.
func (t *Theme) RichContentHeight(width float32, segments []core.RichSegment) float32 {
	if width <= 0 {
		return 0
	}
	const huge = float32(1000000)
	content := t.RichContent(core.Rect{W: width, H: huge}, core.StateNormal)
	extent := float32(0)
	for _, span := range t.layoutRichSpans(content, segments) {
		if bottom := span.Bounds.Y + span.Bounds.H - content.Y; bottom > extent {
			extent = bottom
		}
	}
	return content.Y + extent + (huge - (content.Y + content.H))
}

// RichSpanAt returns the topmost fragment containing pos. Later fragments
// win so shared word edges resolve deterministically.
func RichSpanAt(spans []RichSpanLayout, pos core.Vec2) (RichSpanLayout, bool) {
	for index := len(spans) - 1; index >= 0; index-- {
		bounds := spans[index].Bounds
		if pos.X >= bounds.X && pos.X <= bounds.X+bounds.W && pos.Y >= bounds.Y && pos.Y <= bounds.Y+bounds.H {
			return spans[index], true
		}
	}
	return RichSpanLayout{}, false
}

// layoutRichSpans wraps segments into fragments inside content. Adjacent
// segments flow without forced breaks; explicit newlines break lines.
func (t *Theme) layoutRichSpans(content core.Rect, segments []core.RichSegment) []RichSpanLayout {
	var spans []RichSpanLayout
	if content.W <= 0 || content.H <= 0 {
		return nil
	}
	right := content.X + content.W
	below := content.Y + content.H
	x, y := content.X, content.Y
	breakLine := func() {
		x, y = content.X, y+RichLineHeight
	}
	space := t.richSpaceAdvance()
	for index, segment := range segments {
		paragraphs := strings.Split(segment.Text, "\n")
		for first, paragraph := range paragraphs {
			if first > 0 {
				breakLine()
			}
			for _, word := range strings.Fields(paragraph) {
				width := t.measureRichWord(word)
				if x > content.X && x+width > right {
					breakLine()
				}
				if y+RichLineHeight > below {
					return spans
				}
				spans = append(spans, RichSpanLayout{
					Segment: index,
					Bounds:  core.Rect{X: x, Y: y, W: width, H: RichLineHeight},
					Text:    word,
					Link:    segment.Link,
				})
				x += width + space
			}
		}
	}
	return spans
}

// measureRichWord returns the logical advance of word in message metrics,
// preferring the loaded font raster when a graphics context exists.
func (t *Theme) measureRichWord(word string) float32 {
	if t != nil && t.HasFont() && rl.IsWindowReady() {
		if width := rl.MeasureTextEx(t.FontForSize(RichFontSize), word, RichFontSize, richTextSpacing).X; width > 0 {
			return width
		}
	}
	return float32(len([]rune(word))) * float32(RichFontSize) * richEstimatedAdvance
}

// richSpaceAdvance returns the in-string space advance for message metrics.
// A lone space measures narrower than the same space inside a string, so
// the advance is derived from a spaced pair; headless falls back to the
// same estimation as words so tests stay deterministic.
func (t *Theme) richSpaceAdvance() float32 {
	if t != nil && t.HasFont() && rl.IsWindowReady() {
		font := t.FontForSize(RichFontSize)
		single := rl.MeasureTextEx(font, "a", RichFontSize, richTextSpacing).X
		pair := rl.MeasureTextEx(font, "a a", RichFontSize, richTextSpacing).X
		if space := pair - 2*single; space > 0 {
			return space
		}
	}
	return float32(RichFontSize) * richEstimatedAdvance
}
