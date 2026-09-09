package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// ChatLogContent returns the skin-aware log viewport used for message layout,
// drawing, hit testing, and scroll bounds. Callers must use this — never raw
// log bounds — so wrapped, drawn, and hit rows always agree. It is safe on a
// nil theme, where it returns bounds.
func (t *Theme) ChatLogContent(bounds core.Rect, state core.WidgetState, class ...string) core.Rect {
	var background, border skin.SkinDescriptor
	if t != nil {
		background, _ = t.resolveDescriptor(core.WidgetChatLog, skin.PartBackground, state, class...)
		border, _ = t.resolveDescriptor(core.WidgetChatLog, skin.PartBorder, state, class...)
	}
	return t.snap(ContentRect(bounds, background, border))
}

// Rows layout uses ListRowsContent — the same track-excluded viewport lists
// use — so wrapped text never slides under the scrollbar track.
// ChatMessageHeight returns the content height for one entry at the given
// content width. Empty entries still occupy one row so blank lines stay
// visible. Gaps between entries are added by ChatTotalHeight, not here.
func (t *Theme) ChatMessageHeight(width float32, segments []core.RichSegment) float32 {
	if width <= 0 {
		return RichLineHeight
	}
	const huge = float32(1000000)
	content := core.Rect{W: width, H: huge}
	cursor := richCursor(content, themeRichSpace(t))
	spans := layoutRichSliceInto(t, cursor, segments, nil)
	if len(spans) == 0 {
		return RichLineHeight
	}
	extent := float32(0)
	for _, span := range spans {
		if bottom := span.Bounds.Y + span.Bounds.H; bottom > extent {
			extent = bottom
		}
	}
	if extent < RichLineHeight {
		extent = RichLineHeight
	}
	return extent
}

// ChatTotalHeight sums message heights plus inter-message gaps. With one or
// fewer messages no gap applies; negative gaps clamp to zero.
func ChatTotalHeight(heights []float32, gap float32) float32 {
	if len(heights) == 0 {
		return 0
	}
	if gap < 0 {
		gap = 0
	}
	total := float32(0)
	for _, h := range heights {
		if h > 0 {
			total += h
		}
	}
	total += gap * float32(len(heights)-1)
	return total
}

// ChatWindow returns the first and last message indices intersecting the
// viewport at the given scroll offset. An empty range reports last < first.
func ChatWindow(heights []float32, gap, scrollY, viewportH float32) (int, int) {
	if len(heights) == 0 || viewportH <= 0 {
		return 0, -1
	}
	if gap < 0 {
		gap = 0
	}
	if scrollY < 0 {
		scrollY = 0
	}
	first := -1
	last := -1
	top := float32(0)
	for i, h := range heights {
		bottom := top + h
		if bottom > scrollY && top < scrollY+viewportH {
			if first < 0 {
				first = i
			}
			last = i
		}
		top = bottom + gap
	}
	if first < 0 {
		return 0, -1
	}
	return first, last
}

// ChatMessageRect returns one message rect inside content, translated by the
// scroll offset. Heights must match the ChatMessageHeight values used for
// ChatTotalHeight and ChatWindow so draw and hit-test geometry agree.
func ChatMessageRect(content core.Rect, heights []float32, gap, scrollY float32, index int) (core.Rect, bool) {
	if index < 0 || index >= len(heights) || content.W <= 0 {
		return core.Rect{}, false
	}
	if gap < 0 {
		gap = 0
	}
	top := float32(0)
	for i := 0; i < index; i++ {
		top += heights[i] + gap
	}
	return core.Rect{X: content.X, Y: content.Y + top - scrollY, W: content.W, H: heights[index]}, true
}

// DrawChatLog renders the log shell, the visible message window, and the
// scrollbar. Heights must be the per-message ChatMessageHeight values for the
// passed rows width; hoveredMsg is a message index (or -1) with hoveredSeg a
// segment index within it. segScratch reuses segment storage across messages.
func (t *Theme) DrawChatLog(info core.WidgetInfo, log *widgets.ChatLog, rows core.Rect, heights []float32, gap, scrollY float32, segScratch []core.RichSegment, hoveredMsg, hoveredSeg int, thumbState core.WidgetState) {
	if t == nil || log == nil {
		return
	}
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State, info.Class)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	count := log.MessageCount()
	if rows.W <= 0 || rows.H <= 0 || count <= 0 {
		t.drawListScrollbar(info, rows, scrollY, log.MaxScroll(), thumbState)
		return
	}
	first, last := ChatWindow(heights, gap, scrollY, rows.H)
	t.PushClip(rows)
	for index := first; index <= last; index++ {
		rect, ok := ChatMessageRect(rows, heights, gap, scrollY, index)
		if !ok {
			continue
		}
		segScratch = log.CopyMessageSegmentsInto(index, segScratch[:0])
		if len(segScratch) == 0 {
			continue
		}
		hovered := -1
		if index == hoveredMsg {
			hovered = hoveredSeg
		}
		t.drawChatMessage(info, rect, segScratch, hovered)
	}
	t.PopClip()
	t.drawListScrollbar(info, rows, scrollY, log.MaxScroll(), thumbState)
}

// drawChatMessage lays out one entry inside its row rect and draws fragments.
// Layout uses the row rect directly with no extra insets so measurement via
// ChatMessageHeight and drawing always agree.
func (t *Theme) drawChatMessage(info core.WidgetInfo, row core.Rect, segments []core.RichSegment, hoveredSeg int) {
	if row.W <= 0 || row.H <= 0 || len(segments) == 0 {
		return
	}
	cursor := richCursor(row, themeRichSpace(t))
	spans := layoutRichSliceInto(t, cursor, segments, nil)
	for _, span := range spans {
		if span.Linked() && span.Segment == hoveredSeg {
			t.drawRichHighlight(info, span.Bounds)
		}
		t.drawRichCachedFragment(info, span)
	}
}
