package render

import (
	"image/color"
	"strings"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Rich tooltip metrics for v1. Titles use widget-text sizing while the
// body reuses the fixed rich-text metrics so wrapped fragments match the
// chat layout engine. Widths and heights are logical pixels.
const (
	// RichTooltipMaxWidth caps tooltip content width in logical pixels.
	RichTooltipMaxWidth = TooltipMaxWidth
	// RichTooltipMaxHeight caps the complete outer popup height.
	RichTooltipMaxHeight = 320
	// RichTooltipTitleSize is the title text size in logical pixels.
	RichTooltipTitleSize = 22
	// RichTooltipTitleHeight is the vertical advance per title line.
	RichTooltipTitleHeight = 26
	// RichTooltipSubtitleSize is the subtitle text size in logical pixels.
	RichTooltipSubtitleSize = 20
	// RichTooltipSubtitleHeight is the vertical advance per subtitle line.
	RichTooltipSubtitleHeight = 24
	// RichTooltipIconSize bounds the icon box in logical pixels.
	RichTooltipIconSize = 32
	// RichTooltipIconGap separates the icon column from text.
	RichTooltipIconGap = 8
	// RichTooltipSectionGap separates title, subtitle, and body blocks.
	RichTooltipSectionGap = 4
)

// Rich tooltip draw colors for v1. The shell matches the dark popup
// styling, so body text is light; links default to pale blue unless the
// segment carries its own color.
var (
	richTooltipTitleText    = color.RGBA{R: 240, G: 247, B: 255, A: 255}
	richTooltipSubtitleText = color.RGBA{R: 170, G: 185, B: 205, A: 255}
	richTooltipBodyText     = color.RGBA{R: 230, G: 240, B: 255, A: 255}
	richTooltipLinkText     = color.RGBA{R: 140, G: 190, B: 255, A: 255}
)

// richTooltipRow is one cached word-wrapped title or subtitle line.
// Coordinates are relative to the content origin so cursor motion never
// re-wraps text; DrawRichTooltip adds the absolute content origin.
type richTooltipRow struct {
	// Text is the wrapped line drawn at the relative origin.
	Text string
	// X and Y are the origin relative to the content area.
	X, Y float32
}

// RichTooltipCache owns the laid-out rich tooltip for one hover target.
// Text is wrapped once into content-relative rows; cursor motion only
// translates the placed popup. Update rebuilds text only when data,
// viewport, width, padding, or the theme text and font state changes, so
// steady hover and draws allocate nothing after warmup.
type RichTooltipCache struct {
	// bounds is the absolute outer popup including shell insets.
	bounds core.Rect
	// content is the absolute text area inside the shell insets.
	content core.Rect
	// iconDest is the absolute icon box; hasIcon selects it.
	iconDest core.Rect
	// insetL and insetT are the shell origins; insetX and insetY span both sides.
	insetL, insetT, insetX, insetY float32
	// outerW and outerH are the placed popup sizes.
	outerW, outerH float32
	// hasIcon reports an effective icon survived layout.
	hasIcon bool
	// titleRows and subtitleRows are content-relative wrapped header lines.
	titleRows    []richTooltipRow
	subtitleRows []richTooltipRow
	// bodySpans are content-relative single-line body fragments.
	bodySpans []RichSpanLayout
	// data is the retained payload copy used for change detection.
	data core.RichTooltip
	// revision is the theme text revision of the cached layout.
	revision uint64
	// fontReady is the theme font readiness of the cached layout.
	fontReady bool
	// maxWidth, padding, anchor, and viewport are the layout keys.
	maxWidth float32
	padding  float32
	anchor   core.Vec2
	viewport core.Vec2
	// ready reports layout keys are synchronized, even for empty payloads.
	ready bool
	// valid reports the cached layout carries drawable rows.
	valid bool
}

// Bounds returns the cached outer popup including shell insets.
// It is zero before the first successful Update.
func (c *RichTooltipCache) Bounds() core.Rect {
	if c == nil {
		return core.Rect{}
	}
	return c.bounds
}

// Content returns the cached absolute text area inside the shell.
func (c *RichTooltipCache) Content() core.Rect {
	if c == nil {
		return core.Rect{}
	}
	return c.content
}

// IconDest returns the cached absolute icon box.
func (c *RichTooltipCache) IconDest() core.Rect {
	if c == nil {
		return core.Rect{}
	}
	return c.iconDest
}

// HasIcon reports whether the cached layout carries a drawable icon.
func (c *RichTooltipCache) HasIcon() bool { return c != nil && c.valid && c.hasIcon }

// Valid reports whether the cache holds drawable rows.
func (c *RichTooltipCache) Valid() bool { return c != nil && c.valid }

// TitleLineCount returns the cached wrapped title line count.
func (c *RichTooltipCache) TitleLineCount() int {
	if c == nil {
		return 0
	}
	return len(c.titleRows)
}

// BodySpanCount returns the cached body fragment count.
func (c *RichTooltipCache) BodySpanCount() int {
	if c == nil {
		return 0
	}
	return len(c.bodySpans)
}

// Invalidate flushes the cached layout and releases retained strings.
// Backing arrays are kept for reuse; the next Update rebuilds from scratch.
func (c *RichTooltipCache) Invalidate() {
	if c == nil {
		return
	}
	clearTooltipRows(c.titleRows)
	clearTooltipRows(c.subtitleRows)
	clearTooltipSpans(c.bodySpans)
	clearTooltipSegments(c.data.Segments)
	c.titleRows = c.titleRows[:0]
	c.subtitleRows = c.subtitleRows[:0]
	c.bodySpans = c.bodySpans[:0]
	c.data.Segments = c.data.Segments[:0]
	c.data.Title, c.data.Subtitle = "", ""
	c.data.Class = ""
	c.bounds, c.content, c.iconDest = core.Rect{}, core.Rect{}, core.Rect{}
	c.ready, c.valid, c.hasIcon = false, false, false
}

// Update rebuilds cached text when data, viewport, width, padding, or the
// theme text and font state changes, and translates the placed popup when
// only the anchor moves. It returns true when geometry changed. A nil
// theme lays out with headless estimation; a nil cache returns false.
func (c *RichTooltipCache) Update(theme *Theme, data core.RichTooltip, anchor, viewport core.Vec2, padding float32) bool {
	if c == nil {
		return false
	}
	maxWidth := data.DesiredWidth(RichTooltipMaxWidth)
	if maxWidth > RichTooltipMaxWidth {
		maxWidth = RichTooltipMaxWidth
	}
	if maxWidth <= 0 {
		maxWidth = RichTooltipMaxWidth
	}
	if padding <= 0 {
		padding = TooltipPadding
	}
	revision := richTooltipRevision(theme)
	fontReady := richTooltipFontReady(theme)
	if c.ready && c.layoutKeysEqual(data, viewport, maxWidth, padding, revision, fontReady) {
		if c.anchor == anchor {
			return false
		}
		c.reposition(anchor)
		return true
	}
	c.rebuild(theme, data, anchor, viewport, maxWidth, padding, revision, fontReady)
	return true
}

// layoutKeysEqual reports whether every text-affecting key matches.
// The anchor is intentionally excluded so cursor motion never re-wraps.
func (c *RichTooltipCache) layoutKeysEqual(data core.RichTooltip, viewport core.Vec2, maxWidth, padding float32, revision uint64, fontReady bool) bool {
	if c.revision != revision || c.fontReady != fontReady {
		return false
	}
	if c.maxWidth != maxWidth || c.padding != padding {
		return false
	}
	if c.viewport != viewport {
		return false
	}
	return core.RichTooltipDataEqual(c.data, data)
}

// reposition translates the placed popup to a new anchor without touching
// wrapped text. It touches no heap so cursor-following tooltips stay free.
func (c *RichTooltipCache) reposition(anchor core.Vec2) {
	c.anchor = anchor
	outer := richTooltipPlace(c.outerW, c.outerH, anchor, c.viewport)
	dx := outer.X - c.bounds.X
	dy := outer.Y - c.bounds.Y
	c.bounds = outer
	c.content.X += dx
	c.content.Y += dy
	c.iconDest.X += dx
	c.iconDest.Y += dy
}

// richTooltipRevision returns the nil-safe theme text revision.
func richTooltipRevision(theme *Theme) uint64 {
	if theme == nil {
		return 0
	}
	return theme.TextRevision()
}

// richTooltipFontReady reports whether font metrics (not estimation) drive
// measurement. Layouts made before window or font readiness rebuild once
// real metrics become available.
func richTooltipFontReady(theme *Theme) bool {
	return theme != nil && rl.IsWindowReady()
}

// rebuild wraps header and body text into content-relative rows, sizes the
// popup, and places it at the anchor. It is the only Update path that may
// allocate, and only when layout keys change.
func (c *RichTooltipCache) rebuild(theme *Theme, data core.RichTooltip, anchor, viewport core.Vec2, maxWidth, padding float32, revision uint64, fontReady bool) {
	left, top, right, bottom := richTooltipInsets(theme, data.Class, padding)
	contentW := richTooltipContentWidth(maxWidth, viewport.X, left+right)
	icon := effectiveRichTooltipIcon(data)
	textW := richTooltipTextWidth(contentW, icon)
	header := c.layoutHeader(theme, data, textW, icon)
	body := c.layoutBody(theme, data.Segments, textW, header, viewport.Y, top+bottom)
	fitted := richTooltipFittedWidth(contentW, body, icon)
	contentH := richTooltipContentHeight(header, body)
	outerW, outerH := richTooltipOuterSize(fitted, contentH, viewport, left+right, top+bottom)
	c.insetL, c.insetT, c.insetX, c.insetY = left, top, left+right, top+bottom
	c.outerW, c.outerH = outerW, outerH
	c.storeShell(richTooltipPlace(outerW, outerH, anchor, viewport), outerW, outerH, icon)
	c.storeRows(header, richTooltipTextOffset(icon))
	c.storeBody(header, body, richTooltipTextOffset(icon))
	c.retain(data)
	c.maxWidth, c.padding, c.anchor, c.viewport = maxWidth, padding, anchor, viewport
	c.revision, c.fontReady = revision, fontReady
	c.ready = true
	c.valid = richTooltipHasDrawable(outerW, outerH, header, body, c.hasIcon) && data.HasContent()
	if !c.valid {
		c.bounds = core.Rect{}
	}
}

// richTooltipInsets returns the WidgetTooltip shell insets for the theme.
// The class variant selects Tooltip.<class> when set. Padding fills every
// side when the skin carries no insets so the caller padding always shapes
// geometry; a nil theme yields padding throughout.
func richTooltipInsets(theme *Theme, class string, padding float32) (left, top, right, bottom float32) {
	if theme != nil {
		background, _ := theme.resolveDescriptor(core.WidgetTooltip, skin.PartBackground, core.StateNormal, class)
		border, _ := theme.resolveDescriptor(core.WidgetTooltip, skin.PartBorder, core.StateNormal, class)
		probe := ContentRect(core.Rect{W: 10000, H: 10000}, background, border)
		left, top = probe.X, probe.Y
		right, bottom = 10000-(probe.X+probe.W), 10000-(probe.Y+probe.H)
		if left+top+right+bottom > 0 {
			return left, top, right, bottom
		}
	}
	return padding, padding, padding, padding
}

// richTooltipContentWidth clamps the content width into the viewport.
func richTooltipContentWidth(maxWidth, viewportX, insetX float32) float32 {
	if viewportX > 0 && maxWidth+insetX > viewportX {
		maxWidth = viewportX - insetX
	}
	if maxWidth < 0 {
		return 0
	}
	return maxWidth
}

// effectiveRichTooltipIcon returns the laid-out icon box edge, or zero.
func effectiveRichTooltipIcon(data core.RichTooltip) float32 {
	if !data.HasIcon {
		return 0
	}
	if data.Icon.ID == 0 && (data.Icon.Width <= 0 || data.Icon.Height <= 0) {
		return 0
	}
	return RichTooltipIconSize
}

// richTooltipTextWidth reserves the icon column from the content width.
func richTooltipTextWidth(contentW, icon float32) float32 {
	if icon <= 0 {
		return contentW
	}
	textW := contentW - icon - RichTooltipIconGap
	if textW < 0 {
		return 0
	}
	return textW
}

// richTooltipHeader is the relative header geometry for one layout pass.
type richTooltipHeader struct {
	// title and subtitle are wrapped lines in a text-column origin.
	title    []string
	subtitle []string
	// titleH and subtitleH are the block heights including the gap.
	titleH, subtitleH float32
	// height is the header block height including a resident icon.
	height float32
}

// layoutHeader wraps title and subtitle rows inside the text column.
// Relative row origins start at the text column; storeRows absolutizes them.
func (c *RichTooltipCache) layoutHeader(theme *Theme, data core.RichTooltip, textW, icon float32) richTooltipHeader {
	var header richTooltipHeader
	if textW <= 0 {
		textW = 0
	}
	if data.Title != "" && textW > 0 {
		header.title = wrapTooltipLines(theme, data.Title, textW, RichTooltipTitleSize)
		header.titleH = float32(len(header.title)) * RichTooltipTitleHeight
	}
	if data.Subtitle != "" && textW > 0 {
		header.subtitle = wrapTooltipLines(theme, data.Subtitle, textW, RichTooltipSubtitleSize)
		header.subtitleH = float32(len(header.subtitle)) * RichTooltipSubtitleHeight
		if len(header.title) > 0 {
			header.subtitleH += RichTooltipSectionGap
		}
	}
	header.height = header.titleH + header.subtitleH
	if icon > 0 && icon > header.height {
		header.height = icon
	}
	return header
}

// richTooltipBodyPlan is the relative body geometry for one layout pass.
type richTooltipBodyPlan struct {
	// spans are fragments laid out in a text-column origin.
	spans []RichSpanLayout
	// widest is the widest laid-out row right edge in text coordinates.
	widest float32
	// height is the fragment block extent.
	height float32
	// gapped reports a section gap precedes the body block.
	gapped bool
}

// layoutBody wraps body segments under the header inside the text column.
// It uses the public LayoutRichSpans engine on change only, with bounds
// inflated to cancel the RichText shell insets so fragments land in the
// tooltip text column at a zero origin. storeBody relativizes them later.
// The body is clipped to the remaining outer height so popups never exceed
// the viewport; overflow relies on the draw-time clip.
func (c *RichTooltipCache) layoutBody(theme *Theme, segments []core.RichSegment, textW float32, header richTooltipHeader, viewportY, insetY float32) richTooltipBodyPlan {
	var body richTooltipBodyPlan
	for _, line := range header.title {
		body.widest = maxFloat(body.widest, richTooltipMeasure(theme, line, RichTooltipTitleSize))
	}
	for _, line := range header.subtitle {
		body.widest = maxFloat(body.widest, richTooltipMeasure(theme, line, RichTooltipSubtitleSize))
	}
	if len(segments) == 0 || textW <= 0 {
		return body
	}
	allowed := RichTooltipMaxHeight - insetY - header.height
	if viewportY > 0 && viewportY < RichTooltipMaxHeight {
		allowed = viewportY - insetY - header.height
	}
	if header.height > 0 {
		allowed -= RichTooltipSectionGap
		body.gapped = true
	}
	if allowed < RichLineHeight {
		body.gapped = false
		return body
	}
	spans := layoutRichTooltipBody(theme, segments, textW, allowed)
	extent := float32(0)
	for i := range spans {
		if right := spans[i].Bounds.X + spans[i].Bounds.W; right > body.widest {
			body.widest = right
		}
		if bottom := spans[i].Bounds.Y + spans[i].Bounds.H; bottom > extent {
			extent = bottom
		}
	}
	// Fragments share rows starting at the column origin, so the widest
	// row is the widest fragment right edge, never a single width.
	body.spans = spans
	body.height = extent
	return body
}

// layoutRichTooltipBody runs the shared rich-text wrap engine inside a
// zero-origin column of width textW and height allowed. RichText shell
// insets are measured through the public RichContent and cancelled by
// inflating the outer bounds, so this never depends on private layout.
func layoutRichTooltipBody(theme *Theme, segments []core.RichSegment, textW, allowed float32) []RichSpanLayout {
	left, top, right, bottom := richTextShellInsets(theme)
	outer := core.Rect{X: -left, Y: -top, W: textW + left + right, H: allowed + top + bottom}
	spans := theme.LayoutRichSpans(outer, segments, core.StateNormal)
	return spans
}

// richTextShellInsets measures the RichText background and border insets
// through the public RichContent helper. A nil theme yields zero insets.
func richTextShellInsets(theme *Theme) (left, top, right, bottom float32) {
	if theme == nil {
		return 0, 0, 0, 0
	}
	probe := theme.RichContent(core.Rect{W: 10000, H: 10000}, core.StateNormal)
	return probe.X, probe.Y, 10000 - (probe.X + probe.W), 10000 - (probe.Y + probe.H)
}

// richTooltipMeasure is the nil-safe line advance shared by header layout.
func richTooltipMeasure(theme *Theme, line string, size float32) float32 {
	if theme == nil {
		if line == "" || size <= 0 {
			return 0
		}
		return float32(len([]rune(line))) * size * 0.5
	}
	return theme.MeasureText(line, size, false)
}

// wrapTooltipLines packs text into measured lines that fit width.
// Explicit newlines break first; words pack greedily by theme advance.
func wrapTooltipLines(theme *Theme, text string, width, size float32) []string {
	if text == "" || width <= 0 {
		return nil
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := words[0]
		for _, word := range words[1:] {
			if richTooltipMeasure(theme, current+" "+word, size) <= width {
				current += " " + word
				continue
			}
			lines = append(lines, current)
			current = word
		}
		lines = append(lines, current)
	}
	return lines
}

// richTooltipFittedWidth shrinks the content column to the widest laid-out
// row without rewrapping; rows already fit, so shrinking never overflows.
// The icon column is added back whenever an icon shares the content width.
func richTooltipFittedWidth(contentW float32, body richTooltipBodyPlan, icon float32) float32 {
	if body.widest <= 0 || body.widest >= contentW {
		return contentW
	}
	fitted := body.widest
	if icon > 0 {
		fitted += icon + RichTooltipIconGap
		if fitted > contentW {
			return contentW
		}
	}
	return fitted
}

// richTooltipContentHeight sums the header block, the body extent, and the
// section gap between them when both blocks drew content.
func richTooltipContentHeight(header richTooltipHeader, body richTooltipBodyPlan) float32 {
	contentH := header.height + body.height
	if body.gapped && body.height > 0 {
		contentH += RichTooltipSectionGap
	}
	return contentH
}

// richTooltipOuterSize adds shell insets and clamps into the viewport.
func richTooltipOuterSize(fitted, contentH float32, viewport core.Vec2, insetX, insetY float32) (outerW, outerH float32) {
	outerW, outerH = fitted+insetX, contentH+insetY
	outerH = min(outerH, float32(RichTooltipMaxHeight))
	if viewport.X > 0 && outerW > viewport.X {
		outerW = viewport.X
	}
	if viewport.Y > 0 && outerH > viewport.Y {
		outerH = viewport.Y
	}
	return outerW, outerH
}

// richTooltipTextOffset returns the text-column origin past the icon.
func richTooltipTextOffset(icon float32) float32 {
	if icon > 0 {
		return icon + RichTooltipIconGap
	}
	return 0
}

// storeShell records the outer popup, content area, and icon box.
func (c *RichTooltipCache) storeShell(outer core.Rect, outerW, outerH, icon float32) {
	c.bounds = outer
	c.content = core.Rect{X: outer.X + c.insetL, Y: outer.Y + c.insetT, W: maxFloat(0, outerW-c.insetX), H: maxFloat(0, outerH-c.insetY)}
	c.iconDest = core.Rect{}
	c.hasIcon = icon > 0 && outerW > 0 && outerH > 0
	if c.hasIcon {
		c.iconDest = core.Rect{X: c.content.X, Y: c.content.Y, W: minFloat(icon, c.content.W), H: minFloat(icon, c.content.H)}
	}
}

// storeRows caches content-relative header lines into reused storage.
// Truncated tails are zeroed so old strings never leak across tooltips.
func (c *RichTooltipCache) storeRows(header richTooltipHeader, textX float32) {
	clearTooltipRows(c.titleRows)
	clearTooltipRows(c.subtitleRows)
	c.titleRows = c.titleRows[:0]
	y := float32(0)
	for _, line := range header.title {
		c.titleRows = append(c.titleRows, richTooltipRow{Text: line, X: textX, Y: y})
		y += RichTooltipTitleHeight
	}
	c.subtitleRows = c.subtitleRows[:0]
	y = header.titleH
	for i, line := range header.subtitle {
		yy := y
		if i == 0 && len(header.title) > 0 {
			yy += RichTooltipSectionGap
		}
		c.subtitleRows = append(c.subtitleRows, richTooltipRow{Text: line, X: textX, Y: yy})
		y = yy + RichTooltipSubtitleHeight
	}
}

// storeBody caches content-relative body fragments into reused storage.
// The icon column and header block offset fragments from the content
// origin; tints resolve at draw time from the retained segments.
func (c *RichTooltipCache) storeBody(header richTooltipHeader, body richTooltipBodyPlan, textX float32) {
	clearTooltipSpans(c.bodySpans)
	c.bodySpans = c.bodySpans[:0]
	bodyY := header.height
	if body.gapped {
		bodyY += RichTooltipSectionGap
	}
	for _, span := range body.spans {
		span.Bounds.X += textX
		span.Bounds.Y += bodyY
		c.bodySpans = append(c.bodySpans, span)
	}
}

// retain copies the payload keys used for change detection.
// Strings alias safely; reused backing arrays are cleared first so
// truncated tails never retain obsolete text or link targets.
func (c *RichTooltipCache) retain(data core.RichTooltip) {
	c.data.Title, c.data.Subtitle = data.Title, data.Subtitle
	c.data.TitleColor, c.data.HasTitleColor = data.TitleColor, data.HasTitleColor
	c.data.Class = data.Class
	c.data.Width, c.data.Icon, c.data.HasIcon = data.Width, data.Icon, data.HasIcon
	clearTooltipSegments(c.data.Segments)
	if cap(c.data.Segments) < len(data.Segments) {
		c.data.Segments = make([]core.RichSegment, len(data.Segments))
	} else {
		c.data.Segments = c.data.Segments[:len(data.Segments)]
	}
	copy(c.data.Segments, data.Segments)
}

// clearTooltipRows zeroes cached header lines before storage reuse.
func clearTooltipRows(rows []richTooltipRow) {
	for i := range rows {
		rows[i] = richTooltipRow{}
	}
}

// clearTooltipSpans zeroes cached body fragments before storage reuse.
func clearTooltipSpans(spans []RichSpanLayout) {
	for i := range spans {
		spans[i] = RichSpanLayout{}
	}
}

// clearTooltipSegments zeroes retained body segments before storage reuse.
func clearTooltipSegments(segments []core.RichSegment) {
	for i := range segments {
		segments[i] = core.RichSegment{}
	}
}

// richTooltipHasDrawable reports whether the popup carries visible rows.
func richTooltipHasDrawable(outerW, outerH float32, header richTooltipHeader, body richTooltipBodyPlan, hasIcon bool) bool {
	if outerW <= 0 || outerH <= 0 {
		return false
	}
	if hasIcon {
		return true
	}
	return len(header.title) > 0 || len(header.subtitle) > 0 || len(body.spans) > 0
}

// richTooltipPlace returns outer bounds near the anchor, flipped above or
// to the left when the popup would leave the viewport, then clamped.
func richTooltipPlace(outerW, outerH float32, anchor, viewport core.Vec2) core.Rect {
	bounds := core.Rect{X: anchor.X + TooltipCursorOffset, Y: anchor.Y + TooltipCursorDrop, W: outerW, H: outerH}
	if viewport.X > 0 && bounds.X+bounds.W > viewport.X {
		bounds.X = anchor.X - bounds.W - TooltipCursorOffset
	}
	if viewport.Y > 0 && bounds.Y+bounds.H > viewport.Y {
		bounds.Y = anchor.Y - bounds.H - TooltipCursorDrop
	}
	if bounds.X < 0 {
		bounds.X = 0
	}
	if bounds.Y < 0 {
		bounds.Y = 0
	}
	if viewport.X > 0 {
		bounds.X = max(0, min(bounds.X, viewport.X-bounds.W))
	}
	if viewport.Y > 0 {
		bounds.Y = max(0, min(bounds.Y, viewport.Y-bounds.H))
	}
	return bounds
}

// richTooltipSpanTint resolves one body fragment tint from its segment.
// Explicit colors win, then parent-registered per-kind colors, then link
// blue, then the popup body default.
func (t *Theme) richTooltipSpanTint(segments []core.RichSegment, span RichSpanLayout) color.RGBA {
	if span.HasColor {
		return span.Color.RGBA()
	}
	if span.Segment >= 0 && span.Segment < len(segments) {
		segment := segments[span.Segment]
		if segment.HasColor {
			return segment.Color.RGBA()
		}
		if segment.Link.Kind != core.LinkNone {
			return t.richLinkTint(false, core.Color{}, segment.Link.Kind, richTooltipLinkText)
		}
	}
	if span.Linked() {
		return t.richLinkTint(false, core.Color{}, span.Link.Kind, richTooltipLinkText)
	}
	return richTooltipBodyText
}

// DrawRichTooltip renders the cached rich tooltip shell, icon, title,
// subtitle, and body fragments clipped to the outer popup.
// info.Bounds is ignored; the cache owns geometry. A nil or invalid cache
// draws nothing. Steady draws allocate nothing after cache warmup.
func (t *Theme) DrawRichTooltip(info core.WidgetInfo, cache *RichTooltipCache) {
	if t == nil || cache == nil || !cache.valid {
		return
	}
	bounds := cache.bounds
	if bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	info.Kind = core.WidgetTooltip
	info.Bounds = bounds
	info.Class = cache.data.Class
	t.drawPart(info.Kind, skin.PartBackground, bounds, info.State, info.Class)
	t.DrawWidgetPart(info.Kind, skin.PartBorder, bounds, info.State, info.Class)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	t.PushClip(cache.content)
	t.drawRichTooltipIcon(info, cache)
	t.drawRichTooltipRows(info, cache)
	t.drawRichTooltipBody(info, cache)
	t.PopClip()
}

// drawRichTooltipIcon records and draws the cached icon box.
// A missing texture logs its geometry without drawing pixels.
func (t *Theme) drawRichTooltipIcon(info core.WidgetInfo, cache *RichTooltipCache) {
	if !cache.hasIcon {
		return
	}
	dest := cache.iconDest
	if dest.W <= 0 || dest.H <= 0 {
		return
	}
	icon := cache.data.Icon
	tint := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if icon.HasTint {
		tint = icon.Tint.RGBA()
	}
	descriptor := skin.SkinDescriptor{Tint: core.ToColor(tint), HasTexture: icon.ID != 0}
	descriptor.Texture = skin.Texture{ID: icon.ID, Width: icon.Width, Height: icon.Height}
	descriptor.AtlasRegion = icon.SourceRect()
	t.logDrawCall(info.Kind, skin.PartIcon, info.State, dest, t.snap(dest), descriptor, tint, icon.ID == 0)
	if !rl.IsWindowReady() {
		return
	}
	if icon.ID == 0 {
		if t.debugMode {
			drawFallbackPart(t.snap(dest), tint)
		}
		return
	}
	texture := toRaylibTexture(descriptor.Texture)
	source := atlasRegion(descriptor, texture)
	drawSingleTexture(texture, source, t.snap(dest), tint)
}

// drawRichTooltipRows records and draws cached title and subtitle lines.
// Relative rows resolve against the absolute content origin at draw time.
func (t *Theme) drawRichTooltipRows(info core.WidgetInfo, cache *RichTooltipCache) {
	titleTint := richTooltipTitleText
	if cache.data.HasTitleColor {
		titleTint = cache.data.TitleColor.RGBA()
	}
	for _, row := range cache.titleRows {
		x, y := cache.content.X+row.X, cache.content.Y+row.Y
		rect := core.Rect{X: x, Y: y, W: cache.content.W, H: RichTooltipTitleHeight}
		t.logDrawCall(info.Kind, skin.PartText, info.State, rect, rect, skin.SkinDescriptor{}, titleTint, false)
		t.DrawText(row.Text, x, y, RichTooltipTitleSize, false, titleTint)
	}
	for _, row := range cache.subtitleRows {
		x, y := cache.content.X+row.X, cache.content.Y+row.Y
		rect := core.Rect{X: x, Y: y, W: cache.content.W, H: RichTooltipSubtitleHeight}
		t.logDrawCall(info.Kind, skin.PartText, info.State, rect, rect, skin.SkinDescriptor{}, richTooltipSubtitleText, false)
		t.DrawText(row.Text, x, y, RichTooltipSubtitleSize, false, richTooltipSubtitleText)
	}
}

// drawRichTooltipBody records and draws cached body fragments with link
// underlines. Icon fragments draw the whitelisted graphic. Relative
// fragments resolve against the content origin here, so cursor motion never
// remeasures; tints resolve from retained segments.
func (t *Theme) drawRichTooltipBody(info core.WidgetInfo, cache *RichTooltipCache) {
	for i := range cache.bodySpans {
		span := cache.bodySpans[i]
		tint := t.richTooltipSpanTint(cache.data.Segments, span)
		row := span.Bounds
		row.X += cache.content.X
		row.Y += cache.content.Y
		if span.IsIcon {
			t.drawRichIconBox(info, span.Icon, row)
			continue
		}
		t.logDrawCall(info.Kind, skin.PartText, info.State, row, row, skin.SkinDescriptor{}, tint, false)
		size := span.Size
		if size <= 0 {
			size = RichFontSize
		}
		t.drawRichWord(span.Text, size, span.Font, span.Bold, row.X, row.Y, tint)
		if !span.Linked() {
			continue
		}
		underline := t.snap(core.Rect{X: row.X, Y: row.Y + row.H - 4, W: row.W, H: 1})
		t.logDrawCall(info.Kind, skin.PartOverlay, core.StateNormal, row, underline, skin.SkinDescriptor{}, tint, false)
		if rl.IsWindowReady() {
			rl.DrawRectangleRec(toRaylibRect(underline), tint)
		}
	}
}

// minFloat returns the smaller of two widths for tooltip fitting.
func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
