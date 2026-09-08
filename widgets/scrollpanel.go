package widgets

import (
	"github.com/draxxris/rtgui/core"
	"math"
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
	if s == nil {
		return false
	}
	offset = core.Vec2{X: clampScroll(offset.X, s.maxScroll.X), Y: clampScroll(offset.Y, s.maxScroll.Y)}
	if s.scroll == offset {
		return false
	}
	s.scroll = offset
	s.Frame().SetContentOffset(core.Vec2{X: -offset.X, Y: -offset.Y})
	return true
}

// ScrollBy changes the scroll offset clamped within [0, MaxScroll], reporting changes.
func (s *ScrollPanel) ScrollBy(dx, dy float32) bool {
	if s == nil || (dx == 0 && dy == 0) {
		return false
	}
	return s.SetScroll(core.Vec2{X: s.scroll.X + dx, Y: s.scroll.Y + dy})
}

// SetMaxScroll configures the maximum allowed scroll offset.
func (s *ScrollPanel) SetMaxScroll(max core.Vec2) *ScrollPanel {
	if s != nil {
		s.maxScroll = core.Vec2{X: clampScroll(max.X, float32(math.MaxFloat32)), Y: clampScroll(max.Y, float32(math.MaxFloat32))}
		s.SetScroll(s.scroll)
	}
	return s
}

// clampScroll normalizes non-finite values and clamps offsets on every axis.
func clampScroll(value, maximum float32) float32 {
	if math.IsNaN(float64(value)) || value < 0 {
		return 0
	}
	if value > maximum {
		return maximum
	}
	return value
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
