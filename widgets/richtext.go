package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// RichText is a formatted text widget supporting colored spans, inline
// whitelisted icons, and clickable links. Links stay interactive only here;
// other widgets render the same segments without link activation.
type RichText struct {
	base
}

// NewRichText returns an enabled rich-text message and copies segments safely.
func NewRichText(name string, bounds core.Rect, segments []core.RichSegment) *RichText {
	rt := &RichText{base: newBase(name, core.WidgetRichText, bounds)}
	rt.SetRichSegments(segments)
	return rt
}

// Nil-safe forwards: promoted base methods panic on a nil *RichText when
// evaluating &r.base, so each accessor below guards nil explicitly.

// HasRichText reports whether the message carries display segments.
func (r *RichText) HasRichText() bool {
	if r == nil {
		return false
	}
	return r.base.HasRichText()
}

// RichSegments returns a safe snapshot of the formatted segments.
func (r *RichText) RichSegments() []core.RichSegment {
	if r == nil {
		return nil
	}
	return r.base.RichSegments()
}

// SetRichSegments copies segments and reports whether content changed.
func (r *RichText) SetRichSegments(segments []core.RichSegment) bool {
	if r == nil {
		return false
	}
	return r.base.SetRichSegments(segments)
}

// ClearRichText drops display segments and reports a change.
func (r *RichText) ClearRichText() bool {
	if r == nil {
		return false
	}
	return r.base.ClearRichText()
}

// RichRevision returns the content revision bumped by SetRichSegments.
func (r *RichText) RichRevision() uint64 {
	if r == nil {
		return 0
	}
	return r.base.RichRevision()
}

// RichSegmentCount returns the number of formatted segments without copying.
func (r *RichText) RichSegmentCount() int {
	if r == nil {
		return 0
	}
	return r.base.RichSegmentCount()
}

// RichSegmentAt returns a copy of the indexed segment for cache reads.
func (r *RichText) RichSegmentAt(index int) (core.RichSegment, bool) {
	if r == nil {
		return core.RichSegment{}, false
	}
	return r.base.RichSegmentAt(index)
}

// CopyRichSegmentsInto copies segments into dst reusing its backing store.
// It grows dst only when capacity is short and clears any truncated tail so
// shortened copies never retain old strings. The result shares string bytes
// with the widget and must be treated as read-only by the caller.
func (r *RichText) CopyRichSegmentsInto(dst []core.RichSegment) []core.RichSegment {
	if r == nil {
		clearRichSegments(dst)
		return dst[:0]
	}
	return copyRichSegmentsInto(dst, r.base.richSegments)
}

// RichPlainText concatenates segment text for search and copy operations.
func (r *RichText) RichPlainText() string {
	if r == nil {
		return ""
	}
	return r.base.RichPlainText()
}

// LinkCount returns the number of linked segments in the message.
func (r *RichText) LinkCount() int {
	if r == nil {
		return 0
	}
	count := 0
	for _, segment := range r.base.richSegments {
		if segment.Link.Kind != core.LinkNone {
			count++
		}
	}
	return count
}

// LinkAt returns the nth link in segment order, or false when out of range.
func (r *RichText) LinkAt(index int) (core.Link, bool) {
	if r == nil || index < 0 {
		return core.Link{}, false
	}
	for _, segment := range r.base.richSegments {
		if segment.Link.Kind == core.LinkNone {
			continue
		}
		if index == 0 {
			return segment.Link, true
		}
		index--
	}
	return core.Link{}, false
}

// OnLinkClick attaches a link click callback directly to the rich text widget.
func (r *RichText) OnLinkClick(fn func(core.Link)) *RichText {
	if r != nil {
		r.callbacks.LinkClick = fn
	}
	return r
}

// OnLinkClickHandler returns the link click callback.
func (r *RichText) OnLinkClickHandler() func(core.Link) {
	if r == nil {
		return nil
	}
	return r.callbacks.LinkClick
}

// OnLinkTooltipRequested attaches a hover-text provider for links.
func (r *RichText) OnLinkTooltipRequested(fn func(core.Link) string) *RichText {
	if r != nil {
		r.callbacks.LinkTooltip = fn
	}
	return r
}

// OnLinkTooltipHandler returns the link tooltip provider.
func (r *RichText) OnLinkTooltipHandler() func(core.Link) string {
	if r == nil {
		return nil
	}
	return r.callbacks.LinkTooltip
}

// SetTooltip attaches a fallback hover tooltip string directly to the rich text widget.
func (r *RichText) SetTooltip(text string) *RichText {
	r.base.SetTooltip(text)
	return r
}

// copyRichSegmentsInto copies src into dst reusing backing storage and
// clearing truncated tails. It allocates only when dst capacity is short.
func copyRichSegmentsInto(dst, src []core.RichSegment) []core.RichSegment {
	if len(src) == 0 {
		clearRichSegments(dst)
		return dst[:0]
	}
	oldLen := len(dst)
	if cap(dst) < len(src) {
		out := make([]core.RichSegment, len(src))
		copy(out, src)
		return out
	}
	out := dst[:len(src)]
	copy(out, src)
	if len(src) < oldLen {
		clear(dst[len(src):oldLen])
	}
	return out
}

// clearRichSegments zeroes every entry so a reused slice drops references.
func clearRichSegments(slice []core.RichSegment) {
	clear(slice)
}
