// Package ui owns widget interaction, registration, callbacks, and rendering.
//
// Raylib owns the OS window and frame loop. A UI must be driven from one
// goroutine and translates physical or semantic input into the same widget
// mutations and synchronous callbacks.
package ui

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/dragdrop"
	"github.com/draxxris/rtgui/layout"
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
	// ErrWidgetOwned reports registration of a widget owned by another UI.
	ErrWidgetOwned = errors.New("ui: widget already has a UI owner")
)

// UI owns one interface instance. Hover, press, and focus have exactly one
// owner each and are never mirrored into widgets. Textbox caret and
// selection are the documented exception: they live in the widget's text
// buffer as part of the editing model, while focus gates the caret display
// and focus transfer or loss clears the selection. Container focus is a
// second independent slot: the active frame highlights its bounds and
// scopes frame-bound hotkeys while keyboard focus keeps editing rights.
type UI struct {
	transform *transform.Transform
	theme     *render.Theme

	widgets map[string]widgets.Widget
	order   []string

	hovered widgets.Widget
	pressed widgets.Widget
	focused widgets.Widget
	// activeFrame is the container focus slot. It highlights its bounds
	// and scopes frame-bound hotkeys while keyboard focus keeps editing.
	activeFrame widgets.Widget

	scrollThumbHovered    *widgets.ScrollPanel
	scrollThumbDragging   *widgets.ScrollPanel
	scrollDragStartY      float32
	scrollDragStartScroll float32

	// hotkeys is the library-owned registry for scoped global actions.
	// Iteration is linear and allocation-free on the hot path; N stays tiny.
	hotkeys []hotkeyEntry
	pointer core.Vec2
	// clipboard is the in-memory fallback used headless; windowed clipboard
	// access goes through the system via render with this as mirror.
	clipboard string

	byNode        map[*layout.Node]widgets.Widget
	drag          *dragdrop.Controller
	dragSource    widgets.Widget
	dragSources   map[string]func() dragdrop.Payload
	dropTargets   map[string]*dragdrop.DropTarget
	dragGhost     func(dragdrop.Ghost)
	mouseCaptured bool
	stringScratch []string
	charScratch   []rune
	keyScratch    []rune
	richCaches    map[string]*render.RichLayoutCache
	// richSegScratch reuses segment storage for single-line control draws.
	// Draws consume it synchronously on the owning goroutine.
	richSegScratch  []core.RichSegment
	richTips        map[string]core.RichTooltip
	explicitRichTip core.RichTooltip
	richTipCache    render.RichTooltipCache
	tipRevision     uint64

	menuItems          []core.MenuItem
	menuBounds         core.Rect
	menuOnSelect       func(string)
	menuArmed          int
	menuDown           bool
	contextMenuHandler func(core.Vec2)

	tooltipText    string
	explicitAnchor core.Vec2

	// tooltipDelay is the global hover dwell before a hover-derived
	// tooltip may draw. Non-positive means immediate.
	tooltipDelay time.Duration
	// tooltipAnchor is the global hover anchor; zero follows the cursor.
	tooltipAnchor TooltipAnchor
	// tooltipOpts holds per-widget delay and anchor overrides.
	tooltipOpts map[string]TooltipOptions
	// tooltipClock supplies hover-dwell time; nil selects time.Now.
	tooltipClock func() time.Time
	// hoverSince stamps the last hover-owner or link-cell change.
	hoverSince time.Time

	linkArmedSeg int
	tipWidget    widgets.Widget
	tipSeg       int
	tipText      string

	diagnostics []string
	diagHandler DiagnosticHandler
}

// DiagnosticHandler handles diagnostic messages from UI operations.
type DiagnosticHandler func(msg string)

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
		transform:    t,
		theme:        th,
		widgets:      make(map[string]widgets.Widget),
		byNode:       make(map[*layout.Node]widgets.Widget),
		linkArmedSeg: -1,
		tipSeg:       -1,
	}
}

// Add validates and registers every widget atomically in insertion order.
// Nil widgets, empty names, and duplicate names return meaningful errors
// without changing the registry.
func (u *UI) Add(list ...widgets.Widget) error {
	if u == nil {
		return ErrNilUI
	}
	seen := make(map[string]struct{}, len(list))
	nodes := make(map[*layout.Node]bool, len(list))
	for _, widget := range list {
		if isNilWidget(widget) {
			return ErrNilWidget
		}
		if err := u.validateOwnership(widget); err != nil {
			return err
		}
		if nodes[widget.Frame()] {
			return ErrDuplicateWidget
		}
		nodes[widget.Frame()] = true
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
		u.widgets = make(map[string]widgets.Widget, len(list))
	}
	if u.byNode == nil {
		u.byNode = make(map[*layout.Node]widgets.Widget)
	}
	for _, widget := range list {
		name := widget.Name()
		u.widgets[name] = widget
		u.byNode[widget.Frame()] = widget
		widget.SetOwner(u)
		u.order = append(u.order, name)
	}
	return nil
}

// validateOwnership prevents shared callback state and unusable layout frames.
func (u *UI) validateOwnership(widget widgets.Widget) error {
	if widget.Frame() == nil {
		return fmt.Errorf("ui: widget %q has no layout frame", widget.Name())
	}
	if widget.Owner() != nil && widget.Owner() != u {
		return ErrWidgetOwned
	}
	if other := u.byNode[widget.Frame()]; other != nil && other != widget {
		return fmt.Errorf("ui: widgets share layout frame %q", widget.Name())
	}
	return nil
}

// Remove disposes a widget and its registered descendants, including callbacks.
func (u *UI) Remove(name string) bool {
	if u == nil || u.widgets == nil {
		return false
	}
	widget, exists := u.widgets[name]
	if !exists {
		return false
	}
	u.removeTree(widget.Frame())
	if p := widget.Frame().Parent(); p != nil {
		p.RemoveChild(widget.Frame())
	}
	return true
}

// ClearWidgets disposes all registered widgets and transient interaction owners.
func (u *UI) ClearWidgets() {
	if u == nil {
		return
	}
	for len(u.order) > 0 {
		u.Remove(u.order[len(u.order)-1])
	}
	u.CancelInput()
	clear(u.widgets)
	clear(u.byNode)
	u.order = nil
	u.linkArmedSeg = -1
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
func (u *UI) Hovered() widgets.Widget {
	if u == nil {
		return nil
	}
	return u.hovered
}

// Pressed reports the widget owning the active press gesture.
func (u *UI) Pressed() widgets.Widget {
	if u == nil {
		return nil
	}
	return u.pressed
}

// Focused reports the currently focused widget.
func (u *UI) Focused() widgets.Widget {
	if u == nil {
		return nil
	}
	return u.focused
}

// Lookup returns the widget registered under name, or nil when unknown.
func (u *UI) Lookup(name string) widgets.Widget {
	if u == nil || u.widgets == nil {
		return nil
	}
	return u.widgets[name]
}

// Pointer returns the logical pointer position from the latest mouse event.
func (u *UI) Pointer() core.Vec2 {
	if u == nil {
		return core.Vec2{}
	}
	return u.pointer
}

// clearReferences removes widget from each transient owner slot, forgetting
// any textbox selection held by a focused removal.
func (u *UI) clearReferences(widget widgets.Widget) {
	if u == nil || widget == nil {
		return
	}
	if u.hovered == widget {
		u.setHovered(nil)
	}
	if u.pressed == widget {
		u.pressed = nil
		u.linkArmedSeg = -1
	}
	if u.focused == widget {
		u.clearFocus()
	}
	if u.activeFrame == widget {
		u.clearActiveFrame()
	}
	if u.tipWidget == widget {
		u.clearLinkTip()
	}
	if u.scrollThumbDragging == widget {
		u.scrollThumbDragging = nil
	}
	if u.scrollThumbHovered == widget {
		u.scrollThumbHovered = nil
	}
}

// SetDiagnosticHandler configures an optional callback for runtime warnings.
func (u *UI) SetDiagnosticHandler(handler DiagnosticHandler) {
	if u != nil {
		u.diagHandler = handler
		if u.theme != nil {
			if handler != nil {
				u.theme.SetDiagnosticHandler(func(msg string) {
					u.diagnose("%s", msg)
				})
			} else {
				u.theme.SetDiagnosticHandler(nil)
			}
		}
	}
}

// SetDebugMode enables or disables visible placeholder rendering for missing skins.
func (u *UI) SetDebugMode(enabled bool) {
	if u != nil && u.theme != nil {
		u.theme.SetDebugMode(enabled)
	}
}

// DebugMode reports whether visible placeholder rendering for missing skins is enabled.
func (u *UI) DebugMode() bool {
	return u != nil && u.theme != nil && u.theme.DebugMode()
}

// Diagnostics returns a snapshot of recorded diagnostic warnings.
func (u *UI) Diagnostics() []string {
	if u == nil {
		return nil
	}
	return append([]string(nil), u.diagnostics...)
}

// ClearDiagnostics clears recorded diagnostic messages.
func (u *UI) ClearDiagnostics() {
	if u != nil {
		u.diagnostics = nil
	}
}

// diagnose records and reports a diagnostic message.
func (u *UI) diagnose(format string, args ...any) {
	if u == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	for _, old := range u.diagnostics {
		if old == msg {
			return
		}
	}
	if len(u.diagnostics) == 128 {
		copy(u.diagnostics, u.diagnostics[1:])
		u.diagnostics = u.diagnostics[:127]
	}
	u.diagnostics = append(u.diagnostics, msg)
	if u.diagHandler != nil {
		u.diagHandler(msg)
	}
}

// isNilWidget tests whether an interface or its underlying pointer value is nil.
func isNilWidget(w widgets.Widget) bool {
	if w == nil {
		return true
	}
	v := reflect.ValueOf(w)
	return v.Kind() == reflect.Pointer && v.IsNil()
}
