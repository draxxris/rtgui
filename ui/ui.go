// Package ui is the application-facing facade over rtgui widgets.
//
// Raylib still owns the OS window and frame loop. UI owns the rtgui state
// (transform, capture, theme, widget registry, focus, pressed) and translates
// polled raylib input into widget state plus OnClick/OnChange/OnText callbacks.
//
// Consumption model: HandleMouse and HandleKey return handled=true only when the
// UI actually used the input. The game runs its own camera/world/hotkey input
// when handled=false. Hover alone never consumes.
package ui

import (
	"rtgui/core"
	"rtgui/input"
	"rtgui/render"
	"rtgui/transform"
	"rtgui/widgets"
)

// UI owns one interface instance: transform, capture, theme, registry, focus,
// pressed, and callback maps. No package-global state, so multiple windows can
// each own a UI. Callers must drive it from a single goroutine (the frame loop);
// it has no internal mutex, matching the gallery reference.
type UI struct {
	transform *transform.Transform
	capture   *input.Capture
	theme     *render.Theme

	widgets map[string]*widgets.Widget
	order   []string

	focused *widgets.Widget
	pressed *widgets.Widget

	onClick  map[string]func()
	onChange map[string]func(float32)
	onText   map[string]func(string)
}

// New returns a UI that owns a fresh transform, capture, and theme.
// w and h fix the logical design resolution; later window resizes rescale
// around it instead of reflowing the layout.
// Non-positive dimensions fall back to 800x600 so the viewport stays valid.
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
	c := input.NewCapture()
	return NewWith(t, c, render.NewTheme(t))
}

// NewWith returns a UI borrowing the given transform, capture, and theme.
// Any nil argument is replaced with a fresh instance so the UI stays usable.
func NewWith(t *transform.Transform, c *input.Capture, th *render.Theme) *UI {
	if t == nil {
		t = transform.New(core.Viewport{})
	}
	if c == nil {
		c = input.NewCapture()
	}
	if th == nil {
		th = render.NewTheme(t)
	}
	return &UI{
		transform: t,
		capture:   c,
		theme:     th,
		widgets:   make(map[string]*widgets.Widget),
		onClick:   make(map[string]func()),
		onChange:  make(map[string]func(float32)),
		onText:    make(map[string]func(string)),
	}
}

// Add registers widgets by external Name in insertion order.
// Nil widgets and empty names are ignored. A duplicate name overwrites the
// entry without changing order, so layout rebuilds can re-add safely.
// This differs from sim.Register, which errors on duplicates.
func (u *UI) Add(list ...*widgets.Widget) {
	if u == nil {
		return
	}
	if u.widgets == nil {
		u.widgets = make(map[string]*widgets.Widget)
	}
	for _, w := range list {
		if w == nil || w.Name == "" {
			continue
		}
		if _, exists := u.widgets[w.Name]; !exists {
			u.order = append(u.order, w.Name)
		}
		u.widgets[w.Name] = w
	}
}

// Resize forwards the live window size in pixels to the facade. The logical
// design resolution is fixed at construction and never follows the window:
// resizing rescales the physical/logical mapping instead of reflowing widgets.
// Invalid dimensions are ignored. When the logical size was never set (a
// borrowed empty transform), the first valid Resize seeds it, fixing the
// design resolution from then on.
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
// Nil-safe: returns 1, 1 without a transform.
func (u *UI) Scale() (sx, sy float32) {
	if u == nil || u.transform == nil {
		return 1, 1
	}
	return u.transform.Scale()
}

// ToLogical maps a physical window point into logical UI coordinates.
// Callers pass already-mapped positions to HandleMouse; this helper keeps the
// PhysicalToViewport call in one place. Nil-safe: returns p unchanged.
func (u *UI) ToLogical(p core.Vec2) core.Vec2 {
	if u == nil || u.transform == nil {
		return p
	}
	return u.transform.PhysicalToViewport(p)
}

// Theme exposes the owned theme for skin setup and draw-log inspection.
func (u *UI) Theme() *render.Theme {
	if u == nil {
		return nil
	}
	return u.theme
}

// Capture exposes the owned pointer-capture for tests and sim sharing.
func (u *UI) Capture() *input.Capture {
	if u == nil {
		return nil
	}
	return u.capture
}

// Transform exposes the owned transform for tests.
func (u *UI) Transform() *transform.Transform {
	if u == nil {
		return nil
	}
	return u.transform
}

// Focused reports the currently focused widget, or nil when nothing has focus.
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
