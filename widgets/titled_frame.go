package widgets

import (
	"math"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
)

// DefaultTitleBarHeight is the standard titlebar height in logical pixels.
// Override per window with SetTitleBarHeight before Layout.
const DefaultTitleBarHeight = 32

const (
	titledCloseSize         = 24
	titledCloseRightInset   = 4
	titledCloseTopInset     = 4
	titledTitleLeftInset    = 10
	titledTitleRightReserve = 32
	titledDefaultInset      = 8
)

// TitledFrame is a builder for a titled window composed of five registered
// widgets: a root frame, a titlebar frame, a title label, a close button,
// and a content frame for caller-owned widgets. It is not a Widget itself:
// one Widget is one name plus one layout node, and this control needs five.
// Builders stay ui-free; the caller registers Widgets, calls Layout once,
// then parents content children by ContentName.
//
// Removal of the root through the UI invalidates the builder; do not reuse
// it after remove.
type TitledFrame struct {
	root           *Frame
	titleBar       *Frame
	title          *Label
	close          *Button
	content        *Frame
	onClose         func()
	titleBarHeight  float32
	insetLeft       float32
	insetTop        float32
	insetRight      float32
	insetBottom     float32
	titleInsetLeft  float32
	titleInsetRight float32
	contentTopGap   float32
	titleLeftInset  float32
	titleTopOffset  float32
	closeSize       float32
	closeRightInset float32
	closeTopInset   float32
	variant         string
}

// NewTitledFrame returns a titled window builder with the default "titled"
// variant, input-transparent chrome, and the close button wired to Close.
// Register Widgets, then call Layout. The name must be non-empty for UI
// registration; children derive name + "/titlebar", "/title", "/close",
// and "/content" by convention.
func NewTitledFrame(name string, bounds core.Rect, title string) *TitledFrame {
	t := &TitledFrame{
		root:           NewFrame(name, bounds),
		titleBar:       NewFrame(name+"/titlebar", core.Rect{}),
		title:          NewLabel(name+"/title", core.Rect{}, title),
		close:          NewButton(name+"/close", core.Rect{}, "X"),
		content:        NewFrame(name+"/content", core.Rect{}),
		titleBarHeight:  DefaultTitleBarHeight,
		insetLeft:       titledDefaultInset,
		insetTop:        titledDefaultInset,
		insetRight:      titledDefaultInset,
		insetBottom:     titledDefaultInset,
		titleInsetLeft:  titledDefaultInset,
		titleInsetRight: titledDefaultInset,
		titleLeftInset:  titledTitleLeftInset,
		closeSize:       titledCloseSize,
		closeRightInset: titledCloseRightInset,
		closeTopInset:   titledCloseTopInset,
		variant:         "titled",
	}
	t.titleBar.SetInputTransparent(true)
	t.title.SetInputTransparent(true)
	t.content.SetInputTransparent(true)
	t.close.SetTooltip("Close")
	t.applyVariant()
	t.close.OnClick(func() { t.Close() })
	return t
}

// Widgets returns the five owned widgets in draw order for UI registration.
func (t *TitledFrame) Widgets() []Widget {
	if t == nil {
		return nil
	}
	return []Widget{t.root, t.titleBar, t.title, t.close, t.content}
}

// Layout wires ownership and anchors, then arranges the root tree in one
// call. Call once after registration; call again after changing geometry or
// variant. Content children parented later need one further Arrange.
func (t *TitledFrame) Layout() error {
	if t == nil || t.root == nil {
		return ErrNilWidget
	}
	if err := t.attach(); err != nil {
		return err
	}
	if err := t.placeChrome(); err != nil {
		return err
	}
	if err := t.placeContent(); err != nil {
		return err
	}
	return t.Arrange()
}

// Arrange resolves the root tree after content children join.
func (t *TitledFrame) Arrange() error {
	if t == nil || t.root == nil {
		return ErrNilWidget
	}
	return layout.Arrange(t.root.Frame(), core.Rect{})
}

// attach parents the chrome under its owner. The five nodes are created as
// a unit, so one link check short-circuits repeated layouts. Content joins
// before the titlebar so body content draws first and the titlebar paints
// over it; the root frame border paints last via drawNode.
func (t *TitledFrame) attach() error {
	if t.titleBar.Frame().Parent() != nil {
		return nil
	}
	if err := t.root.Frame().AddChild(t.content.Frame()); err != nil {
		return err
	}
	if err := t.root.Frame().AddChild(t.titleBar.Frame()); err != nil {
		return err
	}
	if err := t.titleBar.Frame().AddChild(t.title.Frame()); err != nil {
		return err
	}
	return t.titleBar.Frame().AddChild(t.close.Frame())
}

// placeChrome anchors the titlebar across the root top and fits the title
// and close button inside it. Per-side insets keep the titlebar clear of
// asymmetric nine-patch rings such as the quest frame's 95px top band.
// The titlebar carries its own side insets because corner flourishes reach
// further into the title band than the side moldings reach into the body.
func (t *TitledFrame) placeChrome() error {
	t.titleBar.SetBounds(core.Rect{H: t.titleBarHeight})
	if err := t.titleBar.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: t.titleInsetLeft, Y: t.insetTop}); err != nil {
		return err
	}
	if err := t.titleBar.SetPoint(layout.AnchorTopRight, nil, layout.AnchorTopRight, core.Vec2{X: -t.titleInsetRight, Y: t.insetTop}); err != nil {
		return err
	}
	t.close.SetBounds(core.Rect{W: t.closeSize, H: t.closeSize})
	if err := t.close.SetPoint(layout.AnchorTopRight, nil, layout.AnchorTopRight, core.Vec2{X: -t.closeRightInset, Y: t.closeTopInset}); err != nil {
		return err
	}
	if err := t.title.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: t.titleLeftInset, Y: t.titleTopOffset}); err != nil {
		return err
	}
	return t.title.SetPoint(layout.AnchorBottomRight, nil, layout.AnchorBottomRight, core.Vec2{X: -titledTitleRightReserve})
}

// placeContent stretches the content frame from the titlebar bottom to the
// root bottom-right, clearing the root ring by the side and bottom insets.
// contentTopGap leaves room for rails drawn between the titlebar and the
// body, such as the quest frame's divider below its title band.
func (t *TitledFrame) placeContent() error {
	if err := t.content.SetPoint(layout.AnchorTopLeft, t.titleBar.Frame(), layout.AnchorBottomLeft, core.Vec2{X: t.insetLeft - t.titleInsetLeft, Y: t.contentTopGap}); err != nil {
		return err
	}
	return t.content.SetPoint(layout.AnchorBottomRight, nil, layout.AnchorBottomRight, core.Vec2{X: -t.insetRight, Y: -t.insetBottom})
}

// ContentName returns the content frame name for UI parenting calls.
func (t *TitledFrame) ContentName() string {
	if t == nil || t.content == nil {
		return ""
	}
	return t.content.Name()
}

// Content returns the content frame hosting caller-owned widgets.
func (t *TitledFrame) Content() *Frame {
	if t == nil {
		return nil
	}
	return t.content
}

// CloseButton returns the close button for styling. Do not retarget its
// click handler; register close interest with OnClose instead.
func (t *TitledFrame) CloseButton() *Button {
	if t == nil {
		return nil
	}
	return t.close
}

// Title returns the titlebar text.
func (t *TitledFrame) Title() string {
	if t == nil || t.title == nil {
		return ""
	}
	return t.title.Text()
}

// SetTitle replaces the titlebar text and reports a change.
func (t *TitledFrame) SetTitle(value string) bool {
	if t == nil || t.title == nil {
		return false
	}
	return t.title.SetText(value)
}

// SetTitleBarHeight overrides the titlebar height. Configure before Layout;
// after Layout, call Layout again.
func (t *TitledFrame) SetTitleBarHeight(height float32) *TitledFrame {
	if t == nil || math.IsNaN(float64(height)) || math.IsInf(float64(height), 0) || height <= 0 {
		return t
	}
	t.titleBarHeight = height
	return t
}

// SetTitleLeftInset controls the title's left inset inside the titlebar.
// Configure before Layout; after Layout, call Layout.
func (t *TitledFrame) SetTitleLeftInset(inset float32) *TitledFrame {
	if t == nil || math.IsNaN(float64(inset)) || math.IsInf(float64(inset), 0) || inset < 0 {
		return t
	}
	t.titleLeftInset = inset
	return t
}

// SetTitleTopOffset shifts the title vertically inside the titlebar.
// Configure before Layout; after Layout, call Layout.
func (t *TitledFrame) SetTitleTopOffset(offset float32) *TitledFrame {
	if t == nil || math.IsNaN(float64(offset)) || math.IsInf(float64(offset), 0) {
		return t
	}
	t.titleTopOffset = offset
	return t
}

// SetCloseButtonSize changes the square close-button size. Configure before
// Layout; after Layout, call Layout.
func (t *TitledFrame) SetCloseButtonSize(size float32) *TitledFrame {
	if t == nil || math.IsNaN(float64(size)) || math.IsInf(float64(size), 0) || size <= 0 {
		return t
	}
	t.closeSize = size
	return t
}

// SetBorderInset offsets chrome and content from the root edge so children
// clear the root nine-patch ring. Keep it equal to the root CSS
// border-image-slice. Configure before Layout; after Layout, call Layout.
func (t *TitledFrame) SetBorderInset(inset float32) *TitledFrame {
	return t.SetBorderInsets(inset, inset, inset, inset)
}

// SetBorderInsets offsets chrome per side as left, top, right, bottom so
// asymmetric rings fit: the quest frame uses wide side and bottom slices
// with a taller top band. Negative values sanitize to zero. Configure
// before Layout; after Layout, call Layout.
func (t *TitledFrame) SetBorderInsets(left, top, right, bottom float32) *TitledFrame {
	if t == nil {
		return t
	}
	t.insetLeft = sanitizeInset(left)
	t.insetTop = sanitizeInset(top)
	t.insetRight = sanitizeInset(right)
	t.insetBottom = sanitizeInset(bottom)
	t.titleInsetLeft = t.insetLeft
	t.titleInsetRight = t.insetRight
	return t
}

// SetTitleBarSideInsets offsets the titlebar from the root edge
// independently of the content sides. Corner flourishes intrude further
// into the title band than side moldings intrude into the body, so ornate
// frames need a wider titlebar margin. Configure before Layout; after
// Layout, call Layout.
func (t *TitledFrame) SetTitleBarSideInsets(left, right float32) *TitledFrame {
	if t == nil {
		return t
	}
	t.titleInsetLeft = sanitizeInset(left)
	t.titleInsetRight = sanitizeInset(right)
	return t
}

// sanitizeInset clamps one border inset to a finite non-negative value.
func sanitizeInset(inset float32) float32 {
	if math.IsNaN(float64(inset)) || math.IsInf(float64(inset), 0) || inset < 0 {
		return 0
	}
	return inset
}

// SetContentTopGap leaves room between the titlebar bottom and the content
// top for divider rails drawn by the root nine-patch. Configure before
// Layout; after Layout, call Layout.
func (t *TitledFrame) SetContentTopGap(gap float32) *TitledFrame {
	if t == nil || math.IsNaN(float64(gap)) || math.IsInf(float64(gap), 0) || gap < 0 {
		return t
	}
	t.contentTopGap = gap
	return t
}

// SetCloseInset offsets the close button from the titlebar top-right by a
// right and top margin. Negative values allow corner overlap on ornate
// frames. Configure before Layout; after Layout, call Layout.
func (t *TitledFrame) SetCloseInset(right, top float32) *TitledFrame {
	if t == nil || math.IsNaN(float64(right)) || math.IsInf(float64(right), 0) || math.IsNaN(float64(top)) || math.IsInf(float64(top), 0) {
		return t
	}
	t.closeRightInset = right
	t.closeTopInset = top
	return t
}

// SetVariant derives per-part CSS classes as style + "-frame", "-titlebar",
// "-title", "-close", and "-content". Empty styles are ignored. Configure
// before Layout; after Layout, call Layout again.
func (t *TitledFrame) SetVariant(style string) *TitledFrame {
	if t == nil || style == "" {
		return t
	}
	t.variant = style
	t.applyVariant()
	return t
}

// applyVariant writes the current variant classes onto the owned widgets.
func (t *TitledFrame) applyVariant() {
	if t == nil {
		return
	}
	t.root.SetClass(t.variant + "-frame")
	t.titleBar.SetClass(t.variant + "-titlebar")
	t.title.SetClass(t.variant + "-title")
	t.close.SetClass(t.variant + "-close")
	t.content.SetClass(t.variant + "-content")
}

// OnClose registers the callback fired by Close after hiding. It replaces
// any previous callback; nil clears it.
func (t *TitledFrame) OnClose(fn func()) *TitledFrame {
	if t != nil {
		t.onClose = fn
	}
	return t
}

// Show reveals the window subtree.
func (t *TitledFrame) Show() {
	if t != nil && t.root != nil {
		t.root.SetVisible(true)
	}
}

// Hide conceals the window subtree without firing OnClose.
func (t *TitledFrame) Hide() {
	if t != nil && t.root != nil {
		t.root.SetVisible(false)
	}
}

// IsOpen reports whether the window subtree is visible.
func (t *TitledFrame) IsOpen() bool {
	return t != nil && t.root != nil && t.root.Visible()
}

// Toggle closes an open window and shows a hidden one.
func (t *TitledFrame) Toggle() {
	if !t.IsOpen() {
		t.Show()
		return
	}
	t.Close()
}

// Close hides the window, then fires OnClose. The X button shares this path.
func (t *TitledFrame) Close() {
	if t == nil || t.root == nil {
		return
	}
	t.Hide()
	if t.onClose != nil {
		t.onClose()
	}
}
