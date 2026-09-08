// Package widgets provides modular, stateful widget types without owning UI interaction state.
package widgets

import (
	"errors"
	"fmt"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
)

var (
	// ErrNilWidget reports an operation attempted on a nil widget.
	ErrNilWidget = errors.New("widgets: nil widget")
	// ErrKindMismatch reports an attempt to cast a widget to the wrong concrete type.
	ErrKindMismatch = errors.New("widgets: kind mismatch")
)

// Widget is the universal interface implemented by all concrete widgets.
// UI-owned hover, press, and focus state is deliberately not stored in widgets.
type Widget interface {
	Name() string
	Kind() core.WidgetKind
	Bounds() core.Rect
	SetBounds(bounds core.Rect)
	Frame() *layout.Node
	SetPoint(source layout.Anchor, target *layout.Node, targetPoint layout.Anchor, offset core.Vec2) error
	Enabled() bool
	SetEnabled(enabled bool) bool
	Visible() bool
	SetVisible(bool)
	InputTransparent() bool
	SetInputTransparent(bool)
	Callbacks() *Callbacks
	Owner() any
	SetOwner(any)
	HitTest(pos core.Vec2) bool
	Snapshot(state core.WidgetState) core.WidgetInfo
	Text() string
	SetText(value string) bool
	Tooltip() string
	SetTooltipText(string)
}

// RichProvider is implemented by every widget carrying optional rich runs.
// The base implementation serves all concrete widgets; Textbox editing
// stays plain and ignores rich runs for input.
type RichProvider interface {
	HasRichText() bool
	RichSegments() []core.RichSegment
	// CopyRichSegmentsInto copies runs into caller-owned storage for
	// allocation-free draw paths; see UI.richSegScratch.
	CopyRichSegmentsInto(dst []core.RichSegment) []core.RichSegment
}

// Callbacks is the single callback registry owned by a widget. UI registration
// and fluent widget setters replace the same slots; they never add listeners.
type Callbacks struct {
	Click       func()
	Change      func(float32)
	Text        func(string)
	TabSelect   func(int)
	LinkClick   func(core.Link)
	LinkTooltip func(core.Link) string
}

// base provides common fields and standard Widget implementation for concrete widgets.
// Every widget carries optional rich segments; plain text renders when none
// are set. Links stay interactive only in RichText; other widgets render
// icons, colors, and emphasis without link activation.
type base struct {
	name             string
	kind             core.WidgetKind
	frame            *layout.Node
	enabled          bool
	text             string
	callbacks        Callbacks
	hidden           bool
	inputTransparent bool
	owner            any
	tooltip          string
	textColor        core.Color
	hasTextColor     bool
	fontSize         float32
	italic           bool
	align            core.TextAlign
	richSegments     []core.RichSegment
	richRevision     uint64
}

// newBase initializes common widget fields.
func newBase(name string, kind core.WidgetKind, bounds core.Rect) base {
	return base{
		name:    name,
		kind:    kind,
		frame:   layout.New(name, bounds),
		enabled: true,
	}
}

// Name returns the widget's immutable external registry identity.
func (b *base) Name() string {
	if b == nil {
		return ""
	}
	return b.name
}

// Kind returns the widget's immutable tagged kind.
func (b *base) Kind() core.WidgetKind {
	if b == nil {
		return core.WidgetKind(-1)
	}
	return b.kind
}

// Bounds returns resolved frame bounds after arrangement or authored bounds.
func (b *base) Bounds() core.Rect {
	if b == nil || b.frame == nil {
		return core.Rect{}
	}
	return b.frame.Bounds()
}

// SetBounds replaces the widget frame's authored bounds.
func (b *base) SetBounds(bounds core.Rect) {
	if b != nil && b.frame != nil {
		b.frame.SetBounds(bounds)
	}
}

// Frame returns the widget's layout node for ownership-tree construction.
func (b *base) Frame() *layout.Node {
	if b == nil {
		return nil
	}
	return b.frame
}

// SetPoint adds or replaces one typed relation on the widget frame.
func (b *base) SetPoint(source layout.Anchor, target *layout.Node, targetPoint layout.Anchor, offset core.Vec2) error {
	if b == nil || b.frame == nil {
		return layout.ErrNilNode
	}
	return b.frame.SetPoint(source, target, targetPoint, offset)
}

// Enabled reports whether the widget accepts interaction.
func (b *base) Enabled() bool {
	return b != nil && b.enabled
}

// SetEnabled changes whether the widget accepts interaction and reports a change.
func (b *base) SetEnabled(enabled bool) bool {
	if b == nil || b.enabled == enabled {
		return false
	}
	b.enabled = enabled
	return true
}

// HitTest reports whether a logical point lies in the widget's effective bounds.
func (b *base) HitTest(pos core.Vec2) bool {
	return b != nil && b.Bounds().Contains(pos)
}

// Snapshot returns the renderer-facing metadata using UI-computed visual state.
func (b *base) Snapshot(state core.WidgetState) core.WidgetInfo {
	if b == nil {
		return core.WidgetInfo{}
	}
	info := core.WidgetInfo{
		Name:     b.name,
		Bounds:   b.Bounds(),
		Kind:     b.kind,
		State:    state,
		FontSize: b.fontSize,
		Italic:   b.italic,
		Align:    b.align,
	}
	if b.hasTextColor {
		info.TextColor = b.textColor
		info.HasTextColor = true
	}
	return info
}

// Text returns the plain widget text, or the rich plain-text fallback when
// rich segments are set. Icons contribute no characters.
func (b *base) Text() string {
	if b == nil {
		return ""
	}
	if len(b.richSegments) > 0 {
		text := ""
		for _, segment := range b.richSegments {
			text += segment.Text
		}
		return text
	}
	return b.text
}

// SetText replaces plain widget text, clears any rich segments, and
// reports a change.
func (b *base) SetText(value string) bool {
	if b == nil {
		return false
	}
	changed := b.text != value
	b.text = value
	if len(b.richSegments) > 0 {
		clear(b.richSegments)
		b.richSegments = b.richSegments[:0]
		b.richRevision++
		if b.richRevision == 0 {
			b.richRevision = 1
		}
		changed = true
	}
	return changed
}

// SetOnClick stores the widget's direct activation callback.
func (b *base) SetOnClick(fn func()) {
	if b != nil {
		b.callbacks.Click = fn
	}
}

// OnClickHandler returns the widget's direct activation callback.
func (b *base) OnClickHandler() func() {
	if b == nil {
		return nil
	}
	return b.callbacks.Click
}

// Callbacks exposes the widget-owned callback slots on the UI goroutine.
func (b *base) Callbacks() *Callbacks { return &b.callbacks }

// Owner reports the UI registration owner, or nil when detached.
func (b *base) Owner() any { return b.owner }

// SetOwner is reserved for UI registration and disposal on the owning goroutine.
func (b *base) SetOwner(owner any) { b.owner = owner }

// Visible reports local visibility. UI traversal also checks all ancestors.
func (b *base) Visible() bool { return b != nil && !b.hidden }

// SetVisible changes local visibility without destroying widget state.
func (b *base) SetVisible(visible bool) { b.hidden = !visible }

// InputTransparent reports whether empty widget space passes pointer input.
func (b *base) InputTransparent() bool { return b.inputTransparent }

// SetInputTransparent opts decorative overlays out of pointer blocking.
func (b *base) SetInputTransparent(value bool) { b.inputTransparent = value }

// SetTooltip attaches a hover tooltip string directly to the widget.
func (b *base) SetTooltip(text string) {
	if b != nil {
		b.tooltip = text
	}
}

// SetTooltipText replaces the same tooltip slot used by fluent widget setters.
func (b *base) SetTooltipText(text string) { b.tooltip = text }

// Tooltip returns the widget's hover tooltip string.
func (b *base) Tooltip() string {
	if b == nil {
		return ""
	}
	return b.tooltip
}

// SetTextColor configures an explicit text color for the widget.
func (b *base) SetTextColor(c core.Color) {
	if b != nil {
		b.textColor = c
		b.hasTextColor = true
	}
}

// TextColor returns the configured text color and whether one was set.
func (b *base) TextColor() (core.Color, bool) {
	if b == nil || !b.hasTextColor {
		return core.Color{}, false
	}
	return b.textColor, true
}

// SetFontSize sets an explicit font size in pixels for text rendering.
func (b *base) SetFontSize(size float32) {
	if b != nil {
		b.fontSize = size
	}
}

// FontSize returns the explicit font size, or 0 if auto-derived from bounds.
func (b *base) FontSize() float32 {
	if b == nil {
		return 0
	}
	return b.fontSize
}

// SetItalic configures whether the widget text uses the italic theme font.
func (b *base) SetItalic(italic bool) {
	if b != nil {
		b.italic = italic
	}
}

// Italic reports whether the widget text uses the italic theme font.
func (b *base) Italic() bool {
	if b == nil {
		return false
	}
	return b.italic
}

// SetAlign configures the horizontal text alignment within content bounds.
func (b *base) SetAlign(align core.TextAlign) {
	if b != nil {
		b.align = align
	}
}

// Align reports the horizontal text alignment within content bounds.
func (b *base) Align() core.TextAlign {
	if b == nil {
		return core.AlignLeft
	}
	return b.align
}

// HasRichText reports whether the widget carries rich display segments.
func (b *base) HasRichText() bool {
	return b != nil && len(b.richSegments) > 0
}

// RichSegments returns a safe snapshot of the rich display segments.
func (b *base) RichSegments() []core.RichSegment {
	if b == nil {
		return nil
	}
	return append([]core.RichSegment(nil), b.richSegments...)
}

// SetRichSegments copies rich display segments and reports a change.
// Plain text is retained as fallback but rich wins for drawing and Text.
func (b *base) SetRichSegments(segments []core.RichSegment) bool {
	if b == nil {
		return false
	}
	if core.EqualRichSegments(b.richSegments, segments) {
		return false
	}
	oldLen := len(b.richSegments)
	b.richSegments = append(b.richSegments[:0], segments...)
	if len(b.richSegments) < oldLen && cap(b.richSegments) >= oldLen {
		full := b.richSegments[:oldLen]
		clear(full[len(b.richSegments):oldLen])
		b.richSegments = full[:len(b.richSegments)]
	}
	b.richRevision++
	if b.richRevision == 0 {
		b.richRevision = 1
	}
	return true
}

// ClearRichText drops rich display segments and reports a change.
func (b *base) ClearRichText() bool {
	if b == nil || len(b.richSegments) == 0 {
		return false
	}
	clear(b.richSegments)
	b.richSegments = b.richSegments[:0]
	b.richRevision++
	if b.richRevision == 0 {
		b.richRevision = 1
	}
	return true
}

// RichRevision returns the revision bumped by rich segment changes.
func (b *base) RichRevision() uint64 {
	if b == nil {
		return 0
	}
	return b.richRevision
}

// RichSegmentCount returns the rich segment count without copying.
func (b *base) RichSegmentCount() int {
	if b == nil {
		return 0
	}
	return len(b.richSegments)
}

// RichSegmentAt returns a copy of the indexed rich segment.
func (b *base) RichSegmentAt(index int) (core.RichSegment, bool) {
	if b == nil || index < 0 || index >= len(b.richSegments) {
		return core.RichSegment{}, false
	}
	return b.richSegments[index], true
}

// CopyRichSegmentsInto copies rich segments into reused storage.
func (b *base) CopyRichSegmentsInto(dst []core.RichSegment) []core.RichSegment {
	if b == nil {
		clearRichSegments(dst)
		return dst[:0]
	}
	return copyRichSegmentsInto(dst, b.richSegments)
}

// RichPlainText concatenates rich segment text for search and fallback.
func (b *base) RichPlainText() string {
	if b == nil {
		return ""
	}
	text := ""
	for _, segment := range b.richSegments {
		text += segment.Text
	}
	return text
}

// As casts widget w to the target type T, returning ErrKindMismatch if w is not of type T.
func As[T Widget](w Widget) (T, error) {
	var zero T
	if w == nil {
		return zero, ErrNilWidget
	}
	if val, ok := w.(T); ok {
		return val, nil
	}
	return zero, fmt.Errorf("%w: widget %q cannot be cast to requested type", ErrKindMismatch, w.Name())
}

// AsButton asserts that w is a *Button.
func AsButton(w Widget) (*Button, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if b, ok := w.(*Button); ok {
		return b, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetButton)
}

// AsLabel asserts that w is a *Label.
func AsLabel(w Widget) (*Label, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if l, ok := w.(*Label); ok {
		return l, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetLabel)
}

// AsCheckbox asserts that w is a *Checkbox.
func AsCheckbox(w Widget) (*Checkbox, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if cb, ok := w.(*Checkbox); ok {
		return cb, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetCheckbox)
}

// AsSlider asserts that w is a *Slider.
func AsSlider(w Widget) (*Slider, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if s, ok := w.(*Slider); ok {
		return s, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetSlider)
}

// AsProgressBar asserts that w is a *ProgressBar.
func AsProgressBar(w Widget) (*ProgressBar, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if p, ok := w.(*ProgressBar); ok {
		return p, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetProgressBar)
}

// AsTextbox asserts that w is a *Textbox.
func AsTextbox(w Widget) (*Textbox, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if tb, ok := w.(*Textbox); ok {
		return tb, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetTextbox)
}

// AsScrollPanel asserts that w is a *ScrollPanel.
func AsScrollPanel(w Widget) (*ScrollPanel, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if sp, ok := w.(*ScrollPanel); ok {
		return sp, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetScrollPanel)
}

// AsDropdown asserts that w is a *Dropdown.
func AsDropdown(w Widget) (*Dropdown, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if dd, ok := w.(*Dropdown); ok {
		return dd, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetDropdown)
}

// AsTabBar asserts that w is a *TabBar.
func AsTabBar(w Widget) (*TabBar, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if tb, ok := w.(*TabBar); ok {
		return tb, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetTabBar)
}

// AsRichText asserts that w is a *RichText.
func AsRichText(w Widget) (*RichText, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if rt, ok := w.(*RichText); ok {
		return rt, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetRichText)
}

// AsCanvas asserts that w is a *Canvas.
func AsCanvas(w Widget) (*Canvas, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if c, ok := w.(*Canvas); ok {
		return c, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetCanvas)
}

// AsFrame asserts that w is a *Frame.
func AsFrame(w Widget) (*Frame, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if f, ok := w.(*Frame); ok {
		return f, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetFrame)
}
