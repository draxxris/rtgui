package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// callback returns the sole callback registry. Register widgets before callbacks.
func (u *UI) callback(name string) *widgets.Callbacks {
	if w := u.Lookup(name); w != nil {
		return w.Callbacks()
	}
	return nil
}

// OnClick replaces the registered widget's activation callback. Nil clears it.
func (u *UI) OnClick(name string, fn func()) {
	if c := u.callback(name); c != nil {
		c.Click = fn
	}
}

// OnChange replaces the registered widget's value callback. Nil clears it.
func (u *UI) OnChange(name string, fn func(float32)) {
	if c := u.callback(name); c != nil {
		c.Change = fn
	}
}

// OnText replaces the registered widget's text callback. Nil clears it.
func (u *UI) OnText(name string, fn func(string)) {
	if c := u.callback(name); c != nil {
		c.Text = fn
	}
}

// OnTabSelect replaces the registered widget's selection callback.
func (u *UI) OnTabSelect(name string, fn func(int)) {
	if c := u.callback(name); c != nil {
		c.TabSelect = fn
	}
}

// OnLinkClick replaces the registered widget's link callback.
func (u *UI) OnLinkClick(name string, fn func(core.Link)) {
	if c := u.callback(name); c != nil {
		c.LinkClick = fn
	}
}

// OnLinkTooltipRequested replaces the registered widget's link tooltip provider.
func (u *UI) OnLinkTooltipRequested(name string, fn func(core.Link) string) {
	if c := u.callback(name); c != nil {
		c.LinkTooltip = fn
	}
}

// fireOnClick invokes the current activation callback once.
func (u *UI) fireOnClick(name string) {
	if c := u.callback(name); c != nil && c.Click != nil {
		c.Click()
	}
}

// fireOnChange invokes the current value callback once.
func (u *UI) fireOnChange(name string, value float32) {
	if c := u.callback(name); c != nil && c.Change != nil {
		c.Change(value)
	}
}

// fireOnText invokes the current text callback once.
func (u *UI) fireOnText(name, value string) {
	if c := u.callback(name); c != nil && c.Text != nil {
		c.Text(value)
	}
}

// fireOnTabSelect invokes the current selection callback once.
func (u *UI) fireOnTabSelect(name string, index int) {
	if c := u.callback(name); c != nil && c.TabSelect != nil {
		c.TabSelect(index)
	}
}

// fireOnLinkClick invokes the current link callback once.
func (u *UI) fireOnLinkClick(name string, link core.Link) {
	if c := u.callback(name); c != nil && c.LinkClick != nil {
		c.LinkClick(link)
	}
}
