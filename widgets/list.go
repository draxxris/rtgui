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
//
// Row content follows two contracts: Label (plus the optional Icon) covers
// the common single-line case, while Content holds an arbitrary widget
// (usually a Frame built with Attach) for richer rows. A non-nil Content
// wins for rendering. Content widgets are borrowed by reference: the list
// copies item structure but never clones content, so callers must not reuse
// one widget across rows and should preserve widget identity across
// SetItems calls when the content is unchanged.
type ListItem struct {
	// ID is the stable row identity used by selection and expansion.
	ID string
	// Label is the plain row text, used when Content is nil.
	Label string
	// Content is the rich row widget; nil renders Label instead.
	Content Widget
	// Icon names a UI-whitelisted inline graphic; empty draws no icon.
	// It applies to Label rows only and is ignored when Content is set.
	Icon string
	// Height overrides the list row height for this row; values <= 0 or NaN
	// inherit the list height. Auto-measured heights are not supported.
	Height float32
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
	// Content is the borrowed row widget from the source item; nil renders
	// Label instead.
	Content Widget
	// Icon names a UI-whitelisted inline graphic; empty draws no icon.
	Icon string
	// Height is the resolved row height: the item override or the list
	// height when the item inherits.
	Height float32
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
	offsets    []float32
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

// Text returns the selected leaf content text, or the plain base text when
// unset. A non-nil row Content wins over Label so text and rendering agree.
// The value resolves from the item tree so it survives collapsing the
// selected leaf's parent category.
func (l *List) Text() string {
	if l == nil {
		return ""
	}
	if l.selected != "" {
		if item := findListItem(l.items, l.selected); item != nil {
			if item.Content != nil {
				return item.Content.Text()
			}
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
	if equalListItems(l.items, items, l.RowHeight()) {
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

// SelectedIndex returns the visible index of the selected leaf. Indices are
// ephemeral: expansion, collapse, and data refreshes renumber them, so
// prefer Selected IDs for stable identity and use the index only for
// viewport concerns like scroll-into-view.
func (l *List) SelectedIndex() (int, bool) {
	if l == nil || l.selected == "" {
		return -1, false
	}
	index := l.IndexOfRow(l.selected)
	if index < 0 {
		return -1, false
	}
	return index, true
}

// TotalHeight returns the stacked height of all visible rows.
func (l *List) TotalHeight() float32 {
	if l == nil || len(l.offsets) == 0 {
		return 0
	}
	return l.offsets[len(l.offsets)-1]
}

// RowHeightAt returns the resolved height of one visible row: the item
// override or the list height when the item inherits.
func (l *List) RowHeightAt(index int) float32 {
	if l == nil || index < 0 || index >= len(l.rows) {
		return DefaultListRowHeight
	}
	if l.rows[index].Height > 0 {
		return l.rows[index].Height
	}
	return l.RowHeight()
}

// RowRect returns one variable-height row rect inside content, translated
// by the scroll offset. Rows stack from content.Y using the cached offsets.
func (l *List) RowRect(content core.Rect, index int) (core.Rect, bool) {
	if l == nil || index < 0 || index >= len(l.rows) || content.W <= 0 {
		return core.Rect{}, false
	}
	if len(l.offsets) != len(l.rows)+1 {
		return core.Rect{}, false
	}
	height := l.offsets[index+1] - l.offsets[index]
	if height <= 0 {
		return core.Rect{}, false
	}
	return core.Rect{X: content.X, Y: content.Y + l.offsets[index] - l.scrollY, W: content.W, H: height}, true
}

// RowAt returns the visible row index under pos, or -1 outside content.
// It inverts RowRect exactly; scrolled-off rows never hit.
func (l *List) RowAt(content core.Rect, pos core.Vec2) int {
	if l == nil || len(l.rows) == 0 || len(l.offsets) != len(l.rows)+1 {
		return -1
	}
	if content.W <= 0 || content.H <= 0 {
		return -1
	}
	if pos.X < content.X || pos.X > content.X+content.W || pos.Y < content.Y || pos.Y > content.Y+content.H {
		return -1
	}
	y := pos.Y - content.Y + l.scrollY
	lo, hi := 0, len(l.rows)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if y < l.offsets[mid] {
			hi = mid - 1
		} else if y >= l.offsets[mid+1] {
			lo = mid + 1
		} else {
			return mid
		}
	}
	return -1
}

// VisibleRange returns the first and last row indices intersecting a
// viewport of contentH logical pixels, clamped to the visible rows.
// An empty range reports last < first.
func (l *List) VisibleRange(contentH float32) (int, int) {
	if l == nil || len(l.rows) == 0 || contentH <= 0 {
		return 0, -1
	}
	if len(l.offsets) != len(l.rows)+1 {
		return 0, -1
	}
	first := l.RowAt(core.Rect{W: 1, H: contentH}, core.Vec2{X: 0, Y: 0})
	last := l.RowAt(core.Rect{W: 1, H: contentH}, core.Vec2{X: 0, Y: contentH - 1})
	if first < 0 {
		first = 0
	}
	if last < 0 {
		last = len(l.rows) - 1
		if l.offsets[last] >= l.scrollY+contentH {
			last = first - 1
		}
	}
	return first, last
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
	max := listMaxScrollTotal(l.TotalHeight(), viewportH)
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

// refresh rebuilds the visible rows, the stacked height offsets, and the
// scroll bounds.
func (l *List) refresh() {
	if l == nil {
		return
	}
	l.rows = flattenListItems(l.items, l.rows[:0], 0, l.RowHeight())
	l.offsets = rebuildListOffsets(l.rows, l.offsets[:0])
	bounds := l.Bounds()
	l.maxScrollY = listMaxScrollTotal(l.TotalHeight(), bounds.H)
	l.scrollY = clampScroll(l.scrollY, l.maxScrollY)
}

// listMaxScrollTotal derives the scroll limit from stacked content height
// and viewport height.
func listMaxScrollTotal(total, viewportH float32) float32 {
	if total <= 0 {
		return 0
	}
	if math.IsNaN(float64(viewportH)) || viewportH < 0 {
		viewportH = 0
	}
	if total <= viewportH {
		return 0
	}
	return total - viewportH
}

// rebuildListOffsets stacks resolved row heights into a prefix sum reused
// across refreshes: offsets[i] is the top of row i and offsets[len] is the
// total height.
func rebuildListOffsets(rows []ListRow, dst []float32) []float32 {
	if cap(dst) < len(rows)+1 {
		dst = make([]float32, len(rows)+1)
	} else {
		dst = dst[:len(rows)+1]
	}
	if len(rows) == 0 {
		return dst[:0]
	}
	for i, row := range rows {
		dst[i+1] = dst[i] + row.Height
	}
	return dst
}

// resolveListHeight maps an item height override to a concrete row height:
// positive finite values win, anything else inherits the list height.
func resolveListHeight(override, inherit float32) float32 {
	if math.IsNaN(float64(override)) || math.IsInf(float64(override), 0) || override <= 0 {
		return inherit
	}
	return override
}

// flattenListItems appends visible rows in order. Trees stay shallow
// (game-authored categories), so recursion beats an explicit stack.
// Row heights resolve against defaultH; Content widgets stay borrowed.
func flattenListItems(items []ListItem, dst []ListRow, depth int, defaultH float32) []ListRow {
	for _, item := range items {
		dst = append(dst, ListRow{
			ID:          item.ID,
			Label:       item.Label,
			Content:     item.Content,
			Icon:        item.Icon,
			Height:      resolveListHeight(item.Height, defaultH),
			Depth:       depth,
			HasChildren: len(item.Children) > 0,
			Expanded:    item.Expanded,
		})
		if len(item.Children) > 0 && item.Expanded {
			dst = flattenListItems(item.Children, dst, depth+1, defaultH)
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

// copyListItems copies the item structure. Scalar fields and child slices
// are duplicated so structural mutations stay isolated; Content widgets are
// borrowed by reference and shared with the caller by design.
func copyListItems(items []ListItem) []ListItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]ListItem, len(items))
	for i, item := range items {
		out[i].ID = item.ID
		out[i].Label = item.Label
		out[i].Content = item.Content
		out[i].Icon = item.Icon
		out[i].Height = item.Height
		out[i].Expanded = item.Expanded
		out[i].Children = copyListItems(item.Children)
	}
	return out
}

// equalListItems compares item trees without allocating. Heights compare
// resolved so every inherit spelling (zero, negative, NaN) equals itself.
func equalListItems(left, right []ListItem, defaultH float32) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].ID != right[i].ID || left[i].Label != right[i].Label ||
			left[i].Content != right[i].Content || left[i].Icon != right[i].Icon ||
			left[i].Expanded != right[i].Expanded ||
			resolveListHeight(left[i].Height, defaultH) != resolveListHeight(right[i].Height, defaultH) {
			return false
		}
		if !equalListItems(left[i].Children, right[i].Children, defaultH) {
			return false
		}
	}
	return true
}
