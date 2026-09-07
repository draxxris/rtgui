package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// ScrollPanel provides a scrollable view container with scissored content.
type ScrollPanel struct {
	base
	scroll              core.Vec2
	maxScroll           core.Vec2
	scrollContentDrawer func(bounds core.Rect, scrollOffset core.Vec2)
}

// NewScrollPanel returns an enabled scroll panel with zero scroll offset.
func NewScrollPanel(name string, bounds core.Rect) *ScrollPanel {
	return &ScrollPanel{base: newBase(name, core.WidgetScrollPanel, bounds)}
}

// Scroll returns the current scroll offset.
func (s *ScrollPanel) Scroll() core.Vec2 {
	if s == nil {
		return core.Vec2{}
	}
	return s.scroll
}

// SetScroll replaces the scroll offset and reports whether it changed.
func (s *ScrollPanel) SetScroll(offset core.Vec2) bool {
	if s == nil || s.scroll == offset {
		return false
	}
	s.scroll = offset
	return true
}

// ScrollBy changes the scroll offset clamped within [0, MaxScroll], reporting changes.
func (s *ScrollPanel) ScrollBy(dx, dy float32) bool {
	if s == nil || (dx == 0 && dy == 0) {
		return false
	}
	newX := s.scroll.X + dx
	newY := s.scroll.Y + dy
	if s.maxScroll.X > 0 {
		if newX < 0 {
			newX = 0
		} else if newX > s.maxScroll.X {
			newX = s.maxScroll.X
		}
	}
	if s.maxScroll.Y > 0 {
		if newY < 0 {
			newY = 0
		} else if newY > s.maxScroll.Y {
			newY = s.maxScroll.Y
		}
	}
	return s.SetScroll(core.Vec2{X: newX, Y: newY})
}

// SetMaxScroll configures the maximum allowed scroll offset.
func (s *ScrollPanel) SetMaxScroll(max core.Vec2) *ScrollPanel {
	if s != nil {
		s.maxScroll = max
	}
	return s
}

// MaxScroll returns the maximum allowed scroll offset.
func (s *ScrollPanel) MaxScroll() core.Vec2 {
	if s == nil {
		return core.Vec2{}
	}
	return s.maxScroll
}

// SetScrollContentDrawer registers a custom drawer for the scroll panel's contents.
func (s *ScrollPanel) SetScrollContentDrawer(fn func(bounds core.Rect, scrollOffset core.Vec2)) *ScrollPanel {
	if s != nil {
		s.scrollContentDrawer = fn
	}
	return s
}

// ScrollContentDrawer returns the registered custom content drawer.
func (s *ScrollPanel) ScrollContentDrawer() func(bounds core.Rect, scrollOffset core.Vec2) {
	if s == nil {
		return nil
	}
	return s.scrollContentDrawer
}

// SetTooltip attaches a hover tooltip string directly to the scroll panel.
func (s *ScrollPanel) SetTooltip(text string) *ScrollPanel {
	s.base.SetTooltip(text)
	return s
}
