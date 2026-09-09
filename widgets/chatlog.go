package widgets

import (
	"math"

	"github.com/draxxris/rtgui/core"
)

const (
	// DefaultChatLogMaxMessages bounds history when maxMessages <= 0.
	DefaultChatLogMaxMessages = 200
	// ChatLogMessageGap separates stacked messages in logical pixels.
	ChatLogMessageGap = float32(4)
)

// ChatMessage is one read-only log entry. ID is assigned by ChatLog and stays
// stable until evicted; Segments are deep-copied on add.
type ChatMessage struct {
	// ID is the stable entry identity assigned by the log.
	ID uint64
	// Segments are the word-wrapped display runs for the entry.
	Segments []core.RichSegment
}

// ChatLog is a bounded read-only log of wrapped rich-text messages. The widget
// owns domain data only; hover, press, scroll-thumb, and layout-cache state
// stay in ui.UI on the owning goroutine. Input stays in Textbox; this widget
// never edits, focuses, or selects text.
type ChatLog struct {
	base
	messages    []ChatMessage
	maxMessages int
	scrollY     float32
	maxScrollY  float32
	autoStick   bool
	nextID      uint64
	revision    uint64
}

// NewChatLog returns an enabled log with bounded history and bottom stick on.
// Non-positive maxMessages selects DefaultChatLogMaxMessages, never unbounded.
func NewChatLog(name string, bounds core.Rect, maxMessages int) *ChatLog {
	if maxMessages <= 0 {
		maxMessages = DefaultChatLogMaxMessages
	}
	return &ChatLog{
		base:        newBase(name, core.WidgetChatLog, bounds),
		maxMessages: maxMessages,
		autoStick:   true,
		nextID:      1,
	}
}

// bumpRevision advances the content revision, skipping zero.
func (l *ChatLog) bumpRevision() {
	l.revision++
	if l.revision == 0 {
		l.revision = 1
	}
}

// AddMessage appends a deep copy of segments as one entry, evicting oldest
// first past capacity, and reports the assigned ID. Empty segment input still
// appends one spacer entry so blank lines stay visible. It reports false for
// a nil log. When AutoStick is on and the view was at the bottom, the offset
// stays glued; the UI reconciles the new limit and completes the stick.
func (l *ChatLog) AddMessage(segments []core.RichSegment) (uint64, bool) {
	if l == nil {
		return 0, false
	}
	wasAtBottom := l.scrollY >= l.maxScrollY
	id := l.nextID
	l.nextID++
	if l.nextID == 0 {
		l.nextID = 1
	}
	entry := ChatMessage{ID: id, Segments: append([]core.RichSegment(nil), segments...)}
	l.messages = append(l.messages, entry)
	l.evictOverflow()
	if l.autoStick && wasAtBottom {
		l.scrollY = l.maxScrollY
	} else {
		l.scrollY = clampScroll(l.scrollY, l.maxScrollY)
	}
	l.bumpRevision()
	return id, true
}

// AddText appends one plain-text entry and reports the assigned ID.
func (l *ChatLog) AddText(text string) (uint64, bool) {
	if l == nil {
		return 0, false
	}
	if text == "" {
		return l.AddMessage(nil)
	}
	return l.AddMessage([]core.RichSegment{{Text: text}})
}

// evictOverflow drops oldest entries past capacity, releasing references.
func (l *ChatLog) evictOverflow() {
	if l == nil || len(l.messages) <= l.maxMessages {
		return
	}
	drop := len(l.messages) - l.maxMessages
	for i := 0; i < drop; i++ {
		clear(l.messages[i].Segments)
		l.messages[i].Segments = nil
		l.messages[i].ID = 0
	}
	copy(l.messages, l.messages[drop:])
	tail := l.messages[len(l.messages)-drop:]
	clear(tail)
	l.messages = l.messages[:len(l.messages)-drop]
}

// Clear drops every entry, resets the offset, and reports whether entries existed.
func (l *ChatLog) Clear() bool {
	if l == nil || len(l.messages) == 0 {
		return false
	}
	for i := range l.messages {
		clear(l.messages[i].Segments)
	}
	clear(l.messages)
	l.messages = l.messages[:0]
	l.scrollY = 0
	l.maxScrollY = 0
	l.bumpRevision()
	return true
}

// MessageCount returns the retained entry count.
func (l *ChatLog) MessageCount() int {
	if l == nil {
		return 0
	}
	return len(l.messages)
}

// MessageAt returns a defensive copy of the indexed entry.
func (l *ChatLog) MessageAt(index int) (ChatMessage, bool) {
	if l == nil || index < 0 || index >= len(l.messages) {
		return ChatMessage{}, false
	}
	src := l.messages[index]
	return ChatMessage{ID: src.ID, Segments: append([]core.RichSegment(nil), src.Segments...)}, true
}

// Messages returns a defensive deep copy of retained entries.
func (l *ChatLog) Messages() []ChatMessage {
	if l == nil || len(l.messages) == 0 {
		return nil
	}
	out := make([]ChatMessage, len(l.messages))
	for i, src := range l.messages {
		out[i].ID = src.ID
		out[i].Segments = append([]core.RichSegment(nil), src.Segments...)
	}
	return out
}

// IndexOfMessage resolves an entry index by stable ID, or -1 when evicted.
func (l *ChatLog) IndexOfMessage(id uint64) int {
	if l == nil || id == 0 {
		return -1
	}
	for i, msg := range l.messages {
		if msg.ID == id {
			return i
		}
	}
	return -1
}

// MessageByID returns a defensive copy of the entry with the given ID.
func (l *ChatLog) MessageByID(id uint64) (ChatMessage, bool) {
	return l.MessageAt(l.IndexOfMessage(id))
}

// MessageSegmentCount returns the segment count for one entry without copying.
func (l *ChatLog) MessageSegmentCount(msgIndex int) int {
	if l == nil || msgIndex < 0 || msgIndex >= len(l.messages) {
		return 0
	}
	return len(l.messages[msgIndex].Segments)
}

// MessageSegmentAt returns a copy of one entry segment for cache reads.
func (l *ChatLog) MessageSegmentAt(msgIndex, segIndex int) (core.RichSegment, bool) {
	if l == nil || msgIndex < 0 || msgIndex >= len(l.messages) {
		return core.RichSegment{}, false
	}
	segs := l.messages[msgIndex].Segments
	if segIndex < 0 || segIndex >= len(segs) {
		return core.RichSegment{}, false
	}
	return segs[segIndex], true
}

// MessageSegments returns a defensive copy of one entry's segments.
func (l *ChatLog) MessageSegments(msgIndex int) []core.RichSegment {
	if l == nil || msgIndex < 0 || msgIndex >= len(l.messages) {
		return nil
	}
	return append([]core.RichSegment(nil), l.messages[msgIndex].Segments...)
}

// CopyMessageSegmentsInto copies one entry's segments into reused storage.
// It grows dst only when capacity is short and clears any truncated tail so
// shortened copies never retain old strings. The result shares string bytes
// with the log and must be treated as read-only until the next copy.
func (l *ChatLog) CopyMessageSegmentsInto(msgIndex int, dst []core.RichSegment) []core.RichSegment {
	if l == nil || msgIndex < 0 || msgIndex >= len(l.messages) {
		clearRichSegments(dst)
		return dst[:0]
	}
	return copyRichSegmentsInto(dst, l.messages[msgIndex].Segments)
}

// MessageLinkCount returns the linked segment count in one entry.
func (l *ChatLog) MessageLinkCount(msgIndex int) int {
	if l == nil || msgIndex < 0 || msgIndex >= len(l.messages) {
		return 0
	}
	count := 0
	for _, segment := range l.messages[msgIndex].Segments {
		if segment.Link.Kind != core.LinkNone {
			count++
		}
	}
	return count
}

// MessageLinkAt returns the nth link of one entry in segment order.
func (l *ChatLog) MessageLinkAt(msgIndex, linkIndex int) (core.Link, int, bool) {
	if l == nil || msgIndex < 0 || msgIndex >= len(l.messages) || linkIndex < 0 {
		return core.Link{}, -1, false
	}
	for segIndex, segment := range l.messages[msgIndex].Segments {
		if segment.Link.Kind == core.LinkNone {
			continue
		}
		if linkIndex == 0 {
			return segment.Link, segIndex, true
		}
		linkIndex--
	}
	return core.Link{}, -1, false
}

// MaxMessages returns the bounded history capacity.
func (l *ChatLog) MaxMessages() int {
	if l == nil || l.maxMessages <= 0 {
		return DefaultChatLogMaxMessages
	}
	return l.maxMessages
}

// SetMaxMessages replaces capacity, evicting oldest first, and reports the log
// for chaining. Non-positive values select the default, never unbounded.
func (l *ChatLog) SetMaxMessages(n int) *ChatLog {
	if l == nil {
		return l
	}
	if n <= 0 {
		n = DefaultChatLogMaxMessages
	}
	if l.maxMessages == n {
		return l
	}
	l.maxMessages = n
	l.evictOverflow()
	l.scrollY = clampScroll(l.scrollY, l.maxScrollY)
	l.bumpRevision()
	return l
}

// Revision returns the content revision bumped by appends, clears, and evictions.
func (l *ChatLog) Revision() uint64 {
	if l == nil {
		return 0
	}
	return l.revision
}

// Text returns concatenated plain entry text with newline separators for
// search and fallback. Icons contribute no characters.
func (l *ChatLog) Text() string {
	if l == nil || len(l.messages) == 0 {
		if l == nil {
			return ""
		}
		return l.base.Text()
	}
	text := ""
	for i, msg := range l.messages {
		if i > 0 {
			text += "\n"
		}
		for _, segment := range msg.Segments {
			text += segment.Text
		}
	}
	return text
}

// ScrollOffset returns the vertical scroll offset in logical pixels.
func (l *ChatLog) ScrollOffset() float32 {
	if l == nil {
		return 0
	}
	return l.scrollY
}

// MaxScroll returns the maximum vertical scroll offset.
func (l *ChatLog) MaxScroll() float32 {
	if l == nil {
		return 0
	}
	return l.maxScrollY
}

// SetScrollOffset replaces the scroll offset clamped to [0, MaxScroll].
// It reports whether the offset changed.
func (l *ChatLog) SetScrollOffset(offset float32) bool {
	if l == nil {
		return false
	}
	offset = clampScroll(offset, l.maxScrollY)
	if l.scrollY == offset {
		return false
	}
	l.scrollY = offset
	return true
}

// ScrollBy advances the scroll offset and reports whether it changed.
func (l *ChatLog) ScrollBy(dy float32) bool {
	if l == nil || dy == 0 || math.IsNaN(float64(dy)) {
		return false
	}
	return l.SetScrollOffset(l.scrollY + dy)
}

// ScrollToBottom jumps to the maximum offset and reports whether it moved.
func (l *ChatLog) ScrollToBottom() bool {
	if l == nil {
		return false
	}
	return l.SetScrollOffset(l.maxScrollY)
}

// IsAtBottom reports whether the view is glued to the newest entry.
func (l *ChatLog) IsAtBottom() bool {
	return l != nil && l.scrollY >= l.maxScrollY
}

// AutoStick reports whether appends keep a bottom-anchored view glued.
func (l *ChatLog) AutoStick() bool {
	return l != nil && l.autoStick
}

// SetAutoStick enables bottom stick and reports the log for chaining.
func (l *ChatLog) SetAutoStick(stick bool) *ChatLog {
	if l != nil {
		l.autoStick = stick
	}
	return l
}

// SetMaxScroll configures an explicit scroll limit for UI reconciliation.
// It clamps the current offset and reports the log for chaining.
func (l *ChatLog) SetMaxScroll(max float32) *ChatLog {
	if l == nil {
		return l
	}
	if math.IsNaN(float64(max)) || max < 0 {
		max = 0
	}
	l.maxScrollY = max
	l.SetScrollOffset(l.scrollY)
	return l
}

// EnsureScrollBounds reconciles the derived scroll limit against total content
// height and viewport height, then clamps the offset. The UI calls this with
// theme-measured heights before hit testing, scrolling, and drawing.
func (l *ChatLog) EnsureScrollBounds(totalH, viewportH float32) bool {
	if l == nil {
		return false
	}
	if math.IsNaN(float64(totalH)) || totalH < 0 {
		totalH = 0
	}
	if math.IsNaN(float64(viewportH)) || viewportH < 0 {
		viewportH = 0
	}
	max := totalH - viewportH
	if max < 0 {
		max = 0
	}
	changed := false
	if max != l.maxScrollY {
		l.maxScrollY = max
		changed = true
	}
	if l.SetScrollOffset(l.scrollY) {
		changed = true
	}
	return changed
}

// OnLinkClick attaches a link click callback directly to the log.
func (l *ChatLog) OnLinkClick(fn func(core.Link)) *ChatLog {
	if l != nil {
		l.callbacks.LinkClick = fn
	}
	return l
}

// OnLinkClickHandler returns the link click callback.
func (l *ChatLog) OnLinkClickHandler() func(core.Link) {
	if l == nil {
		return nil
	}
	return l.callbacks.LinkClick
}

// OnLinkTooltipRequested attaches a hover-text provider for links.
func (l *ChatLog) OnLinkTooltipRequested(fn func(core.Link) string) *ChatLog {
	if l != nil {
		l.callbacks.LinkTooltip = fn
	}
	return l
}

// OnLinkTooltipHandler returns the link tooltip provider.
func (l *ChatLog) OnLinkTooltipHandler() func(core.Link) string {
	if l == nil {
		return nil
	}
	return l.callbacks.LinkTooltip
}

// SetTooltip attaches a hover tooltip string directly to the log.
func (l *ChatLog) SetTooltip(text string) *ChatLog {
	l.base.SetTooltip(text)
	return l
}

// SetTextColor configures an explicit text color for log entries.
func (l *ChatLog) SetTextColor(color core.Color) *ChatLog {
	l.base.SetTextColor(color)
	return l
}

// SetFontSize sets an explicit font size in pixels for log entries.
func (l *ChatLog) SetFontSize(size float32) *ChatLog {
	l.base.SetFontSize(size)
	return l
}

// SetItalic configures whether log entries use the italic theme font.
func (l *ChatLog) SetItalic(italic bool) *ChatLog {
	l.base.SetItalic(italic)
	return l
}

// SetAlign configures the horizontal text alignment within entries.
func (l *ChatLog) SetAlign(align core.TextAlign) *ChatLog {
	l.base.SetAlign(align)
	return l
}
