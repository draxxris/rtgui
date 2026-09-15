package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// Button is an interactive button widget.
type Button struct {
	base
}

// NewButton returns an enabled button with immutable name and kind.
func NewButton(name string, bounds core.Rect, label string) *Button {
	b := newBase(name, core.WidgetButton, bounds)
	b.text = label
	return &Button{base: b}
}

// OnClick attaches a synchronous activation callback directly to the button.
func (b *Button) OnClick(fn func()) *Button {
	b.SetOnClick(fn)
	return b
}

// SetTooltip attaches a hover tooltip string directly to the button.
func (b *Button) SetTooltip(text string) *Button {
	b.base.SetTooltip(text)
	return b
}

// SetTextColor configures an explicit text color for the button.
func (b *Button) SetTextColor(c core.Color) *Button {
	b.base.SetTextColor(c)
	return b
}

// SetFontSize sets an explicit font size in pixels for text rendering.
func (b *Button) SetFontSize(size float32) *Button {
	b.base.SetFontSize(size)
	return b
}

// SetItalic configures whether the button text uses the italic theme font.
func (b *Button) SetItalic(italic bool) *Button {
	b.base.SetItalic(italic)
	return b
}

// SetAlign configures horizontal text alignment within the content bounds.
func (b *Button) SetAlign(align core.TextAlign) *Button {
	b.base.SetAlign(align)
	return b
}

// Label is a non-interactive text display widget.
type Label struct {
	base
}

// NewLabel returns an enabled label with immutable name and kind.
func NewLabel(name string, bounds core.Rect, label string) *Label {
	b := newBase(name, core.WidgetLabel, bounds)
	b.text = label
	return &Label{base: b}
}

// NewStyledLabel returns an enabled label with custom font size, italic style, and alignment.
func NewStyledLabel(name string, bounds core.Rect, label string, fontSize float32, italic bool, align core.TextAlign) *Label {
	b := newBase(name, core.WidgetLabel, bounds)
	b.text = label
	b.fontSize = fontSize
	b.italic = italic
	b.align = align
	return &Label{base: b}
}

// OnClick attaches a click callback to the label.
func (l *Label) OnClick(fn func()) *Label {
	l.SetOnClick(fn)
	return l
}

// SetTooltip attaches a hover tooltip string directly to the label.
func (l *Label) SetTooltip(text string) *Label {
	l.base.SetTooltip(text)
	return l
}

// SetTextColor configures an explicit text color for the label.
func (l *Label) SetTextColor(c core.Color) *Label {
	l.base.SetTextColor(c)
	return l
}

// SetFontSize sets an explicit font size in pixels for text rendering.
func (l *Label) SetFontSize(size float32) *Label {
	l.base.SetFontSize(size)
	return l
}

// SetItalic configures whether the label text uses the italic theme font.
func (l *Label) SetItalic(italic bool) *Label {
	l.base.SetItalic(italic)
	return l
}

// SetAlign configures the horizontal text alignment within the content bounds.
func (l *Label) SetAlign(align core.TextAlign) *Label {
	l.base.SetAlign(align)
	return l
}

// Frame is a container/panel widget. Frames also host render-only row
// content: Attach records detached child widgets and links their layout
// nodes so list rows can draw a small widget tree without UI registration.
type Frame struct {
	base
	children []Widget
}

// NewFrame returns an enabled visual frame with immutable name and kind.
func NewFrame(name string, bounds core.Rect) *Frame {
	return &Frame{base: newBase(name, core.WidgetFrame, bounds)}
}

// SetTooltip attaches a hover tooltip string directly to the frame.
func (f *Frame) SetTooltip(text string) *Frame {
	f.base.SetTooltip(text)
	return f
}

// Attach records detached child widgets for render-only subtree drawing
// and links their layout nodes under the frame. Children already parented
// elsewhere, nil children, and the frame itself are skipped without error
// so UI-registered widgets can never be silently reparented. It returns the
// frame for chaining. Attached widgets are borrowed: the frame never copies
// them, and draw paths position them without taking UI ownership.
func (f *Frame) Attach(children ...Widget) *Frame {
	if f == nil || f.frame == nil {
		return f
	}
	for _, child := range children {
		if child == nil || child.Frame() == nil || child.Frame() == f.frame {
			continue
		}
		if err := f.frame.AddChild(child.Frame()); err != nil {
			continue
		}
		f.children = append(f.children, child)
	}
	return f
}

// Children returns a defensive copy of the attached row-content widgets.
func (f *Frame) Children() []Widget {
	if f == nil || len(f.children) == 0 {
		return nil
	}
	return append([]Widget(nil), f.children...)
}

// AppendChildren copies attached widgets into caller-owned storage for
// allocation-free draw paths.
func (f *Frame) AppendChildren(dst []Widget) []Widget {
	if f == nil {
		return dst
	}
	return append(dst, f.children...)
}
