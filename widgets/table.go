package widgets

import (
	"math"

	"github.com/draxxris/rtgui/core"
)

const (
	// DefaultTableRowHeight is the initial height of every table data row.
	DefaultTableRowHeight = float32(28)
	// DefaultTableHeaderHeight is the initial height of the fixed table header.
	DefaultTableHeaderHeight = float32(32)
)

// SortDir aliases the core sort direction for table APIs and custom sort
// helpers. Callers should use core.None, core.SortAsc, or core.SortDesc.
type SortDir = core.SortDir

// TableColumn describes one stable, fixed-position column in a Table.
// Width is a relative weight; MinWidth and MaxWidth constrain its resolved
// pixel width. A zero MaxWidth means that no upper bound was requested.
type TableColumn struct {
	// ID is the non-empty stable identity used by cells and sorting.
	ID string
	// Title is the displayed header text; empty permits an icon-only header.
	Title string
	// Width is the relative layout weight before pixel clamps are applied.
	Width float32
	// MinWidth is the resolved pixel-width lower bound.
	MinWidth float32
	// MaxWidth is the resolved pixel-width upper bound; zero means unlimited.
	MaxWidth float32
	// Align is the horizontal alignment used for non-numeric cell text.
	Align core.TextAlign
	// Sortable permits header and programmatic sorting for this column.
	Sortable bool
	// Numeric selects SortValue ordering and defaults cell presentation right.
	Numeric bool
}

// TableCell is one read-only display cell in a table row.
// Icon names are checked against the UI inline-icon whitelist at draw time.
type TableCell struct {
	// Text is the displayed plain cell text and text-sort fallback.
	Text string
	// Icon names a UI-whitelisted inline graphic; empty draws no icon.
	Icon string
	// Color is the optional explicit cell tint.
	Color core.Color
	// HasColor selects Color over the table or skin text color.
	HasColor bool
	// Invalid marks a game-defined unusable or unaffordable value.
	Invalid bool
	// SortValue is the numeric value used when HasSortValue is true.
	SortValue float64
	// HasSortValue reports whether SortValue is present for numeric sorting.
	HasSortValue bool
}

// TableRow is one stable data row. Cells follow the current column order;
// short input rows are padded with zero cells when stored.
type TableRow struct {
	// ID is the non-empty stable identity used by selection and updates.
	ID string
	// Cells contains values indexed by the Table's current columns.
	Cells []TableCell
}

// Table is a generic read-only, single-selection, sortable data grid. It owns
// insertion-order rows and a separate sorted view, while UI-owned hover,
// pressed, focus, and scrollbar gesture state remains outside the widget.
type Table struct {
	base
	columns      []TableColumn
	rows         []TableRow
	order        []int
	sortScratch  []int
	selected     string
	sortColumn   string
	sortDir      core.SortDir
	rowHeight    float32
	headerHeight float32
	scrollY      float32
	maxScrollY   float32
	viewportH    float32
	revision     uint64
}

var _ Widget = (*Table)(nil)

// NewTable returns an enabled table with empty data and default metrics.
// Its initial viewport is the constructor bounds height; UI layout can replace
// that value through EnsureScrollBounds after skin insets and track exclusion.
func NewTable(name string, bounds core.Rect) *Table {
	t := &Table{
		base:         newBase(name, core.WidgetTable, bounds),
		rowHeight:    DefaultTableRowHeight,
		headerHeight: DefaultTableHeaderHeight,
		viewportH:    normalizedViewport(bounds.H),
		sortDir:      core.None,
	}
	t.rebuildOrder()
	return t
}

// SetBounds replaces the authored frame bounds and refreshes the default
// viewport used by data-driven scroll reconciliation. The UI may still call
// EnsureScrollBounds with a skin-aware content height afterward.
func (t *Table) SetBounds(bounds core.Rect) {
	if t == nil {
		return
	}
	oldBounds := t.Bounds()
	oldViewport := t.viewportH
	t.base.SetBounds(bounds)
	t.viewportH = normalizedViewport(bounds.H)
	scrollChanged := t.refreshScrollBounds()
	if oldBounds != t.Bounds() || oldViewport != t.viewportH || scrollChanged {
		t.bumpRevision()
	}
}

// Revision returns the table state revision used by external layout and
// tooltip caches. It advances for data, sort, selection, bounds, metric, and
// scroll changes, and is zero for an untouched table or nil receiver.
func (t *Table) Revision() uint64 {
	if t == nil {
		return 0
	}
	return t.revision
}

// Callbacks returns the table-owned callback slots, or nil for nil tables.
func (t *Table) Callbacks() *Callbacks {
	if t == nil {
		return nil
	}
	return t.base.Callbacks()
}

// Columns returns a defensive copy of the current column descriptors.
func (t *Table) Columns() []TableColumn {
	if t == nil || len(t.columns) == 0 {
		return nil
	}
	return append([]TableColumn(nil), t.columns...)
}

// ColumnCount returns the number of current columns without copying.
func (t *Table) ColumnCount() int {
	if t == nil {
		return 0
	}
	return len(t.columns)
}

// ColumnAt returns one column descriptor without exposing table storage.
func (t *Table) ColumnAt(index int) (TableColumn, bool) {
	if t == nil || index < 0 || index >= len(t.columns) {
		return TableColumn{}, false
	}
	return t.columns[index], true
}

// CopyColumnsInto copies columns into caller-owned reusable storage. It is
// the allocation-free counterpart to Columns for warmed render paths.
func (t *Table) CopyColumnsInto(dst []TableColumn) []TableColumn {
	if t == nil {
		clear(dst)
		return dst[:0]
	}
	oldLen := len(dst)
	if cap(dst) < len(t.columns) {
		dst = make([]TableColumn, len(t.columns))
	} else {
		for index := len(t.columns); index < oldLen; index++ {
			dst[index] = TableColumn{}
		}
		dst = dst[:len(t.columns)]
	}
	copy(dst, t.columns)
	return dst
}

// SetColumns atomically replaces column descriptors after validating stable
// IDs. Existing row cells are remapped by column ID, so reordering columns
// cannot silently attach a value to a different title. New columns are padded
// and removed columns are discarded; a vanished sort column clears sorting.
func (t *Table) SetColumns(columns []TableColumn) bool {
	if t == nil || !validTableColumns(columns) || equalTableColumns(t.columns, columns) {
		return false
	}
	oldColumns := append([]TableColumn(nil), t.columns...)
	t.columns = copyTableColumns(t.columns, columns)
	t.remapRows(oldColumns, t.columns)
	t.reconcileSortAfterColumns()
	t.rebuildOrder()
	t.bumpRevision()
	return true
}

// Rows returns a defensive deep copy of insertion-order rows and their cells.
// The returned slice may be freely edited without changing the widget.
func (t *Table) Rows() []TableRow {
	if t == nil || len(t.rows) == 0 {
		return nil
	}
	return copyTableRows(nil, t.rows, len(t.columns))
}

// SetRows atomically replaces insertion-order rows after validating non-empty,
// unique stable IDs. Input cells are copied and normalized to the current
// column count. Selection survives when its row ID remains, and the existing
// scroll offset is clamped against the newly derived maximum.
func (t *Table) SetRows(rows []TableRow) bool {
	if t == nil || !validTableRows(rows) || equalTableRows(t.rows, rows, len(t.columns)) {
		return false
	}
	t.rows = copyTableRows(t.rows, rows, len(t.columns))
	if t.selected != "" && findTableRow(t.rows, t.selected) < 0 {
		t.selected = ""
	}
	t.rebuildOrder()
	t.refreshScrollBounds()
	t.bumpRevision()
	return true
}

// RowCount returns the number of rows in insertion order.
func (t *Table) RowCount() int {
	if t == nil {
		return 0
	}
	return len(t.rows)
}

// RowAt returns a defensive copy of an insertion-order row.
func (t *Table) RowAt(index int) (TableRow, bool) {
	if t == nil || index < 0 || index >= len(t.rows) {
		return TableRow{}, false
	}
	return copyTableRow(t.rows[index], len(t.columns)), true
}

// IndexOfRow returns the insertion-order index for a stable row ID, or -1.
// Sorting never changes this index because the sorted view is separate.
func (t *Table) IndexOfRow(id string) int {
	if t == nil || id == "" {
		return -1
	}
	return findTableRow(t.rows, id)
}

// Selected returns the selected stable row ID, or false when unset.
func (t *Table) Selected() (string, bool) {
	if t == nil || t.selected == "" {
		return "", false
	}
	return t.selected, true
}

// Select chooses an existing row and reports whether the stable selection ID
// changed. Empty and unknown IDs are rejected without disturbing selection.
func (t *Table) Select(id string) bool {
	if t == nil || id == "" || t.selected == id || findTableRow(t.rows, id) < 0 {
		return false
	}
	t.selected = id
	t.bumpRevision()
	return true
}

// ClearSelection drops the current selection and reports whether one existed.
func (t *Table) ClearSelection() bool {
	if t == nil || t.selected == "" {
		return false
	}
	t.selected = ""
	t.bumpRevision()
	return true
}

// SortColumn returns the selected stable column ID and direction. It reports
// an empty ID and None when the table is currently in insertion order.
func (t *Table) SortColumn() (string, core.SortDir) {
	if t == nil || t.sortColumn == "" || t.sortDir == core.None {
		return "", core.None
	}
	return t.sortColumn, t.sortDir
}

// SetSort selects an existing sortable column and direction. Passing core.None
// is the programmatic equivalent of ClearSort; the two sort directions rebuild
// only the separate view and never mutate insertion-order rows.
func (t *Table) SetSort(columnID string, dir core.SortDir) bool {
	if t == nil {
		return false
	}
	if dir == core.None {
		return t.ClearSort()
	}
	if !validSortDirection(dir) || columnID == "" {
		return false
	}
	index := findTableColumn(t.columns, columnID)
	if index < 0 || !t.columns[index].Sortable {
		return false
	}
	if t.sortColumn == columnID && t.sortDir == dir {
		return false
	}
	t.sortColumn = columnID
	t.sortDir = dir
	t.rebuildOrder()
	t.bumpRevision()
	return true
}

// ClearSort restores insertion order and reports whether a sort was armed.
func (t *Table) ClearSort() bool {
	if t == nil || (t.sortColumn == "" && t.sortDir == core.None) {
		return false
	}
	t.sortColumn = ""
	t.sortDir = core.None
	t.rebuildOrder()
	t.bumpRevision()
	return true
}

// CellAt returns a defensive copy of one cell addressed by stable row and
// column IDs. It never exposes the widget's internal cell slice.
func (t *Table) CellAt(rowID, columnID string) (TableCell, bool) {
	if t == nil || rowID == "" || columnID == "" {
		return TableCell{}, false
	}
	rowIndex := findTableRow(t.rows, rowID)
	columnIndex := findTableColumn(t.columns, columnID)
	if rowIndex < 0 || columnIndex < 0 {
		return TableCell{}, false
	}
	return tableCellAt(t.rows[rowIndex], columnIndex), true
}

// SetCellText replaces only the text and optional numeric sort payload of a
// stable cell. It preserves icon, color, and invalid state, reports no change
// for identical input, and rebuilds the sorted view when its column is armed.
func (t *Table) SetCellText(rowID, columnID, text string, sortValue float64, hasSortValue bool) bool {
	if t == nil || rowID == "" || columnID == "" {
		return false
	}
	rowIndex := findTableRow(t.rows, rowID)
	columnIndex := findTableColumn(t.columns, columnID)
	if rowIndex < 0 || columnIndex < 0 {
		return false
	}
	cell := &t.rows[rowIndex].Cells[columnIndex]
	if cell.Text == text && equalFloat(cell.SortValue, sortValue) && cell.HasSortValue == hasSortValue {
		return false
	}
	cell.Text = text
	cell.SortValue = sortValue
	cell.HasSortValue = hasSortValue
	if t.sortColumn == columnID {
		t.rebuildOrder()
	}
	t.bumpRevision()
	return true
}

// Text returns the selected row's first-cell text for search and fallback.
// With no selection, or with a table that has no columns, it returns the base
// widget text inherited from the common widget implementation.
func (t *Table) Text() string {
	if t == nil {
		return ""
	}
	if t.selected != "" {
		if index := findTableRow(t.rows, t.selected); index >= 0 && len(t.rows[index].Cells) > 0 {
			return t.rows[index].Cells[0].Text
		}
	}
	return t.base.Text()
}

// ScrollOffset returns the current vertical scroll offset in logical pixels.
func (t *Table) ScrollOffset() float32 {
	if t == nil {
		return 0
	}
	return t.scrollY
}

// MaxScroll returns the current vertical scroll limit in logical pixels.
func (t *Table) MaxScroll() float32 {
	if t == nil {
		return 0
	}
	return t.maxScrollY
}

// SetScrollOffset replaces the offset clamped to [0, MaxScroll] and reports
// whether the stored value changed. Non-finite input is safely normalized.
func (t *Table) SetScrollOffset(offset float32) bool {
	if t == nil {
		return false
	}
	if !t.setScrollOffset(offset) {
		return false
	}
	t.bumpRevision()
	return true
}

// ScrollBy advances the vertical offset and reports whether it moved.
func (t *Table) ScrollBy(dy float32) bool {
	if t == nil || dy == 0 || math.IsNaN(float64(dy)) {
		return false
	}
	return t.SetScrollOffset(t.scrollY + dy)
}

// ScrollToRow minimally adjusts the view so the sorted-view row identified by
// id is inside the rows viewport below the fixed header. Unknown IDs and
// degenerate metrics do nothing; insertion order remains untouched.
func (t *Table) ScrollToRow(id string) bool {
	if t == nil || id == "" {
		return false
	}
	rowIndex := findTableRow(t.rows, id)
	viewIndex := t.viewIndexForInsertionRow(rowIndex)
	if rowIndex < 0 || viewIndex < 0 || !validTableHeight(t.RowHeight()) || !validTableHeight(t.HeaderHeight()) {
		return false
	}
	viewport := normalizedViewport(t.viewportH)
	if viewport <= t.HeaderHeight() {
		return false
	}
	rowTop := t.HeaderHeight() + float32(viewIndex)*t.RowHeight()
	rowBottom := rowTop + t.RowHeight()
	desired := t.scrollY
	if rowTop-desired < t.HeaderHeight() {
		desired = rowTop - t.HeaderHeight()
	} else if rowBottom-desired > viewport {
		desired = rowBottom - viewport
	}
	return t.SetScrollOffset(desired)
}

// EnsureScrollBounds derives the maximum from the total table viewport height
// and fixed header/data metrics, then clamps the current offset. It is safe to
// call after skin insets and scrollbar track exclusion are known.
func (t *Table) EnsureScrollBounds(viewportH float32) bool {
	if t == nil {
		return false
	}
	viewportH = normalizedViewport(viewportH)
	changed := t.viewportH != viewportH
	t.viewportH = viewportH
	maximum := tableMaxScroll(len(t.rows), t.RowHeight(), t.HeaderHeight(), viewportH)
	if t.maxScrollY != maximum {
		t.maxScrollY = maximum
		changed = true
	}
	if t.setScrollOffset(t.scrollY) {
		changed = true
	}
	if changed {
		t.bumpRevision()
	}
	return changed
}

// RowHeight returns the fixed data-row height, falling back for a zero-value
// or otherwise invalid table.
func (t *Table) RowHeight() float32 {
	if t == nil || !validTableHeight(t.rowHeight) {
		return DefaultTableRowHeight
	}
	return t.rowHeight
}

// SetRowHeight changes the fixed data-row height and returns the table for
// fluent configuration. Non-positive and non-finite values are ignored.
func (t *Table) SetRowHeight(height float32) *Table {
	if t == nil || !validTableHeight(height) || t.rowHeight == height {
		return t
	}
	t.rowHeight = height
	t.refreshScrollBounds()
	t.bumpRevision()
	return t
}

// HeaderHeight returns the fixed header height, falling back for a zero-value
// or otherwise invalid table.
func (t *Table) HeaderHeight() float32 {
	if t == nil || !validTableHeight(t.headerHeight) {
		return DefaultTableHeaderHeight
	}
	return t.headerHeight
}

// SetHeaderHeight changes the fixed header height and returns the table for
// fluent configuration. Non-positive and non-finite values are ignored.
func (t *Table) SetHeaderHeight(height float32) *Table {
	if t == nil || !validTableHeight(height) || t.headerHeight == height {
		return t
	}
	t.headerHeight = height
	t.refreshScrollBounds()
	t.bumpRevision()
	return t
}

// SetTooltip attaches a widget tooltip and returns the table for chaining.
func (t *Table) SetTooltip(text string) *Table {
	if t != nil {
		t.base.SetTooltip(text)
	}
	return t
}

// SetTooltipText replaces the tooltip slot through the common Widget API.
func (t *Table) SetTooltipText(text string) {
	if t != nil {
		t.base.SetTooltipText(text)
	}
}

// SetClass replaces the CSS class and invalidates table layout observers.
func (t *Table) SetClass(name string) {
	if t == nil || t.base.class == name {
		return
	}
	t.base.SetClass(name)
	t.bumpRevision()
}

// SetTextColor configures the default cell text color and returns the table.
func (t *Table) SetTextColor(color core.Color) *Table {
	if t != nil {
		t.base.SetTextColor(color)
	}
	return t
}

// SetFontSize configures the default cell font size and returns the table.
func (t *Table) SetFontSize(size float32) *Table {
	if t != nil {
		t.base.SetFontSize(size)
	}
	return t
}

// SetItalic configures the default cell italic style and returns the table.
func (t *Table) SetItalic(italic bool) *Table {
	if t != nil {
		t.base.SetItalic(italic)
	}
	return t
}

// SetAlign configures the default alignment for cells and returns the table.
func (t *Table) SetAlign(align core.TextAlign) *Table {
	if t != nil {
		t.base.SetAlign(align)
	}
	return t
}

// OnSelect attaches the direct stable-row selection callback.
func (t *Table) OnSelect(fn func(string)) *Table {
	if t != nil {
		t.callbacks.TableSelect = fn
	}
	return t
}

// OnSelectHandler preserves the List-parity callback inspection contract.
func (t *Table) OnSelectHandler() func(string) {
	if t == nil {
		return nil
	}
	return t.callbacks.TableSelect
}

// OnSort attaches the direct sorted-column callback.
func (t *Table) OnSort(fn func(string, core.SortDir)) *Table {
	if t != nil {
		t.callbacks.TableSort = fn
	}
	return t
}

// OnActivate attaches the direct semantic row-activation callback.
func (t *Table) OnActivate(fn func(string)) *Table {
	if t != nil {
		t.callbacks.TableActivate = fn
	}
	return t
}

// OnCellTooltip attaches the direct stable-cell tooltip provider.
func (t *Table) OnCellTooltip(fn func(string, string) string) *Table {
	if t != nil {
		t.callbacks.TableCellTooltip = fn
	}
	return t
}

// ViewRowCount returns the sorted-view row count without copying.
func (t *Table) ViewRowCount() int {
	if t == nil {
		return 0
	}
	return len(t.order)
}

// ViewRowIDAt returns a stable row ID at a sorted-view index without copying.
func (t *Table) ViewRowIDAt(index int) (string, bool) {
	insertion, ok := t.viewInsertionIndexAt(index)
	if !ok {
		return "", false
	}
	return t.rows[insertion].ID, true
}

// ViewCellAt returns a cell at sorted-view and column indices without
// allocating. It is intended for render and hit-test paths after warmup.
func (t *Table) ViewCellAt(viewIndex, columnIndex int) (TableCell, bool) {
	insertion, ok := t.viewInsertionIndexAt(viewIndex)
	if !ok || columnIndex < 0 || columnIndex >= len(t.columns) {
		return TableCell{}, false
	}
	return tableCellAt(t.rows[insertion], columnIndex), true
}

// viewInsertionIndexAt resolves a sorted-view index for internal render paths.
func (t *Table) viewInsertionIndexAt(index int) (int, bool) {
	if t == nil || index < 0 || index >= len(t.order) {
		return -1, false
	}
	insertion := t.order[index]
	if insertion < 0 || insertion >= len(t.rows) {
		return -1, false
	}
	return insertion, true
}

// remapRows preserves existing cells by stable column ID after a column edit.
func (t *Table) remapRows(oldColumns, newColumns []TableColumn) {
	if t == nil || len(t.rows) == 0 {
		return
	}
	remapped := make([]TableRow, len(t.rows))
	for i, row := range t.rows {
		remapped[i].ID = row.ID
		remapped[i].Cells = make([]TableCell, len(newColumns))
		for newIndex, column := range newColumns {
			oldIndex := findTableColumn(oldColumns, column.ID)
			if oldIndex >= 0 && oldIndex < len(row.Cells) {
				remapped[i].Cells[newIndex] = row.Cells[oldIndex]
			}
		}
	}
	clearTableRows(t.rows)
	t.rows = remapped
}

// reconcileSortAfterColumns drops sorting when its column is gone or no
// longer sortable, while retaining direction for a still-valid column.
func (t *Table) reconcileSortAfterColumns() {
	if t == nil || t.sortColumn == "" {
		return
	}
	index := findTableColumn(t.columns, t.sortColumn)
	if index < 0 || !t.columns[index].Sortable {
		t.sortColumn = ""
		t.sortDir = core.None
	}
}

// rebuildOrder recreates the view indexes from insertion order and applies
// the active column comparator without ever reordering the stored rows.
func (t *Table) rebuildOrder() {
	if t == nil {
		return
	}
	columnIndex := -1
	numeric := false
	if t.sortColumn != "" {
		columnIndex = findTableColumn(t.columns, t.sortColumn)
		if columnIndex >= 0 {
			numeric = t.columns[columnIndex].Numeric
		}
	}
	t.order, t.sortScratch = tableOrderInto(t.order, t.sortScratch, t.rows, columnIndex, numeric, t.sortDir)
}

// refreshScrollBounds recomputes the current limit from the remembered
// viewport and clamps the offset after rows or metrics change.
func (t *Table) refreshScrollBounds() bool {
	if t == nil {
		return false
	}
	maximum := tableMaxScroll(len(t.rows), t.RowHeight(), t.HeaderHeight(), t.viewportH)
	changed := t.maxScrollY != maximum
	t.maxScrollY = maximum
	if t.setScrollOffset(t.scrollY) {
		changed = true
	}
	return changed
}

// setScrollOffset applies the clamped offset without changing revision.
func (t *Table) setScrollOffset(offset float32) bool {
	if t == nil {
		return false
	}
	offset = clampTableScroll(offset, t.maxScrollY)
	if t.scrollY == offset {
		return false
	}
	t.scrollY = offset
	return true
}

// viewIndexForInsertionRow resolves an insertion index through the sorted
// order cache, returning -1 for unknown or stale indexes.
func (t *Table) viewIndexForInsertionRow(insertion int) int {
	if t == nil || insertion < 0 || insertion >= len(t.rows) {
		return -1
	}
	for view, index := range t.order {
		if index == insertion {
			return view
		}
	}
	return -1
}

// bumpRevision advances the non-zero table state revision.
func (t *Table) bumpRevision() {
	if t == nil {
		return
	}
	t.revision++
	if t.revision == 0 {
		t.revision = 1
	}
}

// validTableColumns validates stable non-empty unique column IDs.
func validTableColumns(columns []TableColumn) bool {
	seen := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		if column.ID == "" {
			return false
		}
		if _, exists := seen[column.ID]; exists {
			return false
		}
		seen[column.ID] = struct{}{}
	}
	return true
}

// validTableRows validates stable non-empty unique row IDs atomically.
func validTableRows(rows []TableRow) bool {
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.ID == "" {
			return false
		}
		if _, exists := seen[row.ID]; exists {
			return false
		}
		seen[row.ID] = struct{}{}
	}
	return true
}

// equalTableColumns compares descriptors without allocating.
func equalTableColumns(left, right []TableColumn) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// copyTableColumns copies descriptors into reusable storage without retaining
// removed string values in a shortened backing array.
func copyTableColumns(dst, source []TableColumn) []TableColumn {
	oldLen := len(dst)
	if cap(dst) < len(source) {
		dst = make([]TableColumn, len(source))
	} else {
		for index := len(source); index < oldLen; index++ {
			dst[index] = TableColumn{}
		}
		dst = dst[:len(source)]
	}
	copy(dst, source)
	return dst
}

// equalTableRows compares normalized stored rows with possibly short input.
func equalTableRows(stored, incoming []TableRow, columnCount int) bool {
	if len(stored) != len(incoming) {
		return false
	}
	for index := range stored {
		if stored[index].ID != incoming[index].ID {
			return false
		}
		for column := 0; column < columnCount; column++ {
			if !equalTableCell(tableCellAt(stored[index], column), tableCellAt(incoming[index], column)) {
				return false
			}
		}
	}
	return true
}

// equalTableCell compares all value-bearing fields, including exact colors
// and optional flags. NaN sort values are treated as equal for no-op updates.
func equalTableCell(left, right TableCell) bool {
	return left.Text == right.Text && left.Icon == right.Icon && left.Color == right.Color &&
		left.HasColor == right.HasColor && left.Invalid == right.Invalid &&
		equalFloat(left.SortValue, right.SortValue) && left.HasSortValue == right.HasSortValue
}

// equalFloat considers matching NaN payloads semantically equal for setters.
func equalFloat(left, right float64) bool {
	return left == right || (math.IsNaN(left) && math.IsNaN(right))
}

// copyTableRows deep-copies row cells into reusable destination storage.
func copyTableRows(dst []TableRow, source []TableRow, columnCount int) []TableRow {
	oldLen := len(dst)
	if cap(dst) < len(source) {
		dst = make([]TableRow, len(source))
	} else {
		for index := len(source); index < oldLen; index++ {
			clear(dst[index].Cells)
			dst[index] = TableRow{}
		}
		dst = dst[:len(source)]
	}
	for index, row := range source {
		dst[index].ID = row.ID
		dst[index].Cells = copyTableCells(dst[index].Cells, row.Cells, columnCount)
	}
	return dst
}

// copyTableCells copies only the cells represented by current columns and
// zeroes old tails before reusing a row's backing array.
func copyTableCells(dst, source []TableCell, columnCount int) []TableCell {
	clear(dst)
	if cap(dst) < columnCount {
		dst = make([]TableCell, columnCount)
	} else {
		dst = dst[:columnCount]
	}
	clear(dst)
	copy(dst, source)
	return dst
}

// copyTableRow returns a one-row defensive snapshot.
func copyTableRow(row TableRow, columnCount int) TableRow {
	return TableRow{ID: row.ID, Cells: copyTableCells(nil, row.Cells, columnCount)}
}

// clearTableRows releases cell values before replacing reusable row storage.
func clearTableRows(rows []TableRow) {
	for index := range rows {
		clear(rows[index].Cells)
		rows[index] = TableRow{}
	}
}

// findTableColumn returns a descriptor index by stable ID, or -1.
func findTableColumn(columns []TableColumn, id string) int {
	for index := range columns {
		if columns[index].ID == id {
			return index
		}
	}
	return -1
}

// findTableRow returns an insertion-order row index by stable ID, or -1.
func findTableRow(rows []TableRow, id string) int {
	for index := range rows {
		if rows[index].ID == id {
			return index
		}
	}
	return -1
}

// tableCellAt returns a zero cell for a short row rather than panicking.
func tableCellAt(row TableRow, columnIndex int) TableCell {
	if columnIndex < 0 || columnIndex >= len(row.Cells) {
		return TableCell{}
	}
	return row.Cells[columnIndex]
}

// validSortDirection accepts only the two directions a real sort can use.
func validSortDirection(direction core.SortDir) bool {
	return direction == core.SortAsc || direction == core.SortDesc
}

// tableOrderInto fills a reusable view-index slice and stably sorts it when
// requested. Ties explicitly use input indexes, preserving insertion order.
// Scratch storage is returned separately so Table rebuilds do not allocate on
// warmed sort changes.
func tableOrderInto(dst, scratch []int, rows []TableRow, columnIndex int, numeric bool, direction core.SortDir) ([]int, []int) {
	if cap(dst) < len(rows) {
		dst = make([]int, len(rows))
	} else {
		dst = dst[:len(rows)]
	}
	for index := range rows {
		dst[index] = index
	}
	if columnIndex < 0 || !validSortDirection(direction) {
		return dst, scratch
	}
	if cap(scratch) < len(rows) {
		scratch = make([]int, len(rows))
	} else {
		scratch = scratch[:len(rows)]
	}
	stableTableOrder(dst, scratch, rows, columnIndex, numeric, direction)
	return dst, scratch
}

// stableTableOrder merges adjacent runs in place through reusable scratch.
// Choosing the left item when neither side is less keeps equal values stable,
// while the comparator's insertion index remains the deterministic tie-break.
func stableTableOrder(order, scratch []int, rows []TableRow, columnIndex int, numeric bool, direction core.SortDir) {
	if len(order) < 2 {
		return
	}
	source, target := order, scratch
	fromScratch := false
	for width := 1; width < len(order); width *= 2 {
		for start := 0; start < len(order); start += width * 2 {
			mid := minTableIndex(start+width, len(order))
			end := minTableIndex(start+width*2, len(order))
			mergeTableOrder(target, source, start, mid, end, rows, columnIndex, numeric, direction)
		}
		source, target = target, source
		fromScratch = !fromScratch
	}
	if fromScratch {
		copy(order, source)
	}
}

// mergeTableOrder merges one pair of sorted index runs into target storage.
func mergeTableOrder(target, source []int, start, middle, end int, rows []TableRow, columnIndex int, numeric bool, direction core.SortDir) {
	left, right, output := start, middle, start
	for left < middle && right < end {
		if tableRowLess(rows, source[right], source[left], columnIndex, numeric, direction) {
			target[output] = source[right]
			right++
		} else {
			target[output] = source[left]
			left++
		}
		output++
	}
	for left < middle {
		target[output] = source[left]
		left++
		output++
	}
	for right < end {
		target[output] = source[right]
		right++
		output++
	}
}

// minTableIndex returns the smaller index without importing a general helper.
func minTableIndex(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// tableRowLess compares two insertion indexes for the active table sort.
func tableRowLess(rows []TableRow, leftIndex, rightIndex, columnIndex int, numeric bool, direction core.SortDir) bool {
	left := tableCellAt(rows[leftIndex], columnIndex)
	right := tableCellAt(rows[rightIndex], columnIndex)
	if numeric {
		return numericCellLess(left, right, leftIndex, rightIndex, direction)
	}
	return textCellLess(left.Text, right.Text, leftIndex, rightIndex, direction)
}

// numericCellLess orders present numeric values first, leaves missing values
// at the bottom in both directions, and uses text plus index for missing ties.
func numericCellLess(left, right TableCell, leftIndex, rightIndex int, direction core.SortDir) bool {
	leftMissing := !usableSortValue(left)
	rightMissing := !usableSortValue(right)
	if leftMissing != rightMissing {
		return !leftMissing
	}
	if leftMissing {
		return textCellLess(left.Text, right.Text, leftIndex, rightIndex, direction)
	}
	if left.SortValue != right.SortValue {
		if direction == core.SortDesc {
			return left.SortValue > right.SortValue
		}
		return left.SortValue < right.SortValue
	}
	return leftIndex < rightIndex
}

// textCellLess compares strings byte-wise and uses input index as a final tie.
func textCellLess(left, right string, leftIndex, rightIndex int, direction core.SortDir) bool {
	if left != right {
		if direction == core.SortDesc {
			return left > right
		}
		return left < right
	}
	return leftIndex < rightIndex
}

// usableSortValue treats NaN as absent so the comparator stays transitive.
func usableSortValue(cell TableCell) bool {
	return cell.HasSortValue && !math.IsNaN(cell.SortValue)
}

// TableSortRows returns insertion/view indexes stably ordered by one column.
// Numeric columns use present SortValue values, put missing values last, use
// text as the missing-value tie-break, and use the input index as final tie.
// Non-numeric columns compare Text byte-wise; input rows are never modified.
func TableSortRows(rows []TableRow, columnIndex int, numeric bool, direction core.SortDir) []int {
	if len(rows) == 0 {
		return nil
	}
	order, _ := tableOrderInto(nil, nil, rows, columnIndex, numeric, direction)
	return order
}

// TableMaxScroll derives the vertical limit from a total viewport containing
// a fixed header and data rows. Empty tables never scroll, even when their
// header would be taller than the viewport; invalid row/header metrics also
// produce zero so callers cannot create a moving but non-drawable table.
func TableMaxScroll(count int, rowHeight, headerHeight, viewportHeight float32) float32 {
	return tableMaxScroll(count, rowHeight, headerHeight, viewportHeight)
}

// tableMaxScroll contains the finite arithmetic shared by widget methods and
// the exported geometry helper.
func tableMaxScroll(count int, rowHeight, headerHeight, viewportHeight float32) float32 {
	if count <= 0 || !validTableHeight(rowHeight) || !validTableHeight(headerHeight) {
		return 0
	}
	if math.IsInf(float64(viewportHeight), 1) {
		return 0
	}
	viewportHeight = normalizedViewport(viewportHeight)
	total := float64(headerHeight) + float64(count)*float64(rowHeight)
	maximum := total - float64(viewportHeight)
	if maximum <= 0 || math.IsInf(maximum, 0) {
		return 0
	}
	if maximum >= float64(math.MaxFloat32) {
		return math.MaxFloat32
	}
	return float32(maximum)
}

// clampTableScroll normalizes an offset and confines it to a finite maximum.
func clampTableScroll(value, maximum float32) float32 {
	maximum = normalizedMaximum(maximum)
	if math.IsInf(float64(value), 1) {
		value = maximum
	}
	return clampScroll(value, maximum)
}

// normalizedMaximum rejects non-finite and negative explicit limits.
func normalizedMaximum(value float32) float32 {
	if value < 0 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		return 0
	}
	return value
}

// normalizedViewport converts invalid viewport heights to an empty viewport.
func normalizedViewport(value float32) float32 {
	if math.IsInf(float64(value), 1) {
		return math.MaxFloat32
	}
	if value <= 0 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		return 0
	}
	return value
}

// validTableHeight reports whether a fixed metric can produce drawable rows.
func validTableHeight(value float32) bool {
	return value > 0 && !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}
