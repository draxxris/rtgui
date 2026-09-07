package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// Canvas is a custom immediate-mode drawing widget.
type Canvas struct {
	base
	canvasDraw func(bounds core.Rect)
}

// NewCanvas returns an enabled custom drawing widget that executes drawFn during Draw.
func NewCanvas(name string, bounds core.Rect, drawFn func(bounds core.Rect)) *Canvas {
	return &Canvas{
		base:       newBase(name, core.WidgetCanvas, bounds),
		canvasDraw: drawFn,
	}
}

// CanvasDraw returns the widget's custom draw function, if any.
func (c *Canvas) CanvasDraw() func(bounds core.Rect) {
	if c == nil {
		return nil
	}
	return c.canvasDraw
}

// SetCanvasDraw replaces the widget's custom draw function.
func (c *Canvas) SetCanvasDraw(fn func(bounds core.Rect)) *Canvas {
	if c != nil {
		c.canvasDraw = fn
	}
	return c
}

// SetTooltip attaches a hover tooltip string directly to the canvas widget.
func (c *Canvas) SetTooltip(text string) *Canvas {
	c.base.SetTooltip(text)
	return c
}
