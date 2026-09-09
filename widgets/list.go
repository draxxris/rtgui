package widgets

import (
	"math"

	"github.com/draxxris/rtgui/core"
)

const (
	// DefaultListRowHeight is the initial row height in logical pixels.
	DefaultListRowHeight = float32(28)
	// DefaultListIndent is the initial per-depth indent in logical pixels.
	DefaultListIndent = float32(16)
	// ListChevronWidth reserves the right-edge expander zone on category rows.
	ListChevronWidth = float32(28)
	// ListIconSlotWidth reserves the icon box on rows carrying an icon name.
	ListIconSlotWidth = float32(20)
)

// ListItem is one node in a collapsible list. An item with children is a
// toggle-only category; an item without children is a selectable leaf.
// IDs must be unique across the whole tree and stable across updates so
// selection and expanded state survive data refreshes by identity.
type ListItem struct {
	// ID is the stable row identity used by selection and expansion.
	ID string
	// Label is the plain row text.
	Label string
	// Icon names a UI-whitelisted inline graphic; empty draws no icon.
	Icon string
	// Expanded controls whether a category shows its children.
	Expanded bool
	// Children holds nested rows; empty means a selectable leaf.
	Children []ListItem
}

// ListRow is one flattened visible row. Depth 0 is a top-level entry;
// deeper rows are shown only while every ancestor is expanded.
type ListRow struct {
	// ID is the stable row identity from the source item.
	ID string
	// Label is the plain row text from the source item.
	Label string
	// Icon names a UI-whitelisted inline graphic; empty draws no icon.
	Icon string
	// Depth is the nesting level, used for indentation.
	Depth int
	// HasChildren reports a toggle-only category row.
	HasChildren bool
	// Expanded reports whether a category currently shows its children.
	Expanded bool
}

// List is a collapsible single-selection list widget. Category rows toggle
// expansion; leaf rows select. The widget owns domain data only; hover,
// press, and scroll-thumb state stay in ui.UI on the owning goroutine.
type List struct {
	base
	items      []ListItem
	rows       []ListRow
	selected   string
	rowHeight  float32
	indent     float32
	scrollY    float32
	maxScrollY float32
}

// NewList returns an enabled list with default row metrics.
func NewList(name string, bounds core.Rect) *List {
	l := &List{
		base:      newBase(name, core.WidgetList, bounds),
		rowHeight: DefaultListRowHeight,
		indent:    DefaultListIndent,
	}
	l.refresh()
	return l
}

// RowHeight returns the fixed row height in logical pixels.
func (l *List) RowHeight() float32 {
	if l == nil || l.rowHeight <= 0 {
		return DefaultListRowHeight
	}
	return l.rowHeight
}

// SetRowHeight replaces the row height and reports the list for chaining.
// Non-positive values are ignored.
func (l *List) SetRowHeight(height float32) *List {
	if l == nil || height <= 0 || math.IsNaN(float64(height)) || l.rowHeight == height {
		return l
	}
	l.rowHeight = height
	l.refresh()
	return l
}

// Indent returns the per-depth indent in logical pixels.
func (l *List) Indent() float32 {
	if l == nil || l.indent < 0 || math.IsNaN(float64(l.indent)) {
		return DefaultListIndent
	}
	return l.indent
}

// SetIndent replaces the per-depth indent and reports the list for chaining.
// Negative values are ignored; zero removes indentation.
func (l *List) SetIndent(width float32) *List {
	if l == nil || width < 0 || math.IsNaN(float64(width)) || l.indent == width {
		return l
	}
	l.indent = width
	return l
}

// Text returns the selected leaf label, or the plain base text when unset.
// The label resolves from the item tree so it survives collapsing the
// selected leaf's parent category.
func (l *List) Text() string {
	if l == nil {
		return ""
	}
	if l.selected != "" {
		if item := findListItem(l.items, l.selected); item != nil {
			return item.Label
		}
	}
	return l.base.Text()
}

// Items returns a defensive deep copy of the item tree.
func (l *List) Items() []ListItem {
	if l == nil || len(l.items) == 0 {
		return nil
	}
	return copyListItems(l.items)
}

// SetItems replaces the item tree, preserving the selection only when the
// selected ID remains a leaf. Expanded flags come from the incoming tree.
// It reports whether visible rows or the selection changed.
func (l *List) SetItems(items []ListItem) bool {
	if l == nil {
		return false
	}
	if equalListItems(l.items, items) {
		return false
	}
	selected := l.selected
	l.items = copyListItems(items)
	l.refresh()
	if selected != "" && !l.isLeaf(selected) {
		l.selected = ""
	}
	return true
}

// VisibleRows returns a copy of the flattened visible rows.
func (l *List) VisibleRows() []ListRow {
	if l == nil || len(l.rows) == 0 {
		return nil
	}
	return append([]ListRow(nil), l.rows...)
}

// AppendVisibleRows copies visible rows into reusable caller-owned storage.
func (l *List) AppendVisibleRows(dst []ListRow) []ListRow {
	if l == nil {
		return dst
	}
	return append(dst, l.rows...)
}

// VisibleRowCount returns the flattened visible row count.
func (l *List) VisibleRowCount() int {
	if l == nil {
		return 0
	}
	return len(l.rows)
}

// VisibleRowAt returns a copy of the indexed visible row.
func (l *List) VisibleRowAt(index int) (ListRow, bool) {
	if l == nil || index < 0 || index >= len(l.rows) {
		return ListRow{}, false
	}
	return l.rows[index], true
}

// IndexOfRow resolves a visible row index by stable ID, or -1 when hidden.
func (l *List) IndexOfRow(id string) int {
	if l == nil {
		return -1
	}
	for i, row := range l.rows {
		if row.ID == id {
			return i
		}
	}
	return -1
}

// Selected returns the selected leaf ID, or false when unset.
func (l *List) Selected() (string, bool) {
	if l == nil || l.selected == "" {
		return "", false
	}
	return l.selected, true
}

// Select chooses a leaf row and reports whether the selection changed.
// Categories, unknown IDs, and empty IDs never select; use ClearSelection.
func (l *List) Select(id string) bool {
	if l == nil || id == "" || l.selected == id || !l.isLeaf(id) {
		return false
	}
	l.selected = id
	return true
}

// ClearSelection drops the selection and reports whether one was held.
func (l *List) ClearSelection() bool {
	if l == nil || l.selected == "" {
		return false
	}
	l.selected = ""
	return true
}

// IsExpanded reports whether a category currently shows its children.
func (l *List) IsExpanded(id string) bool {
	if l == nil {
		return false
	}
	item := findListItem(l.items, id)
	return item != nil && len(item.Children) > 0 && item.Expanded
}

// SetExpanded expands or collapses a category and reports a change.
// Leaves and unknown IDs report false.
func (l *List) SetExpanded(id string, expanded bool) bool {
	if l == nil {
		return false
	}
	item := findListItem(l.items, id)
	if item == nil || len(item.Children) == 0 || item.Expanded == expanded {
		return false
	}
	item.Expanded = expanded
	l.refresh()
	return true
}

// Toggle flips a category and reports a change. Leaves report false.
func (l *List) Toggle(id string) bool {
	if l == nil {
		return false
	}
	item := findListItem(l.items, id)
	if item == nil || len(item.Children) == 0 {
		return false
	}
	item.Expanded = !item.Expanded
	l.refresh()
	return true
}

// ExpandAll expands every category and reports whether anything changed.
func (l *List) ExpandAll() bool {
	if l == nil {
		return false
	}
	if !setAllListExpanded(l.items, true) {
		return false
	}
	l.refresh()
	return true
}

// CollapseAll collapses every category and reports whether anything changed.
func (l *List) CollapseAll() bool {
	if l == nil {
		return false
	}
	if !setAllListExpanded(l.items, false) {
		return false
	}
	l.refresh()
	return true
}

// ScrollOffset returns the vertical scroll offset in logical pixels.
func (l *List) ScrollOffset() float32 {
	if l == nil {
		return 0
	}
	return l.scrollY
}

// MaxScroll returns the maximum vertical scroll offset.
func (l *List) MaxScroll() float32 {
	if l == nil {
		return 0
	}
	return l.maxScrollY
}

// SetScrollOffset replaces the scroll offset clamped to [0, MaxScroll].
// It reports whether the offset changed.
func (l *List) SetScrollOffset(offset float32) bool {
	if l == nil {
		return false
	}
	offset = clampScroll(offset, l.maxScrollY)
	if l.scrollY == offset {
		return false
	}
	l.scrollY = offset
	return true
}

// ScrollBy advances the scroll offset and reports whether it changed.
func (l *List) ScrollBy(dy float32) bool {
	if l == nil || dy == 0 || math.IsNaN(float64(dy)) {
		return false
	}
	return l.SetScrollOffset(l.scrollY + dy)
}

// SetMaxScroll configures an explicit scroll limit for UI reconciliation.
// It clamps the current offset and reports the list for chaining.
func (l *List) SetMaxScroll(max float32) *List {
	if l == nil {
		return l
	}
	if math.IsNaN(float64(max)) || max < 0 {
		max = 0
	}
	l.maxScrollY = max
	l.SetScrollOffset(l.scrollY)
	return l
}

// EnsureScrollBounds reconciles the derived scroll limit against a viewport
// height (usually the theme content height) and clamps the offset.
// Widgets without skin knowledge use bounds height; the UI calls this with
// the true content height before hit testing, scrolling, and drawing.
func (l *List) EnsureScrollBounds(viewportH float32) bool {
	if l == nil {
		return false
	}
	changed := false
	max := listMaxScroll(len(l.rows), l.RowHeight(), viewportH)
	if max != l.maxScrollY {
		l.maxScrollY = max
		changed = true
	}
	if l.SetScrollOffset(l.scrollY) {
		changed = true
	}
	return changed
}

// OnSelect attaches a leaf-selection callback directly to the list.
func (l *List) OnSelect(fn func(string)) *List {
	if l != nil {
		l.callbacks.ListSelect = fn
	}
	return l
}

// OnSelectHandler returns the direct leaf-selection callback.
func (l *List) OnSelectHandler() func(string) {
	if l == nil {
		return nil
	}
	return l.callbacks.ListSelect
}

// OnToggle attaches a category toggle callback directly to the list.
func (l *List) OnToggle(fn func(string, bool)) *List {
	if l != nil {
		l.callbacks.ListToggle = fn
	}
	return l
}

// OnToggleHandler returns the direct category toggle callback.
func (l *List) OnToggleHandler() func(string, bool) {
	if l == nil {
		return nil
	}
	return l.callbacks.ListToggle
}

// SetTooltip attaches a hover tooltip string directly to the list.
func (l *List) SetTooltip(text string) *List {
	l.base.SetTooltip(text)
	return l
}

// SetTextColor configures an explicit text color for list rows.
func (l *List) SetTextColor(color core.Color) *List {
	l.base.SetTextColor(color)
	return l
}

// SetFontSize sets an explicit font size in pixels for list rows.
func (l *List) SetFontSize(size float32) *List {
	l.base.SetFontSize(size)
	return l
}

// SetItalic configures whether list rows use the italic theme font.
func (l *List) SetItalic(italic bool) *List {
	l.base.SetItalic(italic)
	return l
}

// SetAlign configures the horizontal text alignment within rows.
func (l *List) SetAlign(align core.TextAlign) *List {
	l.base.SetAlign(align)
	return l
}

// isLeaf reports whether id names a selectable leaf row.
func (l *List) isLeaf(id string) bool {
	item := findListItem(l.items, id)
	return item != nil && len(item.Children) == 0
}

// refresh rebuilds the visible rows and reconciles scroll bounds.
func (l *List) refresh() {
	if l == nil {
		return
	}
	l.rows = flattenListItems(l.items, l.rows[:0], 0)
	bounds := l.Bounds()
	l.maxScrollY = listMaxScroll(len(l.rows), l.RowHeight(), bounds.H)
	l.scrollY = clampScroll(l.scrollY, l.maxScrollY)
}

// listMaxScroll derives the scroll limit from row count and viewport height.
func listMaxScroll(count int, rowH, viewportH float32) float32 {
	if count <= 0 || rowH <= 0 {
		return 0
	}
	if math.IsNaN(float64(viewportH)) || viewportH < 0 {
		viewportH = 0
	}
	total := float32(count) * rowH
	if total <= viewportH {
		return 0
	}
	return total - viewportH
}

// flattenListItems appends visible rows in order. Trees stay shallow
// (game-authored categories), so recursion beats an explicit stack.
func flattenListItems(items []ListItem, dst []ListRow, depth int) []ListRow {
	for _, item := range items {
		dst = append(dst, ListRow{
			ID:          item.ID,
			Label:       item.Label,
			Icon:        item.Icon,
			Depth:       depth,
			HasChildren: len(item.Children) > 0,
			Expanded:    item.Expanded,
		})
		if len(item.Children) > 0 && item.Expanded {
			dst = flattenListItems(item.Children, dst, depth+1)
		}
	}
	return dst
}

// findListItem resolves a mutable item by stable ID.
func findListItem(items []ListItem, id string) *ListItem {
	if id == "" {
		return nil
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
		if found := findListItem(items[i].Children, id); found != nil {
			return found
		}
	}
	return nil
}

// setAllListExpanded assigns every category flag and reports a change.
func setAllListExpanded(items []ListItem, expanded bool) bool {
	changed := false
	for i := range items {
		if len(items[i].Children) > 0 && items[i].Expanded != expanded {
			items[i].Expanded = expanded
			changed = true
		}
		if setAllListExpanded(items[i].Children, expanded) {
			changed = true
		}
	}
	return changed
}

// copyListItems deep-copies an item tree.
func copyListItems(items []ListItem) []ListItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]ListItem, len(items))
	for i, item := range items {
		out[i].ID = item.ID
		out[i].Label = item.Label
		out[i].Icon = item.Icon
		out[i].Expanded = item.Expanded
		out[i].Children = copyListItems(item.Children)
	}
	return out
}

// equalListItems compares item trees without allocating.
func equalListItems(left, right []ListItem) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].ID != right[i].ID || left[i].Label != right[i].Label ||
			left[i].Icon != right[i].Icon || left[i].Expanded != right[i].Expanded {
			return false
		}
		if !equalListItems(left[i].Children, right[i].Children) {
			return false
		}
	}
	return true
}
