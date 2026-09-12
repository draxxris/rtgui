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

// OnListSelect replaces the registered list's leaf-selection callback.
func (u *UI) OnListSelect(name string, fn func(string)) {
	if c := u.callback(name); c != nil {
		c.ListSelect = fn
	}
}

// OnListToggle replaces the registered list's category toggle callback.
func (u *UI) OnListToggle(name string, fn func(string, bool)) {
	if c := u.callback(name); c != nil {
		c.ListToggle = fn
	}
}

// OnTableSelect replaces the registered table's stable-row selection callback.
func (u *UI) OnTableSelect(name string, fn func(string)) {
	if c := u.callback(name); c != nil {
		c.TableSelect = fn
	}
}

// OnTableSort replaces the registered table's sorted-column callback.
func (u *UI) OnTableSort(name string, fn func(string, core.SortDir)) {
	if c := u.callback(name); c != nil {
		c.TableSort = fn
	}
}

// OnTableActivate replaces the registered table's semantic activation callback.
func (u *UI) OnTableActivate(name string, fn func(string)) {
	if c := u.callback(name); c != nil {
		c.TableActivate = fn
	}
}

// OnTableCellTooltip replaces the registered table cell tooltip provider.
func (u *UI) OnTableCellTooltip(name string, fn func(string, string) string) {
	if c := u.callback(name); c != nil {
		c.TableCellTooltip = fn
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

// OnChatLink replaces a registered chat log's link callback. Nil clears it.
// It shares the single link-click slot with OnLinkClick.
func (u *UI) OnChatLink(name string, fn func(core.Link)) {
	u.OnLinkClick(name, fn)
}

// OnChatLinkTooltipRequested replaces a registered chat log's link tooltip
// provider. It shares the single link-tooltip slot with OnLinkTooltipRequested.
func (u *UI) OnChatLinkTooltipRequested(name string, fn func(core.Link) string) {
	u.OnLinkTooltipRequested(name, fn)
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

// fireOnListSelect invokes the current list selection callback once.
func (u *UI) fireOnListSelect(name, id string) {
	if c := u.callback(name); c != nil && c.ListSelect != nil {
		c.ListSelect(id)
	}
}

// fireOnListToggle invokes the current list toggle callback once.
func (u *UI) fireOnListToggle(name, id string, expanded bool) {
	if c := u.callback(name); c != nil && c.ListToggle != nil {
		c.ListToggle(id, expanded)
	}
}

// fireOnTableSelect invokes the current table selection callback once.
func (u *UI) fireOnTableSelect(name, id string) {
	if c := u.callback(name); c != nil && c.TableSelect != nil {
		c.TableSelect(id)
	}
}

// fireOnTableSort invokes the current table sort callback once.
func (u *UI) fireOnTableSort(name, columnID string, dir core.SortDir) {
	if c := u.callback(name); c != nil && c.TableSort != nil {
		c.TableSort(columnID, dir)
	}
}

// fireOnTableActivate invokes the current semantic table activation callback.
func (u *UI) fireOnTableActivate(name, id string) {
	if c := u.callback(name); c != nil && c.TableActivate != nil {
		c.TableActivate(id)
	}
}

// fireOnLinkClick invokes the current link callback once.
func (u *UI) fireOnLinkClick(name string, link core.Link) {
	if c := u.callback(name); c != nil && c.LinkClick != nil {
		c.LinkClick(link)
	}
}
