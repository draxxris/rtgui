package ui

import (
	"fmt"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// scrollContentRect returns the skin-aware content area inside the scroll panel's border and padding.
func (u *UI) scrollContentRect(sp *widgets.ScrollPanel) core.Rect {
	if sp == nil {
		return core.Rect{}
	}
	if u == nil || u.theme == nil {
		return sp.Bounds()
	}
	return u.theme.ScrollContent(sp.Bounds(), u.visualState(sp))
}

// scrollTrackRect computes the vertical scrollbar track rectangle along the right edge.
func (u *UI) scrollTrackRect(sp *widgets.ScrollPanel) core.Rect {
	if sp == nil {
		return core.Rect{}
	}
	content := u.scrollContentRect(sp)
	trackWidth := float32(16)
	if content.W < trackWidth || content.H <= 0 {
		return core.Rect{}
	}
	return core.Rect{
		X: content.X + content.W - trackWidth,
		Y: content.Y,
		W: trackWidth,
		H: content.H,
	}
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
	maxScroll := sp.MaxScroll()
	if maxScroll.Y <= 0 || track.H <= 0 {
		return core.Rect{}
	}
	totalContent := track.H + maxScroll.Y
	thumbH := track.H * (track.H / totalContent)
	if thumbH < 24 {
		thumbH = 24
	}
	if thumbH > track.H {
		thumbH = track.H
	}
	travel := track.H - thumbH
	progress := sp.Scroll().Y / maxScroll.Y
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}
	return core.Rect{
		X: track.X,
		Y: track.Y + progress*travel,
		W: track.W,
		H: thumbH,
	}
}

// scrollThumbRect computes the vertical scrollbar thumb rectangle for callers without a UI instance.
func scrollThumbRect(sp *widgets.ScrollPanel) core.Rect {
	var u *UI
	return u.scrollThumbRect(sp)
}

// updateScrollThumbHover updates which scroll panel thumb is hovered by pos.
func (u *UI) updateScrollThumbHover(pos core.Vec2) {
	if u == nil {
		return
	}
	target := u.topmostAt(pos, core.WidgetScrollPanel)
	if sp, ok := target.(*widgets.ScrollPanel); ok && sp.Enabled() && sp.MaxScroll().Y > 0 {
		if u.scrollThumbRect(sp).Contains(pos) {
			u.scrollThumbHovered = sp
			return
		}
	}
	u.scrollThumbHovered = nil
}

// handleScrollbarPress tests if pos hits a scrollbar thumb or track and starts drag or jumps.
func (u *UI) handleScrollbarPress(pos core.Vec2) bool {
	if u == nil {
		return false
	}
	target := u.topmostAt(pos, core.WidgetScrollPanel)
	sp, ok := target.(*widgets.ScrollPanel)
	if !ok || !sp.Enabled() || sp.MaxScroll().Y <= 0 {
		return false
	}
	thumb := u.scrollThumbRect(sp)
	if thumb.Contains(pos) {
		u.scrollThumbDragging = sp
		u.scrollDragStartY = pos.Y
		u.scrollDragStartScroll = sp.Scroll().Y
		return true
	}
	track := u.scrollTrackRect(sp)
	if track.Contains(pos) {
		travel := track.H - thumb.H
		if travel > 0 {
			targetY := pos.Y - track.Y - thumb.H/2
			progress := targetY / travel
			if progress < 0 {
				progress = 0
			} else if progress > 1 {
				progress = 1
			}
			sp.SetScroll(core.Vec2{X: sp.Scroll().X, Y: progress * sp.MaxScroll().Y})
		}
		return true
	}
	return false
}

// handleScrollbarDrag adjusts scroll offset during an active thumb drag gesture.
func (u *UI) handleScrollbarDrag(pos core.Vec2) bool {
	if u == nil || u.scrollThumbDragging == nil {
		return false
	}
	sp := u.scrollThumbDragging
	if !sp.Enabled() || sp.MaxScroll().Y <= 0 {
		u.scrollThumbDragging = nil
		return false
	}
	track := u.scrollTrackRect(sp)
	thumb := u.scrollThumbRect(sp)
	travel := track.H - thumb.H
	if travel > 0 {
		deltaY := pos.Y - u.scrollDragStartY
		scrollDelta := (deltaY / travel) * sp.MaxScroll().Y
		newY := u.scrollDragStartScroll + scrollDelta
		if newY < 0 {
			newY = 0
		} else if newY > sp.MaxScroll().Y {
			newY = sp.MaxScroll().Y
		}
		sp.SetScroll(core.Vec2{X: sp.Scroll().X, Y: newY})
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

// drawScrollbar renders the visual track and thumb for a scroll panel.
func (u *UI) drawScrollbar(widget *widgets.ScrollPanel) {
	if widget == nil || widget.MaxScroll().Y <= 0 {
		return
	}
	trackRect := u.scrollTrackRect(widget)
	if trackRect.H <= 0 || trackRect.W <= 0 {
		return
	}
	u.theme.DrawWidgetPart(core.WidgetScrollPanel, skin.PartTrack, trackRect, core.StateNormal)

	thumbState := core.StateNormal
	if u.scrollThumbDragging == widget {
		thumbState = core.StatePressed
	} else if u.scrollThumbHovered == widget {
		thumbState = core.StateHovered
	}
	thumbRect := u.scrollThumbRect(widget)
	if thumbRect.H > 0 {
		u.theme.DrawWidgetPart(core.WidgetScrollPanel, skin.PartThumb, thumbRect, thumbState)
	}
}

// ScrollThumbState reports the visual state (Normal, Hovered, Pressed) of a scroll panel thumb.
func (u *UI) ScrollThumbState(name string) (core.WidgetState, error) {
	if u == nil {
		return core.StateNormal, ErrNilWidget
	}
	w := u.Lookup(name)
	if w == nil {
		return core.StateNormal, fmt.Errorf("ui: widget %q not found", name)
	}
	sp, ok := w.(*widgets.ScrollPanel)
	if !ok {
		return core.StateNormal, fmt.Errorf("ui: widget %q is not a scroll panel", name)
	}
	if u.scrollThumbDragging == sp {
		return core.StatePressed, nil
	}
	if u.scrollThumbHovered == sp {
		return core.StateHovered, nil
	}
	return core.StateNormal, nil
}
