package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// TabBar is a segmented tab selection widget.
type TabBar struct {
	base
	tabSelected int
	tabLabels   []string
	onTabSelect func(int)
}

// NewTabBar returns an enabled tab bar and copies labels safely.
func NewTabBar(name string, bounds core.Rect, labels []string, selected int) *TabBar {
	tb := &TabBar{
		base:        newBase(name, core.WidgetTabBar, bounds),
		tabSelected: -1,
	}
	tb.SetTabLabels(labels)
	tb.SetSelectedTab(selected)
	return tb
}

// TabCount returns the number of tab labels.
func (t *TabBar) TabCount() int {
	if t == nil {
		return 0
	}
	return len(t.tabLabels)
}

// TabLabels returns a safe copy of the tab labels.
func (t *TabBar) TabLabels() []string {
	if t == nil {
		return nil
	}
	return append([]string(nil), t.tabLabels...)
}

// SetTabLabels copies labels and keeps selection only when it remains valid.
func (t *TabBar) SetTabLabels(labels []string) bool {
	if t == nil {
		return false
	}
	changed := !equalStrings(t.tabLabels, labels)
	if changed {
		t.tabLabels = append(t.tabLabels[:0], labels...)
	}
	if t.tabSelected >= len(t.tabLabels) {
		t.tabSelected = -1
		changed = true
	}
	return changed
}

// SelectedTab returns the selected tab index, or -1 when unset.
func (t *TabBar) SelectedTab() int {
	if t == nil {
		return -1
	}
	return t.tabSelected
}

// SetSelectedTab selects a tab and reports whether the selection changed.
func (t *TabBar) SetSelectedTab(index int) bool {
	if t == nil || index < 0 || index >= len(t.tabLabels) || t.tabSelected == index {
		return false
	}
	t.tabSelected = index
	return true
}

// TabSelection returns the selected tab label, or false if unset.
func (t *TabBar) TabSelection() (string, bool) {
	if t == nil || t.tabSelected < 0 || t.tabSelected >= len(t.tabLabels) {
		return "", false
	}
	return t.tabLabels[t.tabSelected], true
}

// OnTabSelect attaches a tab selection callback directly to the tab bar.
func (t *TabBar) OnTabSelect(fn func(int)) *TabBar {
	if t != nil {
		t.onTabSelect = fn
	}
	return t
}

// OnTabSelectHandler returns the direct tab selection callback.
func (t *TabBar) OnTabSelectHandler() func(int) {
	if t == nil {
		return nil
	}
	return t.onTabSelect
}

// SetTooltip attaches a hover tooltip string directly to the tab bar.
func (t *TabBar) SetTooltip(text string) *TabBar {
	t.base.SetTooltip(text)
	return t
}

// SetTextColor configures an explicit text color for the tab bar labels.
func (t *TabBar) SetTextColor(color core.Color) *TabBar {
	t.base.SetTextColor(color)
	return t
}

// SetFontSize sets an explicit font size in pixels for the tab bar labels.
func (t *TabBar) SetFontSize(size float32) *TabBar {
	t.base.SetFontSize(size)
	return t
}
