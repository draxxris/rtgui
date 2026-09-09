package ui

import (
	"fmt"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// scrollState reports the vertical scroll offset and limit shared by every
// scrollbar owner. Scroll panels keep an explicit app-set limit while lists
// derive theirs; gestures only need offset and limit, so one path serves both.
func scrollState(w widgets.Widget) (offset, max float32, ok bool) {
	switch v := w.(type) {
	case *widgets.ScrollPanel:
		if v == nil {
			return 0, 0, false
		}
		return v.Scroll().Y, v.MaxScroll().Y, true
	case *widgets.List:
		if v == nil {
			return 0, 0, false
		}
		return v.ScrollOffset(), v.MaxScroll(), true
	default:
		return 0, 0, false
	}
}

// setScrollState replaces the vertical offset, preserving a panel's
// horizontal scroll. It reports whether the offset changed.
func setScrollState(w widgets.Widget, offset float32) bool {
	switch v := w.(type) {
	case *widgets.ScrollPanel:
		if v == nil {
			return false
		}
		return v.SetScroll(core.Vec2{X: v.Scroll().X, Y: offset})
	case *widgets.List:
		if v == nil {
			return false
		}
		return v.SetScrollOffset(offset)
	default:
		return false
	}
}

// scrollOwnerContent returns the skin-aware viewport for a scrollbar owner.
// Lists reconcile their derived limit against the true content height first.
func (u *UI) scrollOwnerContent(w widgets.Widget) (core.Rect, bool) {
	switch v := w.(type) {
	case *widgets.ScrollPanel:
		if v == nil {
			return core.Rect{}, false
		}
		return u.scrollContentRect(v), true
	case *widgets.List:
		if v == nil {
			return core.Rect{}, false
		}
		return u.reconcileListBounds(v), true
	default:
		return core.Rect{}, false
	}
}

// innermostScrollOwner walks hit ancestry for the deepest scrollbar owner.
// Clipping keeps outer tracks clear of inner content, so the first match
// owns the gesture.
func (u *UI) innermostScrollOwner(pos core.Vec2) widgets.Widget {
	for w := u.hitSurface(pos); w != nil; w = u.parentWidget(w) {
		if !u.available(w) {
			continue
		}
		if _, _, ok := scrollState(w); ok {
			return w
		}
	}
	return nil
}

// scrollContentRect returns the skin-aware content area inside the scroll panel's border and padding.
func (u *UI) scrollContentRect(sp *widgets.ScrollPanel) core.Rect {
	if sp == nil {
		return core.Rect{}
	}
	if u == nil || u.theme == nil {
		return sp.Bounds()
	}
	return u.theme.ScrollContent(sp.Bounds(), u.visualState(sp), sp.Class())
}

// scrollTrackRect computes the vertical scrollbar track rectangle along the right edge.
func (u *UI) scrollTrackRect(sp *widgets.ScrollPanel) core.Rect {
	if sp == nil {
		return core.Rect{}
	}
	track, _ := render.ScrollTrackRect(u.scrollContentRect(sp), sp.MaxScroll().Y)
	return track
}

// scrollTrackRect computes the vertical scrollbar track rectangle for callers without a UI instance.
func scrollTrackRect(sp *widgets.ScrollPanel) core.Rect {
	var u *UI
	return u.scrollTrackRect(sp)
}

// scrollThumbRect computes the vertical scrollbar thumb rectangle proportional to scroll offset.
func (u *UI) scrollThumbRect(sp *widgets.ScrollPanel) core.Rect {
	if sp == nil {
		return core.Rect{}
	}
	track := u.scrollTrackRect(sp)
	offset, max, _ := scrollState(sp)
	return render.ScrollThumbRect(track, offset, max)
}

// scrollThumbRect computes the vertical scrollbar thumb rectangle for callers without a UI instance.
func scrollThumbRect(sp *widgets.ScrollPanel) core.Rect {
	var u *UI
	return u.scrollThumbRect(sp)
}

// updateScrollThumbHover updates which scrollbar thumb is hovered by pos.
func (u *UI) updateScrollThumbHover(pos core.Vec2) {
	if u == nil {
		return
	}
	u.scrollThumbHovered = nil
	owner := u.innermostScrollOwner(pos)
	if owner == nil {
		return
	}
	content, ok := u.scrollOwnerContent(owner)
	if !ok {
		return
	}
	offset, max, _ := scrollState(owner)
	if max <= 0 {
		return
	}
	track, ok := render.ScrollTrackRect(content, max)
	if !ok {
		return
	}
	if render.ScrollThumbRect(track, offset, max).Contains(pos) {
		u.scrollThumbHovered = owner
	}
}

// handleScrollbarPress tests if pos hits a scrollbar thumb or track and starts drag or jumps.
func (u *UI) handleScrollbarPress(pos core.Vec2) bool {
	if u == nil {
		return false
	}
	owner := u.innermostScrollOwner(pos)
	if owner == nil {
		return false
	}
	content, ok := u.scrollOwnerContent(owner)
	if !ok {
		return false
	}
	offset, max, _ := scrollState(owner)
	if max <= 0 {
		return false
	}
	track, ok := render.ScrollTrackRect(content, max)
	if !ok {
		return false
	}
	thumb := render.ScrollThumbRect(track, offset, max)
	if thumb.Contains(pos) {
		u.scrollThumbDragging = owner
		u.scrollDragStartY = pos.Y
		u.scrollDragStartScroll = offset
		return true
	}
	if track.Contains(pos) {
		u.jumpScrollThumb(owner, track, thumb, pos.Y)
		return true
	}
	return false
}

// jumpScrollThumb moves scroll to center the thumb on a track press.
func (u *UI) jumpScrollThumb(owner widgets.Widget, track, thumb core.Rect, y float32) {
	travel := track.H - thumb.H
	if travel <= 0 {
		return
	}
	progress := (y - track.Y - thumb.H/2) / travel
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}
	_, max, _ := scrollState(owner)
	setScrollState(owner, progress*max)
}

// handleScrollbarDrag adjusts scroll offset during an active thumb drag gesture.
func (u *UI) handleScrollbarDrag(pos core.Vec2) bool {
	if u == nil || u.scrollThumbDragging == nil {
		return false
	}
	owner := u.scrollThumbDragging
	if !u.available(owner) {
		u.scrollThumbDragging = nil
		return false
	}
	content, ok := u.scrollOwnerContent(owner)
	if !ok {
		u.scrollThumbDragging = nil
		return false
	}
	offset, max, _ := scrollState(owner)
	if max <= 0 {
		u.scrollThumbDragging = nil
		return false
	}
	track, ok := render.ScrollTrackRect(content, max)
	if !ok {
		u.scrollThumbDragging = nil
		return false
	}
	thumb := render.ScrollThumbRect(track, offset, max)
	if travel := track.H - thumb.H; travel > 0 {
		setScrollState(owner, u.scrollDragStartScroll+(pos.Y-u.scrollDragStartY)/travel*max)
	}
	return true
}

// handleScrollbarRelease ends an active thumb drag gesture.
func (u *UI) handleScrollbarRelease() bool {
	if u == nil || u.scrollThumbDragging == nil {
		return false
	}
	u.scrollThumbDragging = nil
	return true
}

// scrollThumbStateFor reports the visual state of one scrollbar thumb.
func (u *UI) scrollThumbStateFor(w widgets.Widget) core.WidgetState {
	if w == nil {
		return core.StateNormal
	}
	if u.scrollThumbDragging == w {
		return core.StatePressed
	}
	if u.scrollThumbHovered == w {
		return core.StateHovered
	}
	return core.StateNormal
}

// drawScrollbar renders the visual track and thumb for a scroll panel.
func (u *UI) drawScrollbar(widget *widgets.ScrollPanel) {
	if widget == nil || widget.MaxScroll().Y <= 0 {
		return
	}
	trackRect := u.scrollTrackRect(widget)
	if trackRect.H <= 0 || trackRect.W <= 0 {
		return
	}
	u.theme.DrawWidgetPart(core.WidgetScrollPanel, skin.PartTrack, trackRect, core.StateNormal, widget.Class())

	thumbState := u.scrollThumbStateFor(widget)
	thumbRect := u.scrollThumbRect(widget)
	if thumbRect.H > 0 {
		u.theme.DrawWidgetPart(core.WidgetScrollPanel, skin.PartThumb, thumbRect, thumbState, widget.Class())
	}
}

// ScrollThumbState reports the visual state (Normal, Hovered, Pressed) of a
// scroll panel or collapsible list scrollbar thumb.
func (u *UI) ScrollThumbState(name string) (core.WidgetState, error) {
	if u == nil {
		return core.StateNormal, ErrNilWidget
	}
	w := u.Lookup(name)
	if w == nil {
		return core.StateNormal, fmt.Errorf("ui: widget %q not found", name)
	}
	switch w.(type) {
	case *widgets.ScrollPanel, *widgets.List:
		return u.scrollThumbStateFor(w), nil
	default:
		return core.StateNormal, fmt.Errorf("ui: widget %q is not scrollable", name)
	}
}
