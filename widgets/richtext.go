package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// RichText is a formatted text widget supporting colored spans and clickable links.
type RichText struct {
	base
	richSegments  []core.RichSegment
	onLinkClick   func(core.Link)
	onLinkTooltip func(core.Link) string
}

// NewRichText returns an enabled rich-text message and copies segments safely.
func NewRichText(name string, bounds core.Rect, segments []core.RichSegment) *RichText {
	rt := &RichText{base: newBase(name, core.WidgetRichText, bounds)}
	rt.SetRichSegments(segments)
	return rt
}

// RichSegments returns a safe snapshot of the formatted segments.
func (r *RichText) RichSegments() []core.RichSegment {
	if r == nil {
		return nil
	}
	return append([]core.RichSegment(nil), r.richSegments...)
}

// SetRichSegments copies segments and reports whether segment content changed.
func (r *RichText) SetRichSegments(segments []core.RichSegment) bool {
	if r == nil {
		return false
	}
	if equalRichSegments(r.richSegments, segments) {
		return false
	}
	r.richSegments = append(r.richSegments[:0], segments...)
	return true
}

// RichPlainText concatenates segment text for search and copy operations.
func (r *RichText) RichPlainText() string {
	if r == nil {
		return ""
	}
	text := ""
	for _, segment := range r.richSegments {
		text += segment.Text
	}
	return text
}

// LinkCount returns the number of linked segments in the message.
func (r *RichText) LinkCount() int {
	if r == nil {
		return 0
	}
	count := 0
	for _, segment := range r.richSegments {
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
	for _, segment := range r.richSegments {
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
		r.onLinkClick = fn
	}
	return r
}

// OnLinkClickHandler returns the link click callback.
func (r *RichText) OnLinkClickHandler() func(core.Link) {
	if r == nil {
		return nil
	}
	return r.onLinkClick
}

// OnLinkTooltipRequested attaches a hover-text provider for links.
func (r *RichText) OnLinkTooltipRequested(fn func(core.Link) string) *RichText {
	if r != nil {
		r.onLinkTooltip = fn
	}
	return r
}

// OnLinkTooltipHandler returns the link tooltip provider.
func (r *RichText) OnLinkTooltipHandler() func(core.Link) string {
	if r == nil {
		return nil
	}
	return r.onLinkTooltip
}

// SetTooltip attaches a fallback hover tooltip string directly to the rich text widget.
func (r *RichText) SetTooltip(text string) *RichText {
	r.base.SetTooltip(text)
	return r
}

// equalRichSegments compares message segment snapshots field by field.
func equalRichSegments(left, right []core.RichSegment) bool {
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
