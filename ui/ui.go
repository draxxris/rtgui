// Package ui owns widget interaction, registration, callbacks, and rendering.
//
// Raylib owns the OS window and frame loop. A UI must be driven from one
// goroutine and translates physical or semantic input into the same widget
// mutations and synchronous callbacks.
package ui

import (
	"errors"
	"fmt"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/transform"
	"github.com/draxxris/rtgui/widgets"
)

var (
	// ErrNilUI reports registration attempted through a nil UI.
	ErrNilUI = errors.New("ui: nil UI")
	// ErrNilWidget reports a nil widget in an Add request.
	ErrNilWidget = errors.New("ui: nil widget")
	// ErrEmptyWidgetName reports a widget without an external registry name.
	ErrEmptyWidgetName = errors.New("ui: empty widget name")
	// ErrDuplicateWidget reports a name already present in the registry or Add request.
	ErrDuplicateWidget = errors.New("ui: duplicate widget name")
)

type callbackRecord struct {
	onClick  func()
	onChange func(float32)
	onText   func(string)
}

// UI owns one interface instance. Hover, press, and focus have exactly one
// owner each and are never mirrored into widgets.
type UI struct {
	transform *transform.Transform
	theme     *render.Theme

	widgets map[string]*widgets.Widget
	order   []string

	hovered *widgets.Widget
	pressed *widgets.Widget
	focused *widgets.Widget
	pointer core.Vec2

	callbacks map[string]callbackRecord
}

// New returns a UI with a fixed logical design resolution. Non-positive
// dimensions fall back to 800x600 so the viewport remains usable.
func New(w, h int) *UI {
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}
	viewport := core.Viewport{
		Viewport:    core.Rect{W: float32(w), H: float32(h)},
		LogicalSize: core.Vec2{X: float32(w), Y: float32(h)},
	}
	t := transform.New(viewport)
	return NewWith(t, render.NewTheme(t))
}

// NewWith returns a UI borrowing the transform and theme. Nil arguments are
// replaced with usable instances.
func NewWith(t *transform.Transform, th *render.Theme) *UI {
	if t == nil {
		t = transform.New(core.Viewport{})
	}
	if th == nil {
		th = render.NewTheme(t)
	}
	return &UI{
		transform: t,
		theme:     th,
		widgets:   make(map[string]*widgets.Widget),
		callbacks: make(map[string]callbackRecord),
	}
}

// Add validates and registers every widget atomically in insertion order.
// Nil widgets, empty names, and duplicate names return meaningful errors
// without changing the registry.
func (u *UI) Add(list ...*widgets.Widget) error {
	if u == nil {
		return ErrNilUI
	}
	seen := make(map[string]struct{}, len(list))
	for _, widget := range list {
		if widget == nil {
			return ErrNilWidget
		}
		name := widget.Name()
		if name == "" {
			return ErrEmptyWidgetName
		}
		if _, exists := u.widgets[name]; exists {
			return fmt.Errorf("%w: %q", ErrDuplicateWidget, name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("%w: %q", ErrDuplicateWidget, name)
		}
		seen[name] = struct{}{}
	}
	if u.widgets == nil {
		u.widgets = make(map[string]*widgets.Widget, len(list))
	}
	for _, widget := range list {
		name := widget.Name()
		u.widgets[name] = widget
		u.order = append(u.order, name)
	}
	return nil
}

// Remove deletes a named widget, clears every active reference to it, and
// preserves callback registrations for a later widget with the same name.
func (u *UI) Remove(name string) bool {
	if u == nil || u.widgets == nil {
		return false
	}
	widget, exists := u.widgets[name]
	if !exists {
		return false
	}
	delete(u.widgets, name)
	u.clearReferences(widget)
	for index, orderedName := range u.order {
		if orderedName != name {
			continue
		}
		copy(u.order[index:], u.order[index+1:])
		u.order = u.order[:len(u.order)-1]
		break
	}
	return true
}

// ClearWidgets removes all widgets and transient owners while retaining named callbacks.
func (u *UI) ClearWidgets() {
	if u == nil {
		return
	}
	u.widgets = make(map[string]*widgets.Widget)
	u.order = nil
	u.hovered = nil
	u.pressed = nil
	u.focused = nil
}

// Resize updates physical dimensions while retaining the fixed logical size.
// Invalid dimensions are ignored. An empty borrowed transform receives its
// logical design size from the first valid call.
func (u *UI) Resize(w, h int) {
	if u == nil || w <= 0 || h <= 0 {
		return
	}
	if u.transform != nil && (u.transform.Viewport.LogicalSize.X <= 0 || u.transform.Viewport.LogicalSize.Y <= 0) {
		_ = u.transform.SetViewport(core.Viewport{
			Viewport:    core.Rect{W: float32(w), H: float32(h)},
			LogicalSize: core.Vec2{X: float32(w), Y: float32(h)},
		})
	} else if u.transform != nil {
		viewport := u.transform.Viewport
		viewport.Viewport = core.Rect{W: float32(w), H: float32(h)}
		_ = u.transform.SetViewport(viewport)
	}
	if u.theme != nil && u.transform != nil {
		_ = u.theme.SetViewport(u.transform.Viewport)
	}
}

// Scale reports the physical-per-logical stretch from the owned transform.
func (u *UI) Scale() (sx, sy float32) {
	if u == nil || u.transform == nil {
		return 1, 1
	}
	return u.transform.Scale()
}

// ToLogical maps a physical window point into logical UI coordinates.
func (u *UI) ToLogical(point core.Vec2) core.Vec2 {
	if u == nil || u.transform == nil {
		return point
	}
	return u.transform.PhysicalToViewport(point)
}

// Theme exposes the owned theme for skin setup and drawing configuration.
func (u *UI) Theme() *render.Theme {
	if u == nil {
		return nil
	}
	return u.theme
}

// Transform exposes the owned logical-to-physical transform.
func (u *UI) Transform() *transform.Transform {
	if u == nil {
		return nil
	}
	return u.transform
}

// Hovered reports the currently hovered topmost eligible widget.
func (u *UI) Hovered() *widgets.Widget {
	if u == nil {
		return nil
	}
	return u.hovered
}

// Pressed reports the widget owning the active press gesture.
func (u *UI) Pressed() *widgets.Widget {
	if u == nil {
		return nil
	}
	return u.pressed
}

// Focused reports the currently focused widget.
func (u *UI) Focused() *widgets.Widget {
	if u == nil {
		return nil
	}
	return u.focused
}

// Lookup returns the widget registered under name, or nil when unknown.
func (u *UI) Lookup(name string) *widgets.Widget {
	if u == nil || u.widgets == nil {
		return nil
	}
	return u.widgets[name]
}

// clearReferences removes widget from each transient owner slot.
func (u *UI) clearReferences(widget *widgets.Widget) {
	if u.hovered == widget {
		u.hovered = nil
	}
	if u.pressed == widget {
		u.pressed = nil
	}
	if u.focused == widget {
		u.focused = nil
	}
}
