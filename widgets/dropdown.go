package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// Dropdown is a selection widget with an openable popup list.
// Per-row rich runs are the single rich source: the closed control shows
// the selected row's runs when set, else the plain selection. Inherited
// base rich runs are inert for dropdowns. Links in rows render in color
// without activation.
type Dropdown struct {
	base
	dropdownIndex     int
	dropdownItems     []string
	dropdownRichItems [][]core.RichSegment
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

// AppendDropdownItems copies items into reusable caller-owned storage.
func (d *Dropdown) AppendDropdownItems(dst []string) []string { return append(dst, d.dropdownItems...) }

// SetDropdownItems copies items, preserves selection if still valid, and
// clears any per-row rich overlay.
func (d *Dropdown) SetDropdownItems(items []string) bool {
	if d == nil {
		return false
	}
	changed := !equalStrings(d.dropdownItems, items)
	if changed {
		clear(d.dropdownItems)
		d.dropdownItems = append(d.dropdownItems[:0], items...)
	}
	if len(d.dropdownRichItems) > 0 {
		clearRichDropdownItems(d.dropdownRichItems)
		d.dropdownRichItems = d.dropdownRichItems[:0]
		changed = true
	}
	// Base rich runs are inert for dropdowns (rows are the single source),
	// so item replacement resets them too rather than stranding state.
	if d.ClearRichText() {
		changed = true
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

// HasRichDropdownItems reports whether any popup row carries rich runs.
func (d *Dropdown) HasRichDropdownItems() bool {
	return d != nil && len(d.dropdownRichItems) > 0
}

// RichDropdownRows returns the internal per-row rich overlay for drawing.
// The result aliases widget storage and must be treated as read-only until
// the next mutation; rows without runs are nil.
func (d *Dropdown) RichDropdownRows() [][]core.RichSegment {
	if d == nil {
		return nil
	}
	return d.dropdownRichItems
}

// RichDropdownItem returns a safe copy of one row's rich runs.
func (d *Dropdown) RichDropdownItem(index int) ([]core.RichSegment, bool) {
	if d == nil || index < 0 || index >= len(d.dropdownRichItems) {
		return nil, false
	}
	if len(d.dropdownRichItems[index]) == 0 {
		return nil, false
	}
	return append([]core.RichSegment(nil), d.dropdownRichItems[index]...), true
}

// SetRichDropdownItem replaces one row's rich runs and reports a change.
// The plain item text is left untouched as fallback.
func (d *Dropdown) SetRichDropdownItem(index int, segments []core.RichSegment) bool {
	if d == nil || index < 0 || index >= len(d.dropdownItems) {
		return false
	}
	for len(d.dropdownRichItems) < len(d.dropdownItems) {
		d.dropdownRichItems = append(d.dropdownRichItems, nil)
	}
	if core.EqualRichSegments(d.dropdownRichItems[index], segments) {
		return false
	}
	d.dropdownRichItems[index] = append(d.dropdownRichItems[index][:0], segments...)
	return true
}

// ClearRichDropdownItems drops every per-row rich overlay.
func (d *Dropdown) ClearRichDropdownItems() bool {
	if d == nil || len(d.dropdownRichItems) == 0 {
		return false
	}
	clearRichDropdownItems(d.dropdownRichItems)
	d.dropdownRichItems = d.dropdownRichItems[:0]
	return true
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

// clearRichDropdownItems zeroes nested row storage before reuse.
func clearRichDropdownItems(rows [][]core.RichSegment) {
	for i := range rows {
		clear(rows[i])
		rows[i] = nil
	}
}
