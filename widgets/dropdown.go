package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// Dropdown is a selection widget with an openable popup list.
type Dropdown struct {
	base
	dropdownIndex int
	dropdownItems []string
}

// NewDropdown returns an enabled dropdown and copies items safely.
func NewDropdown(name string, bounds core.Rect, items []string, index int) *Dropdown {
	d := &Dropdown{
		base:          newBase(name, core.WidgetDropdown, bounds),
		dropdownIndex: -1,
	}
	d.SetDropdownItems(items)
	d.SetDropdownIndex(index)
	return d
}

// DropdownIndex returns the selected dropdown item index, or -1 when unset.
func (d *Dropdown) DropdownIndex() int {
	if d == nil {
		return -1
	}
	return d.dropdownIndex
}

// SetDropdownIndex selects an item and reports whether the selection changed.
func (d *Dropdown) SetDropdownIndex(index int) bool {
	if d == nil || index < 0 || index >= len(d.dropdownItems) || d.dropdownIndex == index {
		return false
	}
	d.dropdownIndex = index
	return true
}

// DropdownSelection returns the selected item text without exposing internal slice.
func (d *Dropdown) DropdownSelection() (string, bool) {
	if d == nil || d.dropdownIndex < 0 || d.dropdownIndex >= len(d.dropdownItems) {
		return "", false
	}
	return d.dropdownItems[d.dropdownIndex], true
}

// Text returns the selected item text or empty string.
func (d *Dropdown) Text() string {
	if s, ok := d.DropdownSelection(); ok {
		return s
	}
	return d.base.Text()
}

// DropdownItems returns a safe copy of the dropdown item list.
func (d *Dropdown) DropdownItems() []string {
	if d == nil {
		return nil
	}
	return append([]string(nil), d.dropdownItems...)
}

// SetDropdownItems copies items and preserves selection if still valid.
func (d *Dropdown) SetDropdownItems(items []string) bool {
	if d == nil {
		return false
	}
	changed := !equalStrings(d.dropdownItems, items)
	if changed {
		d.dropdownItems = append(d.dropdownItems[:0], items...)
	}
	if d.dropdownIndex >= len(d.dropdownItems) {
		d.dropdownIndex = -1
		changed = true
	}
	return changed
}

// DropdownPopupBounds returns the popup list rectangle below the control.
func (d *Dropdown) DropdownPopupBounds() core.Rect {
	if d == nil {
		return core.Rect{}
	}
	bounds := d.Bounds()
	return core.Rect{
		X: bounds.X,
		Y: bounds.Y + bounds.H + 4,
		W: bounds.W,
		H: float32(len(d.dropdownItems) * 36),
	}
}

// DropdownItemCount returns the count of dropdown items.
func (d *Dropdown) DropdownItemCount() int {
	if d == nil {
		return 0
	}
	return len(d.dropdownItems)
}

// SetTooltip attaches a hover tooltip string directly to the dropdown.
func (d *Dropdown) SetTooltip(text string) *Dropdown {
	d.base.SetTooltip(text)
	return d
}

// SetTextColor configures an explicit text color for the dropdown.
func (d *Dropdown) SetTextColor(color core.Color) *Dropdown {
	d.base.SetTextColor(color)
	return d
}

// SetFontSize sets an explicit font size in pixels for the dropdown.
func (d *Dropdown) SetFontSize(size float32) *Dropdown {
	d.base.SetFontSize(size)
	return d
}

// equalStrings compares string slices without allocating.
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
