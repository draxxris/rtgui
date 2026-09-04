// Package core contains the data types shared by the rtgui packages.
//
// The package deliberately has no rendering or window-system dependency.
package core

import "image/color"

// Status is returned by operations that reject an invalid value or cannot
// resolve a requested resource.
type Status int32

const (
	StatusInvalidArg Status = iota + 1
	StatusMissingSkin
)

func (s Status) Error() string {
	switch s {
	case StatusInvalidArg:
		return "invalid argument"
	case StatusMissingSkin:
		return "missing skin"
	default:
		return "rtgui error"
	}
}

// Rect is an axis-aligned rectangle in logical UI coordinates.
type Rect struct {
	X, Y, W, H float32
}

// Contains reports whether p is inside the rectangle, including its edges.
func (r Rect) Contains(p Vec2) bool {
	return p.X >= r.X && p.X <= r.X+r.W && p.Y >= r.Y && p.Y <= r.Y+r.H
}

type Vec2 struct {
	X, Y float32
}

type Color struct {
	R, G, B, A uint8
}

func ToColor(c color.RGBA) Color { return Color{R: c.R, G: c.G, B: c.B, A: c.A} }
func (c Color) RGBA() color.RGBA { return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A} }

type WidgetKind int32

const (
	WidgetButton WidgetKind = iota
	WidgetLabel
	WidgetCheckbox
	WidgetTextbox
	WidgetScrollPanel
	WidgetDropdown
	WidgetSlider
	WidgetProgressBar
	WidgetFrame
)

type WidgetState int32

const (
	StateNormal WidgetState = iota
	StateFocused
	StateHovered
	StatePressed
	StateDisabled
	StateSelected
)

// WidgetInfo is the renderer-facing snapshot of a widget.
type WidgetInfo struct {
	ID         uint32
	Name       string
	Bounds     Rect
	Kind       WidgetKind
	State      WidgetState
	HasCapture bool
	IsClipped  bool
	ClipRect   Rect
}

type Viewport struct {
	Viewport    Rect
	LogicalSize Vec2
}
