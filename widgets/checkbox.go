package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// Checkbox is a toggleable box widget.
type Checkbox struct {
	base
	checked bool
}

// NewCheckbox returns an enabled checkbox with immutable name and kind.
func NewCheckbox(name string, bounds core.Rect, checked bool) *Checkbox {
	return &Checkbox{
		base:    newBase(name, core.WidgetCheckbox, bounds),
		checked: checked,
	}
}

// NewCheckboxWithLabel returns an enabled checkbox with a label and immutable name and kind.
func NewCheckboxWithLabel(name string, bounds core.Rect, label string, checked bool) *Checkbox {
	b := newBase(name, core.WidgetCheckbox, bounds)
	b.text = label
	return &Checkbox{
		base:    b,
		checked: checked,
	}
}

// Checked reports the checkbox toggle state.
func (c *Checkbox) Checked() bool {
	return c != nil && c.checked
}

// SetChecked changes the checkbox toggle state and reports whether it changed.
func (c *Checkbox) SetChecked(checked bool) bool {
	if c == nil || c.checked == checked {
		return false
	}
	c.checked = checked
	return true
}

// OnClick attaches a toggle callback directly to the checkbox.
func (c *Checkbox) OnClick(fn func()) *Checkbox {
	c.SetOnClick(fn)
	return c
}

// SetTooltip attaches a hover tooltip string directly to the checkbox.
func (c *Checkbox) SetTooltip(text string) *Checkbox {
	c.base.SetTooltip(text)
	return c
}

// SetTextColor configures an explicit text color for the checkbox label.
func (c *Checkbox) SetTextColor(color core.Color) *Checkbox {
	c.base.SetTextColor(color)
	return c
}

// SetFontSize sets an explicit font size in pixels for the checkbox label.
func (c *Checkbox) SetFontSize(size float32) *Checkbox {
	c.base.SetFontSize(size)
	return c
}

// SetItalic configures whether the checkbox label uses the italic theme font.
func (c *Checkbox) SetItalic(italic bool) *Checkbox {
	c.base.SetItalic(italic)
	return c
}

// SetAlign configures the horizontal text alignment for the checkbox label.
func (c *Checkbox) SetAlign(align core.TextAlign) *Checkbox {
	c.base.SetAlign(align)
	return c
}
