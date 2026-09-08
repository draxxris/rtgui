package render

import (
	"github.com/draxxris/rtgui/core"
)

// LayoutRichSingleLine flows segments left to right without wrapping for
// single-line controls. Newlines become spaces so buttons, labels, and rows
// never gain rows. Spans carry each segment's size, face, and emphasis; rows
// center the tallest span. The result aliases reuse when provided and is
// clipped by callers to content.
func (t *Theme) LayoutRichSingleLine(content core.Rect, segments []core.RichSegment, align core.TextAlign, reuse []RichSpanLayout) []RichSpanLayout {
	out := reuse[:0]
	if content.W <= 0 || content.H <= 0 || len(segments) == 0 {
		return out
	}
	maxH := t.singleLineMaxH(segments)
	y0 := content.Y + (content.H-maxH)/2
	if y0 < content.Y {
		y0 = content.Y
	}
	total := t.measureRichSingleLine(segments)
	offset := singleLineOffset(align, content.W, total)
	if offset < 0 {
		offset = 0
	}
	x := content.X + offset
	for index, segment := range segments {
		size := richSpanSize(segment)
		font := richSpanFont(segment)
		spanH := richRowHeight(size)
		spanY := y0 + (maxH-spanH)/2
		if segment.HasIcon {
			width := clampRichWordWidth(float32(RichIconSize), x, content.X+content.W, content.W)
			if x >= content.X+content.W || width <= 0 {
				break
			}
			out = append(out, RichSpanLayout{
				Segment:  index,
				Bounds:   core.Rect{X: x, Y: spanY, W: width, H: spanH},
				Link:     segment.Link,
				HasColor: segment.HasColor,
				Color:    segment.Color,
				Icon:     segment.Icon,
				IsIcon:   true,
				Bold:     segment.Bold,
				Size:     size,
				Font:     font,
			})
			x += width + float32(RichIconGap)
			continue
		}
		out = t.appendSingleLineWords(out, index, segment, size, font, content, &x, spanY, spanH)
		if x >= content.X+content.W {
			break
		}
	}
	return out
}

// DrawRichSingleLine layouts and draws single-line rich runs clipped to
// content with the requested alignment. Links render in registered colors
// but earn no underline or highlight outside RichText.
func (t *Theme) DrawRichSingleLine(info core.WidgetInfo, content core.Rect, segments []core.RichSegment, align core.TextAlign) {
	if t == nil || len(segments) == 0 || content.W <= 0 || content.H <= 0 {
		return
	}
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	t.PushClip(content)
	defer t.PopClip()
	oldLen := len(t.singleScratch)
	spans := t.LayoutRichSingleLine(content, segments, align, t.singleScratch[:0])
	for _, span := range spans {
		t.drawRichFragment(info, span, false)
	}
	// Retain capacity for the next draw without retaining stale strings.
	if len(spans) < oldLen {
		clear(t.singleScratch[len(spans):oldLen])
	}
	t.singleScratch = spans
}

// singleLineMaxH returns the tallest row advance across single-line runs.
func (t *Theme) singleLineMaxH(segments []core.RichSegment) float32 {
	maxH := float32(RichLineHeight)
	for _, segment := range segments {
		if height := richRowHeight(richSpanSize(segment)); height > maxH {
			maxH = height
		}
	}
	return maxH
}

// measureRichSingleLine returns the unwrapped advance of segments.
func (t *Theme) measureRichSingleLine(segments []core.RichSegment) float32 {
	total := float32(0)
	for _, segment := range segments {
		if segment.HasIcon {
			total += float32(RichIconSize + RichIconGap)
			continue
		}
		total += t.measureSingleLineText(segment.Text, richSpanFont(segment), richSpanSize(segment))
	}
	return total
}

// measureSingleLineText advances words and spaces with newlines as spaces.
// It scans by slice indexes without allocating, mirroring layoutRichTextInto.
func (t *Theme) measureSingleLineText(text, font string, size float32) float32 {
	width := float32(0)
	space := t.richSpaceFor(themeRichSpace(t), size)
	pos := 0
	for pos < len(text) {
		value := text[pos]
		if value == '\n' || isRichSpaceByte(value) {
			if value == '\t' {
				width += float32(richTabSpaces) * space
			} else {
				width += space
			}
			pos++
			continue
		}
		start := pos
		for pos < len(text) && !isRichSpaceByte(text[pos]) && text[pos] != '\n' {
			pos++
		}
		width += t.measureRichWordStyled(text[start:pos], font, size)
	}
	return width
}

// appendSingleLineWords flows one text run into single-line spans.
// Words alias the segment text by slice indexes without allocating.
func (t *Theme) appendSingleLineWords(out []RichSpanLayout, index int, segment core.RichSegment, size float32, font string, content core.Rect, x *float32, y, spanH float32) []RichSpanLayout {
	right := content.X + content.W
	space := t.richSpaceFor(themeRichSpace(t), size)
	text := segment.Text
	pos := 0
	for pos < len(text) {
		value := text[pos]
		if value == '\n' || isRichSpaceByte(value) {
			if *x >= right {
				return out
			}
			if value == '\t' {
				*x += float32(richTabSpaces) * space
			} else {
				*x += space
			}
			pos++
			continue
		}
		start := pos
		for pos < len(text) && !isRichSpaceByte(text[pos]) && text[pos] != '\n' {
			pos++
		}
		out = t.emitSingleLineWord(out, index, segment, size, font, text[start:pos], right, content.W, x, y, spanH)
		if *x >= right {
			return out
		}
	}
	return out
}

// emitSingleLineWord appends one measured word run, clipping it to the row.
func (t *Theme) emitSingleLineWord(out []RichSpanLayout, index int, segment core.RichSegment, size float32, font, word string, right, contentW float32, x *float32, y, spanH float32) []RichSpanLayout {
	width := t.measureRichWordStyled(word, font, size)
	if *x >= right || width <= 0 {
		return out
	}
	width = clampRichWordWidth(width, *x, right, contentW)
	out = append(out, RichSpanLayout{
		Segment:  index,
		Bounds:   core.Rect{X: *x, Y: y, W: width, H: spanH},
		Text:     word,
		Link:     segment.Link,
		HasColor: segment.HasColor,
		Color:    segment.Color,
		Bold:     segment.Bold,
		Size:     size,
		Font:     font,
	})
	*x += width
	return out
}

// singleLineOffset aligns an unwrapped run inside content width.
func singleLineOffset(align core.TextAlign, contentW, totalW float32) float32 {
	switch align {
	case core.AlignCenter:
		if totalW < contentW {
			return (contentW - totalW) / 2
		}
		return 0
	case core.AlignRight:
		if totalW < contentW {
			return contentW - totalW
		}
		return 0
	default:
		return 0
	}
}
