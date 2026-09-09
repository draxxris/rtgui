package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// chatGap is the inter-message spacing used for layout, drawing, and hit
// testing so all three paths agree on one value.
const chatGap = float32(widgets.ChatLogMessageGap)

// lookupChatLog resolves a registered available chat log for semantic paths.
func (u *UI) lookupChatLog(op, name string) (*widgets.ChatLog, bool) {
	target := u.Lookup(name)
	if target == nil {
		u.diagnose("ui.%s: widget %q not found", op, name)
		return nil, false
	}
	log, ok := target.(*widgets.ChatLog)
	if !ok {
		u.diagnose("ui.%s: widget %q is %v, expected WidgetChatLog", op, name, target.Kind())
		return nil, false
	}
	if !u.available(log) {
		u.diagnose("ui.%s: widget %q is disabled", op, name)
		return nil, false
	}
	return log, true
}

// chatContentRect returns the skin-aware viewport inside the log shell.
func (u *UI) chatContentRect(log *widgets.ChatLog) core.Rect {
	if log == nil {
		return core.Rect{}
	}
	if u == nil || u.theme == nil {
		return log.Bounds()
	}
	return u.theme.ChatLogContent(log.Bounds(), u.visualState(log), log.Class())
}

// chatHeights measures one content height per retained message at width,
// reusing UI scratch storage. Empty logs return nil without touching scratch.
func (u *UI) chatHeights(log *widgets.ChatLog, width float32) []float32 {
	if log == nil || log.MessageCount() == 0 {
		return nil
	}
	count := log.MessageCount()
	if cap(u.chatHeightScratch) < count {
		u.chatHeightScratch = make([]float32, count)
	} else {
		u.chatHeightScratch = u.chatHeightScratch[:count]
	}
	heights := u.chatHeightScratch
	for i := 0; i < count; i++ {
		u.chatSegScratch = log.CopyMessageSegmentsInto(i, u.chatSegScratch[:0])
		heights[i] = u.theme.ChatMessageHeight(width, u.chatSegScratch)
	}
	clear(u.chatSegScratch)
	u.chatSegScratch = u.chatSegScratch[:0]
	return heights
}

// reconcileChatBounds syncs the derived scroll limit with theme-measured
// heights before hit testing, scrolling, and drawing agree on one viewport.
// Track reservation narrows the wrap width, so measurement runs in two stable
// passes: full content first, then the track-excluded rows when overflowing.
// A bottom-anchored view stays glued across appends via AutoStick.
func (u *UI) reconcileChatBounds(log *widgets.ChatLog) (core.Rect, core.Rect, []float32, float32) {
	if log == nil {
		return core.Rect{}, core.Rect{}, nil, 0
	}
	wasAtBottom := log.IsAtBottom()
	content := u.chatContentRect(log)
	heights := u.chatHeights(log, content.W)
	total := render.ChatTotalHeight(heights, chatGap)
	rows := render.ListRowsContent(content, total-content.H)
	if rows.W != content.W {
		heights = u.chatHeights(log, rows.W)
		total = render.ChatTotalHeight(heights, chatGap)
		rows = render.ListRowsContent(content, total-rows.H)
	}
	log.EnsureScrollBounds(total, rows.H)
	if log.AutoStick() && wasAtBottom {
		log.ScrollToBottom()
	}
	return content, rows, heights, total
}

// chatVisibleWindow reconciles bounds and returns the rows viewport, message
// heights, and visible index range. It reports false when nothing can hit.
func (u *UI) chatVisibleWindow(log *widgets.ChatLog) (core.Rect, []float32, int, int, bool) {
	if log == nil || u.theme == nil {
		return core.Rect{}, nil, 0, -1, false
	}
	_, rows, heights, _ := u.reconcileChatBounds(log)
	if rows.W <= 0 || rows.H <= 0 || len(heights) == 0 {
		return core.Rect{}, nil, 0, -1, false
	}
	first, last := render.ChatWindow(heights, chatGap, log.ScrollOffset(), rows.H)
	return rows, heights, first, last, true
}

// chatMessageAt resolves the message and segment indices under pos, or -1.
// Layout uses the reconciled rows rect so hover, press, and draw agree.
func (u *UI) chatMessageAt(log *widgets.ChatLog, pos core.Vec2) (int, int) {
	rows, heights, first, last, ok := u.chatVisibleWindow(log)
	if !ok || !rows.Contains(pos) {
		return -1, -1
	}
	for index := first; index <= last; index++ {
		rect, ok := render.ChatMessageRect(rows, heights, chatGap, log.ScrollOffset(), index)
		if !ok || !rect.Contains(pos) {
			continue
		}
		return index, u.chatSegmentInRow(log, rect, index, pos)
	}
	return -1, -1
}

// chatSegmentInRow lays out one message row and resolves the segment under
// pos, or -1 for gaps between word runs. The scratch copy is released before
// returning so hover, press, and draw never retain it.
func (u *UI) chatSegmentInRow(log *widgets.ChatLog, rect core.Rect, index int, pos core.Vec2) int {
	u.chatSegScratch = log.CopyMessageSegmentsInto(index, u.chatSegScratch[:0])
	segs := u.chatSegScratch
	spans := u.theme.LayoutRichSpans(rect, segs, u.visualState(log))
	clear(u.chatSegScratch)
	u.chatSegScratch = u.chatSegScratch[:0]
	fragment, ok := render.RichSpanAt(spans, pos)
	if !ok {
		return -1
	}
	return fragment.Segment
}

// chatLinkAt resolves the linked segment under pos with its message identity.
func (u *UI) chatLinkAt(log *widgets.ChatLog, pos core.Vec2) (int, uint64, int, core.Link, bool) {
	msgIndex, segIndex := u.chatMessageAt(log, pos)
	if msgIndex < 0 || segIndex < 0 {
		return -1, 0, -1, core.Link{}, false
	}
	segment, ok := log.MessageSegmentAt(msgIndex, segIndex)
	if !ok || segment.Link.Kind == core.LinkNone {
		return -1, 0, -1, core.Link{}, false
	}
	msg, ok := log.MessageAt(msgIndex)
	if !ok {
		return -1, 0, -1, core.Link{}, false
	}
	return msgIndex, msg.ID, segIndex, segment.Link, true
}

// drawChatLog renders one log with pointer-aware link highlighting.
func (u *UI) drawChatLog(log *widgets.ChatLog, state core.WidgetState) {
	info := log.Snapshot(state)
	_, rows, heights, _ := u.reconcileChatBounds(log)
	hoveredMsg, hoveredSeg := u.chatHoverIndex(log, heights)
	u.theme.DrawChatLog(info, log, rows, heights, chatGap, log.ScrollOffset(), u.chatSegScratch[:0], hoveredMsg, hoveredSeg, u.scrollThumbStateFor(log))
	if needsBorder(log.Kind()) {
		u.theme.DrawWidgetPart(log.Kind(), skin.PartBorder, log.Bounds(), state, log.Class())
	}
}

// chatHoverIndex resolves the highlighted message index and segment for
// drawing, or -1. The lookup is gated on hover-derived tip state so
// highlight, press arm, and tooltip always agree on the same cell. Message
// IDs resolve to indices at draw time so eviction never mis-highlights.
func (u *UI) chatHoverIndex(log *widgets.ChatLog, heights []float32) (int, int) {
	if log == nil || u.tipWidget != log || u.tipChatMsg == 0 || u.tipSeg < 0 {
		return -1, -1
	}
	if u.hovered != log {
		return -1, -1
	}
	index := log.IndexOfMessage(u.tipChatMsg)
	if index < 0 || index >= len(heights) {
		return -1, -1
	}
	return index, u.tipSeg
}

// disarmLink forgets an armed link press for RichText and ChatLog alike.
// Chat message IDs are never zero, so zero disarms the chat arm.
func (u *UI) disarmLink() {
	if u == nil {
		return
	}
	u.linkArmedSeg = -1
	u.linkArmedChatMsg = 0
}

// pressChatLog arms the linked segment under pos for a press gesture.
func (u *UI) pressChatLog(log *widgets.ChatLog, pos core.Vec2) {
	_, msgID, segIndex, _, ok := u.chatLinkAt(log, pos)
	if !ok {
		u.disarmLink()
		return
	}
	u.linkArmedSeg = segIndex
	u.linkArmedChatMsg = msgID
}

// releaseChatLog commits an armed link on a matching release. Pressing a link
// and releasing elsewhere cancels without a callback; pressing plain text
// keeps the shared click behavior. Every release is consumed.
func (u *UI) releaseChatLog(log *widgets.ChatLog, pos core.Vec2) bool {
	armed := u.linkArmedSeg
	armedMsg := u.linkArmedChatMsg
	u.disarmLink()
	if u.hitSurface(pos) != log {
		return true
	}
	msgIndex, msgID, segIndex, link, linked := u.chatLinkAt(log, pos)
	_ = msgIndex
	if linked && armedMsg != 0 && armedMsg == msgID && armed == segIndex {
		u.fireOnLinkClick(log.Name(), link)
	} else if armedMsg == 0 && armed < 0 && log.HitTest(pos) {
		u.activateWidget(log)
	}
	return true
}

// refreshChatTip updates hover-derived link tooltip state for one log after
// updateHover on the normal input path. Provider text is requested only on
// hover change; repeat frames over the same cell reuse the cached text.
func (u *UI) refreshChatTip(log *widgets.ChatLog) {
	msgIndex, msgID, segIndex, link, ok := u.chatLinkAt(log, u.pointer)
	_ = msgIndex
	if !ok {
		u.dismissLinkTip()
		return
	}
	if u.tipWidget == log && u.tipChatMsg == msgID && u.tipSeg == segIndex && u.tipRevision == log.Revision() {
		return
	}
	u.tipRevision = log.Revision()
	if u.tipWidget != log || u.tipChatMsg != msgID || u.tipSeg != segIndex {
		u.hoverSince = u.tooltipNow()
	}
	u.tipWidget, u.tipSeg = log, segIndex
	u.tipChatMsg = msgID
	if link.Tooltip != "" {
		u.tipText = link.Tooltip
		return
	}
	u.tipText = ""
	if fn := log.Callbacks().LinkTooltip; fn != nil {
		u.tipText = fn(link)
	}
}

// AppendChatMessage appends segments to a registered available log through
// the shared widget path without manufacturing pointer or hover state.
// It reports the assigned message ID, or false for unknown, disabled, or
// non-chat targets. Eviction is silent; there is no overflow signal in v1.
func (u *UI) AppendChatMessage(name string, segments []core.RichSegment) (uint64, bool) {
	if u == nil {
		return 0, false
	}
	u.reconcileInteraction()
	log, ok := u.lookupChatLog("AppendChatMessage", name)
	if !ok {
		return 0, false
	}
	id, _ := log.AddMessage(segments)
	u.reconcileChatBounds(log)
	return id, true
}

// AppendChatText appends one plain-text entry to a registered available log.
func (u *UI) AppendChatText(name, text string) (uint64, bool) {
	if u == nil {
		return 0, false
	}
	u.reconcileInteraction()
	log, ok := u.lookupChatLog("AppendChatText", name)
	if !ok {
		return 0, false
	}
	id, _ := log.AddText(text)
	u.reconcileChatBounds(log)
	return id, true
}

// ActivateChatLink fires the OnLinkClick callback for the nth link of one
// chat message in segment order, without manufacturing pointer or hover
// state. It reports false for unknown, disabled, non-chat, evicted, or
// linkless targets.
func (u *UI) ActivateChatLink(name string, messageID uint64, linkIndex int) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	log, ok := u.lookupChatLog("ActivateChatLink", name)
	if !ok {
		return false
	}
	msgIndex := log.IndexOfMessage(messageID)
	if msgIndex < 0 {
		u.diagnose("ui.ActivateChatLink: message %d evicted in %q", messageID, name)
		return false
	}
	link, _, ok := log.MessageLinkAt(msgIndex, linkIndex)
	if !ok {
		u.diagnose("ui.ActivateChatLink: link index %d out of range in %q", linkIndex, name)
		return false
	}
	u.fireOnLinkClick(name, link)
	return true
}

// HoveredChatLink reports the linked chat segment under the pointer while one
// is hovered. The message ID identifies the entry; the segment index matches
// the entry's segment order.
func (u *UI) HoveredChatLink() (string, uint64, core.Link, int, bool) {
	if u == nil || u.tipWidget == nil || u.tipWidget != u.hovered || u.tipSeg < 0 || u.tipChatMsg == 0 {
		return "", 0, core.Link{}, -1, false
	}
	log, ok := u.tipWidget.(*widgets.ChatLog)
	if !ok {
		return "", 0, core.Link{}, -1, false
	}
	msgIndex := log.IndexOfMessage(u.tipChatMsg)
	if msgIndex < 0 {
		return "", 0, core.Link{}, -1, false
	}
	segment, valid := log.MessageSegmentAt(msgIndex, u.tipSeg)
	if !valid || segment.Link.Kind == core.LinkNone {
		return "", 0, core.Link{}, -1, false
	}
	return log.Name(), u.tipChatMsg, segment.Link, u.tipSeg, true
}

// ScrollChatToBottom jumps a registered available log to its newest entry.
func (u *UI) ScrollChatToBottom(name string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	log, ok := u.lookupChatLog("ScrollChatToBottom", name)
	if !ok {
		return false
	}
	u.reconcileChatBounds(log)
	return log.ScrollToBottom()
}
