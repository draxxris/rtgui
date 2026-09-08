package render

import (
	"unicode/utf8"

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
	// richTabSpaces counts one tab stop as four spaces for layout advances.
	richTabSpaces = 4
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
	// HasColor selects Color over theme defaults and link blue.
	HasColor bool
	// Color is the fragment color used only when HasColor is true.
	Color core.Color
}

// Linked reports whether the fragment belongs to a linked segment.
func (s RichSpanLayout) Linked() bool { return s.Link.Kind != core.LinkNone }

// RichSegmentSource is the indexed segment view a layout cache reads.
// widgets.RichText satisfies it without render importing widgets; any test
// fake with the same three methods works. Revision must change on every
// content change so steady-state updates skip deep compares allocation-free.
type RichSegmentSource interface {
	RichSegmentCount() int
	RichSegmentAt(index int) (core.RichSegment, bool)
	RichRevision() uint64
}

// RichLayoutCache is a reusable per-widget rich-text layout owned by the UI.
// One cache serves one widget: Update rebuilds only when bounds, skin-aware
// content, font readiness, theme text revision, or segment content changes.
// State is used only to resolve content insets, never as a raw key, so hover
// transitions with identical insets never rebuild. Steady-state
// Update/Spans/SpanAt calls allocate nothing after backing stores warm up.
// Truncated span and segment tails are zeroed on shrink so old strings never
// leak through reused backing arrays.
type RichLayoutCache struct {
	spans     []RichSpanLayout
	segCache  []core.RichSegment
	bounds    core.Rect
	content   core.Rect
	source    RichSegmentSource
	sourceRev uint64
	sourceLen int
	textRev   uint64
	fontReady bool
	indexed   bool
	ready     bool
}

// Spans returns the cached fragments for drawing and hit testing. The slice
// aliases cache storage and must be treated as read-only until the next
// Update; it never allocates.
func (c *RichLayoutCache) Spans() []RichSpanLayout {
	if c == nil {
		return nil
	}
	return c.spans
}

// Len returns the cached fragment count without allocating.
func (c *RichLayoutCache) Len() int {
	if c == nil {
		return 0
	}
	return len(c.spans)
}

// Content returns the skin-aware content rect from the last Update.
func (c *RichLayoutCache) Content() core.Rect {
	if c == nil {
		return core.Rect{}
	}
	return c.content
}

// SpanAt returns the topmost cached fragment containing pos for hit testing.
// Later fragments win so shared word edges resolve deterministically. The
// scan touches only cached storage and never allocates.
func (c *RichLayoutCache) SpanAt(pos core.Vec2) (RichSpanLayout, bool) {
	if c == nil {
		return RichSpanLayout{}, false
	}
	return RichSpanAt(c.spans, pos)
}

// Invalidate drops cached fragments and keys while retaining backing capacity.
// Truncated tails are zeroed so shortened layouts release string references.
// The next Update always rebuilds.
func (c *RichLayoutCache) Invalidate() {
	if c == nil {
		return
	}
	clear(c.spans)
	c.spans = c.spans[:0]
	clear(c.segCache)
	c.segCache = c.segCache[:0]
	c.bounds = core.Rect{}
	c.content = core.Rect{}
	c.source = nil
	c.sourceRev = 0
	c.sourceLen = 0
	c.textRev = 0
	c.fontReady = false
	c.indexed = false
	c.ready = false
}

// Update rebuilds the cache from an indexed source without defensive copies.
// The theme text revision is read internally, matching RichTooltipCache, so
// font swaps and skin changes always invalidate even with matching insets.
// State resolves content insets only and never forces a rebuild by itself.
// It reports true only when the layout was rebuilt.
func (c *RichLayoutCache) Update(theme *Theme, bounds core.Rect, source RichSegmentSource, state core.WidgetState) bool {
	if c == nil {
		return false
	}
	content := themeRichContent(theme, bounds, state)
	ready := richFontReady(theme)
	revision := richTextRevision(theme)
	count, rev := richSourceKeys(source)
	if c.ready && c.indexed && c.source == source && c.bounds == bounds && c.content == content && c.fontReady == ready && c.textRev == revision && c.sourceRev == rev && c.sourceLen == count {
		return false
	}
	oldLen := len(c.spans)
	space := themeRichSpace(theme)
	cursor := richCursor(content, space)
	c.spans = layoutRichSourceInto(theme, cursor, count, source, c.spans[:0])
	clearRichSpanTail(c.spans, oldLen)
	clear(c.segCache)
	c.segCache = c.segCache[:0]
	c.bounds = bounds
	c.content = content
	c.source = source
	c.sourceRev = rev
	c.sourceLen = count
	c.textRev = revision
	c.fontReady = ready
	c.indexed = true
	c.ready = true
	return true
}

// UpdateSegments rebuilds the cache from a segment slice for tests and legacy
// callers. Segments are copied into reused storage only on change, and the
// theme text revision is read internally, so the steady-state path stays
// allocation-free after warmup. It reports true only when rebuilt.
func (c *RichLayoutCache) UpdateSegments(theme *Theme, bounds core.Rect, segments []core.RichSegment, state core.WidgetState) bool {
	if c == nil {
		return false
	}
	content := themeRichContent(theme, bounds, state)
	ready := richFontReady(theme)
	revision := richTextRevision(theme)
	if c.ready && !c.indexed && c.bounds == bounds && c.content == content && c.fontReady == ready && c.textRev == revision && equalCachedRichSegments(c.segCache, segments) {
		return false
	}
	oldSpans := len(c.spans)
	oldSegs := len(c.segCache)
	c.segCache = copyCachedRichSegments(c.segCache, segments, oldSegs)
	space := themeRichSpace(theme)
	cursor := richCursor(content, space)
	cached := c.segCache
	c.spans = layoutRichSliceInto(theme, cursor, cached, c.spans[:0])
	clearRichSpanTail(c.spans, oldSpans)
	c.bounds = bounds
	c.content = content
	c.source = nil
	c.sourceRev = 0
	c.sourceLen = len(segments)
	c.textRev = revision
	c.fontReady = ready
	c.indexed = false
	c.ready = true
	return true
}

// RichContent returns the skin-aware message content area used for wrapping,
// drawing, and hit testing. It resolves RichText background and border with
// normal fallback and snaps when pixel snap is on. Callers must use this —
// never raw message bounds — so wrapped, drawn, and hit rows always agree.
// It is safe on a nil theme, where it returns the normalized bounds.
func (t *Theme) RichContent(bounds core.Rect, state core.WidgetState) core.Rect {
	return themeRichContent(t, bounds, state)
}

// LayoutRichSpans wraps segments into single-line fragments inside the
// skin-aware content area. Vertically overflowing lines are clipped so hit
// testing never reaches invisible text. It stays for external tests; the UI
// should use RichLayoutCache plus DrawRichTextLayout instead.
func (t *Theme) LayoutRichSpans(bounds core.Rect, segments []core.RichSegment, state core.WidgetState) []RichSpanLayout {
	cursor := richCursor(themeRichContent(t, bounds, state), themeRichSpace(t))
	return layoutRichSliceInto(t, cursor, segments, nil)
}

// RichContentHeight returns the outer height for width that fits every
// wrapped line plus skin insets, so chat frames can size to their content.
func (t *Theme) RichContentHeight(width float32, segments []core.RichSegment) float32 {
	if width <= 0 {
		return 0
	}
	const huge = float32(1000000)
	content := themeRichContent(t, core.Rect{W: width, H: huge}, core.StateNormal)
	extent := float32(0)
	cursor := richCursor(content, themeRichSpace(t))
	for _, span := range layoutRichSliceInto(t, cursor, segments, nil) {
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

// themeRichContent resolves the skin-aware content rect without allocating.
// It is nil-safe so caches keep working headless and with a nil theme.
func themeRichContent(theme *Theme, bounds core.Rect, state core.WidgetState) core.Rect {
	if theme == nil {
		return bounds
	}
	background, _ := theme.resolveDescriptor(core.WidgetRichText, skin.PartBackground, state)
	border, _ := theme.resolveDescriptor(core.WidgetRichText, skin.PartBorder, state)
	return theme.snap(ContentRect(bounds, background, border))
}

// richFontReady reports the font path used by measurement without allocating.
// Both font presence and window readiness matter because measurement falls
// back to estimation headless.
func richFontReady(theme *Theme) bool {
	return theme != nil && rl.IsWindowReady()
}

// richTextRevision returns the nil-safe theme text revision for cache keys,
// matching RichTooltipCache so font and skin edits always invalidate.
func richTextRevision(theme *Theme) uint64 {
	if theme == nil {
		return 0
	}
	return theme.TextRevision()
}

// themeRichSpace returns the cached space advance for one layout rebuild.
func themeRichSpace(theme *Theme) float32 {
	if theme == nil {
		return float32(RichFontSize) * richEstimatedAdvance
	}
	return theme.richSpaceAdvance()
}

// richSourceKeys reads count and revision from an indexed source safely.
func richSourceKeys(source RichSegmentSource) (int, uint64) {
	if source == nil {
		return 0, 0
	}
	return source.RichSegmentCount(), source.RichRevision()
}

// layoutRichSliceInto wraps a segment slice into reused span storage with an
// allocation-free word scanner. Newlines force breaks, runs of spaces and tabs
// advance by the cached space width, and overlong words split by runes across
// rows so CJK, URLs, and hashes never bleed past content.
func layoutRichSliceInto(theme *Theme, cursor richCursorState, segments []core.RichSegment, reuse []RichSpanLayout) []RichSpanLayout {
	if cursor.content.W <= 0 || cursor.content.H <= 0 {
		return reuse[:0]
	}
	for index := range segments {
		segment := segments[index]
		reuse = layoutRichTextInto(theme, &cursor, reuse, index, segment.Text, segment.Link, segment.HasColor, segment.Color)
		if cursor.overflow {
			break
		}
	}
	return reuse
}

// layoutRichSourceInto wraps an indexed source into reused span storage with
// the same scanner as the slice path. Missing indexes are skipped so a source
// that shrinks mid-rebuild cannot trap the layout.
func layoutRichSourceInto(theme *Theme, cursor richCursorState, count int, source RichSegmentSource, reuse []RichSpanLayout) []RichSpanLayout {
	if cursor.content.W <= 0 || cursor.content.H <= 0 || count <= 0 {
		return reuse[:0]
	}
	for index := 0; index < count; index++ {
		segment, ok := sourceRichSegmentAt(source, index)
		if !ok {
			continue
		}
		reuse = layoutRichTextInto(theme, &cursor, reuse, index, segment.Text, segment.Link, segment.HasColor, segment.Color)
		if cursor.overflow {
			break
		}
	}
	return reuse
}

// sourceRichSegmentAt reads one indexed segment without allocating a closure.
// A nil source or out-of-range index reports false so rebuilds stay safe.
func sourceRichSegmentAt(source RichSegmentSource, index int) (core.RichSegment, bool) {
	if source == nil {
		return core.RichSegment{}, false
	}
	return source.RichSegmentAt(index)
}

// richCursorState tracks the pen and clipping edges across one rebuild.
// wrapped distinguishes automatic wrap rows (leading spaces skipped) from
// explicit newline rows (indentation preserved).
type richCursorState struct {
	content  core.Rect
	right    float32
	below    float32
	space    float32
	x        float32
	y        float32
	overflow bool
	wrapped  bool
}

// richCursor seeds the pen at the content origin with clipping edges.
// The value stays on the caller stack; only rebuilds use it.
func richCursor(content core.Rect, space float32) richCursorState {
	return richCursorState{content: content, right: content.X + content.W, below: content.Y + content.H, space: space, x: content.X, y: content.Y}
}

// layoutRichTextInto scans one segment without Split/Fields allocations and
// appends its word fragments into reuse. Spaces advance the pen, newlines
// break rows, and vertical overflow latches so callers stop cleanly.
func layoutRichTextInto(theme *Theme, cursor *richCursorState, reuse []RichSpanLayout, index int, text string, link core.Link, hasColor bool, tint core.Color) []RichSpanLayout {
	pos := 0
	for pos < len(text) {
		byteValue := text[pos]
		if byteValue == '\n' {
			cursor.newline()
			if cursor.overflow {
				return reuse
			}
			pos++
			continue
		}
		if isRichSpaceByte(byteValue) {
			cursor.advanceSpace(byteValue)
			pos++
			continue
		}
		start := pos
		for pos < len(text) && !isRichSpaceByte(text[pos]) && text[pos] != '\n' {
			pos++
		}
		reuse = cursor.appendWord(theme, reuse, index, text[start:pos], link, hasColor, tint)
		if cursor.overflow {
			return reuse
		}
	}
	return reuse
}

// newline moves the pen to an explicit-break row preserving indentation.
// Wrapped-row skipping is cleared so intentional indents survive newlines.
func (c *richCursorState) newline() {
	if c == nil {
		return
	}
	c.x = c.content.X
	c.y += RichLineHeight
	c.wrapped = false
	if c.y+RichLineHeight > c.below {
		c.overflow = true
	}
}

// wrapLine moves the pen to an automatic-wrap row that skips carried spaces.
// Only word wrapping sets this; explicit newlines preserve indentation.
func (c *richCursorState) wrapLine() {
	if c == nil {
		return
	}
	c.x = c.content.X
	c.y += RichLineHeight
	c.wrapped = true
	if c.y+RichLineHeight > c.below {
		c.overflow = true
	}
}

// advanceSpace moves the pen for one space-class byte without allocating.
// Spaces carried onto automatic-wrap rows are skipped, while indentation
// after explicit newlines and at text start is preserved.
func (c *richCursorState) advanceSpace(byteValue byte) {
	if c == nil || c.wrapped {
		return
	}
	if byteValue == '\t' {
		c.x += float32(richTabSpaces) * c.space
		return
	}
	c.x += c.space
}

// appendWord measures one word and appends its fragment, wrapping first when
// needed. Overlong words split by runes across rows so unspaced scripts and
// long tokens never bleed; each emitted row clears the wrap-skip flag.
func (c *richCursorState) appendWord(theme *Theme, reuse []RichSpanLayout, index int, word string, link core.Link, hasColor bool, tint core.Color) []RichSpanLayout {
	if word == "" {
		return reuse
	}
	if theme.measureRichWord(word) > c.content.W && c.content.W > 0 {
		return c.appendLongWord(theme, reuse, index, word, link, hasColor, tint)
	}
	width := theme.measureRichWord(word)
	if c.x > c.content.X && c.x+width > c.right {
		c.wrapLine()
		if c.overflow {
			return reuse
		}
	}
	if c.y+RichLineHeight > c.below {
		c.overflow = true
		return reuse
	}
	width = clampRichWordWidth(width, c.x, c.right, c.content.W)
	reuse = append(reuse, RichSpanLayout{
		Segment:  index,
		Bounds:   core.Rect{X: c.x, Y: c.y, W: width, H: RichLineHeight},
		Text:     word,
		Link:     link,
		HasColor: hasColor,
		Color:    tint,
	})
	c.x += width
	c.wrapped = false
	return reuse
}

// appendLongWord splits an overlong word by runes into fitting fragments.
// Single-rune overflow still emits with clamped geometry to guarantee
// progress on pathologically narrow content.
func (c *richCursorState) appendLongWord(theme *Theme, reuse []RichSpanLayout, index int, word string, link core.Link, hasColor bool, tint core.Color) []RichSpanLayout {
	start := 0
	for start < len(word) {
		if c.x != c.content.X {
			c.wrapLine()
			if c.overflow {
				return reuse
			}
		}
		end := fitRichPrefix(theme, word[start:], c.content.W)
		if end <= 0 {
			end = firstRuneLen(word[start:])
		}
		chunk := word[start : start+end]
		width := clampRichWordWidth(theme.measureRichWord(chunk), c.x, c.right, c.content.W)
		if c.y+RichLineHeight > c.below {
			c.overflow = true
			return reuse
		}
		reuse = append(reuse, RichSpanLayout{
			Segment:  index,
			Bounds:   core.Rect{X: c.x, Y: c.y, W: width, H: RichLineHeight},
			Text:     chunk,
			Link:     link,
			HasColor: hasColor,
			Color:    tint,
		})
		c.x += width
		c.wrapped = false
		start += end
		if c.overflow {
			return reuse
		}
		// Full-width chunks always continue on a fresh row.
		if start < len(word) {
			c.wrapLine()
			if c.overflow {
				return reuse
			}
			c.x = c.content.X
		}
	}
	return reuse
}

// fitRichPrefix returns the longest leading byte prefix of word fitting width.
// It advances by whole runes and measures rebuild-only, never steady-state.
func fitRichPrefix(theme *Theme, word string, width float32) int {
	end := 0
	for end < len(word) {
		runeLen := firstRuneLen(word[end:])
		next := end + runeLen
		if theme.measureRichWord(word[:next]) > width {
			break
		}
		end = next
	}
	return end
}

// firstRuneLen returns the byte length of the first rune in text without
// allocating. Invalid encodings advance one byte to guarantee progress.
func firstRuneLen(text string) int {
	if text == "" {
		return 0
	}
	_, width := utf8.DecodeRuneInString(text)
	if width <= 0 || width > len(text) {
		return 1
	}
	return width
}

// clampRichWordWidth keeps fragment geometry inside the content row without
// allocating. Widths larger than content are handled by rune splitting; this
// clamps only rounding tails on the current row.
func clampRichWordWidth(width, x, right, contentW float32) float32 {
	if contentW <= 0 {
		return 0
	}
	if width > contentW {
		return contentW
	}
	if x+width > right {
		width = right - x
		if width < 0 {
			return 0
		}
	}
	return width
}

// isRichSpaceByte reports ASCII horizontal whitespace handled as advances.
func isRichSpaceByte(value byte) bool {
	switch value {
	case ' ', '\t', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}

// measureRichWord returns the logical advance of word in message metrics,
// preferring the loaded font raster when a graphics context exists.
func (t *Theme) measureRichWord(word string) float32 {
	if word == "" {
		return 0
	}
	if t != nil && t.HasFont() && rl.IsWindowReady() {
		if width := rl.MeasureTextEx(t.FontForSize(RichFontSize), word, RichFontSize, richTextSpacing).X; width > 0 {
			return width
		}
	}
	if t != nil && rl.IsWindowReady() {
		return float32(rl.MeasureText(word, RichFontSize))
	}
	return float32(utf8.RuneCountInString(word)) * float32(RichFontSize) * richEstimatedAdvance
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
	if t != nil && rl.IsWindowReady() {
		return float32(rl.MeasureText("a a", RichFontSize) - 2*rl.MeasureText("a", RichFontSize))
	}
	return float32(RichFontSize) * richEstimatedAdvance
}

// equalCachedRichSegments compares cached and incoming segments field by field
// without allocating. Length mismatch short-circuits before any compare.
func equalCachedRichSegments(left, right []core.RichSegment) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// copyCachedRichSegments copies segments into reused cache storage and zeroes
// truncated tails. It allocates only when cached capacity is short.
func copyCachedRichSegments(cached, segments []core.RichSegment, oldLen int) []core.RichSegment {
	if len(segments) == 0 {
		clear(cached)
		return cached[:0]
	}
	if cap(cached) < len(segments) {
		out := make([]core.RichSegment, len(segments))
		copy(out, segments)
		return out
	}
	out := cached[:len(segments)]
	copy(out, segments)
	if len(segments) < oldLen && oldLen <= cap(cached) {
		clear(cached[len(segments):oldLen])
	}
	return out
}

// clearRichSpanTail zeroes entries truncated by a shrink so reused backing
// never retains old fragment strings or links.
func clearRichSpanTail(current []RichSpanLayout, oldLen int) {
	if len(current) >= oldLen || cap(current) < oldLen {
		return
	}
	clear(current[len(current):oldLen])
}
