package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/widgets"
)

// ActivateLink fires the OnLinkClick callback for the nth link of a rich-text
// widget in segment order, without manufacturing pointer or hover state.
// It reports false for unknown, disabled, non-rich-text, or linkless targets.
func (u *UI) ActivateLink(name string, index int) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	target := u.Lookup(name)
	if target == nil {
		u.diagnose("ui.ActivateLink: widget %q not found", name)
		return false
	}
	rt, ok := target.(*widgets.RichText)
	if !ok {
		u.diagnose("ui.ActivateLink: widget %q is %v, expected WidgetRichText", name, target.Kind())
		return false
	}
	if !u.available(rt) {
		u.diagnose("ui.ActivateLink: widget %q is disabled", name)
		return false
	}
	link, ok := rt.LinkAt(index)
	if !ok {
		u.diagnose("ui.ActivateLink: link index %d out of range in %q", index, name)
		return false
	}
	u.fireOnLinkClick(name, link)
	return true
}

// HoveredLink reports the linked segment under the pointer while one is
// hovered. Applications poll this to drive an app-side pointing-hand
// cursor; rtgui never touches the OS cursor itself so headless operation
// stays intact. The index is a segment index matching DrawRichText hover.
func (u *UI) HoveredLink() (string, core.Link, int, bool) {
	if u == nil || u.tipWidget == nil || u.tipWidget != u.hovered || u.tipSeg < 0 {
		return "", core.Link{}, -1, false
	}
	rt, ok := u.tipWidget.(*widgets.RichText)
	if !ok {
		return "", core.Link{}, -1, false
	}
	segment, valid := rt.RichSegmentAt(u.tipSeg)
	if !valid || segment.Link.Kind == core.LinkNone {
		return "", core.Link{}, -1, false
	}
	return rt.Name(), segment.Link, u.tipSeg, true
}

// richSpans lays out message segments in resolved bounds for input and draw.
// It returns nil without a theme so headless callers degrade gracefully.
func (u *UI) richSpans(message *widgets.RichText) []render.RichSpanLayout {
	if message == nil || u.theme == nil {
		return nil
	}
	return u.richCache(message).Spans()
}

// richCache shares one revisioned layout between drawing and pointer queries.
func (u *UI) richCache(message *widgets.RichText) *render.RichLayoutCache {
	if u.richCaches == nil {
		u.richCaches = make(map[string]*render.RichLayoutCache)
	}
	cache := u.richCaches[message.Name()]
	if cache == nil {
		cache = &render.RichLayoutCache{}
		u.richCaches[message.Name()] = cache
	}
	cache.Update(u.theme, message.Bounds(), message, u.visualState(message))
	return cache
}

// richLinkSegAt resolves the linked segment under pos, or -1 for plain text.
func (u *UI) richLinkSegAt(message *widgets.RichText, pos core.Vec2) int {
	fragment, ok := render.RichSpanAt(u.richSpans(message), pos)
	if !ok || !fragment.Linked() {
		return -1
	}
	return fragment.Segment
}

// releaseRichText commits an armed link on a matching release. Pressing a
// link and releasing elsewhere cancels without a callback; pressing plain
// text keeps the shared click behavior. Every release is consumed.
func (u *UI) releaseRichText(message *widgets.RichText, pos core.Vec2) bool {
	armed := u.linkArmedSeg
	u.linkArmedSeg = -1
	if u.hitSurface(pos) != message {
		return true
	}
	fragment, ok := render.RichSpanAt(u.richSpans(message), pos)
	if !ok {
		return true
	}
	if fragment.Linked() && armed == fragment.Segment {
		u.fireOnLinkClick(message.Name(), fragment.Link)
	} else if armed < 0 && message.HitTest(pos) {
		u.activateWidget(message)
	}
	return true
}

// refreshLinkTip updates hover-derived link tooltip state after updateHover
// on the normal input path. Provider text is requested only on hover change;
// repeat frames over the same segment reuse the cached text.
func (u *UI) refreshLinkTip() {
	if u.hovered == nil || u.hovered.Kind() != core.WidgetRichText || !u.hovered.Enabled() {
		u.clearLinkTip()
		return
	}
	rt, ok := u.hovered.(*widgets.RichText)
	if !ok {
		u.clearLinkTip()
		return
	}
	segment := u.richLinkSegAt(rt, u.pointer)
	if segment < 0 {
		u.clearLinkTip()
		return
	}
	if u.tipWidget == u.hovered && u.tipSeg == segment && u.tipRevision == rt.RichRevision() {
		return
	}
	value, _ := rt.RichSegmentAt(segment)
	link := value.Link
	u.tipRevision = rt.RichRevision()
	u.tipWidget, u.tipSeg = u.hovered, segment
	if link.Tooltip != "" {
		u.tipText = link.Tooltip
		return
	}
	u.tipText = ""
	fn := rt.Callbacks().LinkTooltip
	if fn != nil {
		u.tipText = fn(link)
	}
}

// clearLinkTip forgets hover-derived link tooltip state without reporting.
func (u *UI) clearLinkTip() {
	if u == nil {
		return
	}
	u.tipWidget = nil
	u.tipSeg = -1
	u.tipText = ""
}
