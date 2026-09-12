package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// tableContentRect returns the skin-aware table shell viewport.
func (u *UI) tableContentRect(table *widgets.Table) core.Rect {
	if table == nil {
		return core.Rect{}
	}
	if u == nil || u.theme == nil {
		return table.Bounds()
	}
	return u.theme.TableContent(table.Bounds(), u.visualState(table), table.Class())
}

// reconcileTableBounds derives the table's scroll limit before hit testing or
// drawing. Callers derive the track-excluded row width from its returned shell.
func (u *UI) reconcileTableBounds(table *widgets.Table) core.Rect {
	if table == nil {
		return core.Rect{}
	}
	content := u.tableContentRect(table)
	table.EnsureScrollBounds(content.H)
	return content
}

// tableGeometry returns the reconciled shell, shared row viewport, and cached
// relative columns used by table hit testing.
func (u *UI) tableGeometry(table *widgets.Table) (core.Rect, core.Rect, []core.Rect) {
	content := u.reconcileTableBounds(table)
	if table == nil {
		return content, core.Rect{}, nil
	}
	rows := render.TableRowsContent(content, table.MaxScroll())
	return content, rows, u.tableColumnSlots(table, rows, table.MaxScroll())
}

// tableColumnSlots returns cached relative slots for UI hit testing. The
// caller-owned scratch is copied by TableLayoutCache before it is reused.
func (u *UI) tableColumnSlots(table *widgets.Table, content core.Rect, maxScroll float32) []core.Rect {
	if u == nil || table == nil {
		return nil
	}
	if u.tableLayouts == nil {
		u.tableLayouts = make(map[string]*render.TableLayoutCache)
	}
	cache := u.tableLayouts[table.Name()]
	if cache == nil {
		cache = &render.TableLayoutCache{}
		u.tableLayouts[table.Name()] = cache
	}
	u.tableColumns = table.CopyColumnsInto(u.tableColumns[:0])
	state := u.visualState(table)
	key := uint64(0)
	if u.theme != nil {
		key = (u.theme.TextRevision() << 8) | uint64(state+1)
	}
	cache.Update(content.W, u.tableColumns, content, maxScroll > 0, key, table.Class(), u.theme != nil && u.theme.GetPixelSnap())
	return cache.Slots()
}

// tableRowAt resolves a sorted-view row ID under pos, or false outside the
// rows viewport. The returned identity survives sorting and data refreshes.
func (u *UI) tableRowAt(table *widgets.Table, pos core.Vec2) (string, bool) {
	if table == nil {
		return "", false
	}
	_, content, _ := u.tableGeometry(table)
	index, header := render.TableRowAt(content, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset(), pos)
	if header || index < 0 {
		return "", false
	}
	id, ok := table.ViewRowIDAt(index)
	return id, ok
}

// tableHeaderColumnAt resolves a header column ID under pos, including
// unsortable columns so a press can be armed and safely canceled on release.
func (u *UI) tableHeaderColumnAt(table *widgets.Table, pos core.Vec2) (string, bool) {
	if table == nil {
		return "", false
	}
	_, content, slots := u.tableGeometry(table)
	_, header := render.TableRowAt(content, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset(), pos)
	if !header {
		return "", false
	}
	headerRect := render.TableHeaderRect(content, table.HeaderHeight())
	index := render.TableCellAt(content, slots, headerRect, pos)
	if index < 0 {
		return "", false
	}
	column, ok := table.ColumnAt(index)
	if !ok {
		return "", false
	}
	return column.ID, true
}

// tableCellAt resolves a stable row and column ID under the pointer for
// tooltip lookup. It intentionally does not diagnose ordinary pointer misses.
func (u *UI) tableCellAt(table *widgets.Table, pos core.Vec2) (string, string, bool) {
	if table == nil {
		return "", "", false
	}
	_, content, slots := u.tableGeometry(table)
	viewIndex, header := render.TableRowAt(content, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset(), pos)
	if header || viewIndex < 0 {
		return "", "", false
	}
	row, ok := render.TableRowRect(content, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset(), viewIndex)
	if !ok {
		return "", "", false
	}
	columnIndex := render.TableCellAt(content, slots, row, pos)
	if columnIndex < 0 {
		return "", "", false
	}
	rowID, ok := table.ViewRowIDAt(viewIndex)
	if !ok {
		return "", "", false
	}
	column, ok := table.ColumnAt(columnIndex)
	if !ok {
		return "", "", false
	}
	return rowID, column.ID, true
}

// tableHoverRowID returns the hovered row only while this table owns hover.
func (u *UI) tableHoverRowID(table *widgets.Table) string {
	if table == nil || u.hovered != table {
		return ""
	}
	id, _ := u.tableRowAt(table, u.pointer)
	return id
}

// tableHoverColumnID returns the hovered header column only while this table
// owns hover; row hover and header hover are mutually exclusive.
func (u *UI) tableHoverColumnID(table *widgets.Table) string {
	if table == nil || u.hovered != table {
		return ""
	}
	id, _ := u.tableHeaderColumnAt(table, u.pointer)
	return id
}

// tablePressArms stores a stable row or header identity for MSFT release
// matching. Blank areas clear both arms without making a table action.
func (u *UI) tablePressArms(table *widgets.Table, pos core.Vec2) {
	u.clearTableArm()
	if table == nil {
		return
	}
	if rowID, ok := u.tableRowAt(table, pos); ok {
		u.tableArmedRow = rowID
		return
	}
	if columnID, ok := u.tableHeaderColumnAt(table, pos); ok {
		u.tableArmedColumn = columnID
	}
}

// clearTableArm forgets the current table row/header press identity.
func (u *UI) clearTableArm() {
	if u == nil {
		return
	}
	u.tableArmedRow = ""
	u.tableArmedColumn = ""
}

// drawTable renders one table with stable-ID pointer state and a shared
// scrollbar gesture state.
func (u *UI) drawTable(table *widgets.Table, state core.WidgetState) {
	if table == nil || u.theme == nil {
		return
	}
	info := table.Snapshot(state)
	content := u.reconcileTableBounds(table)
	if u.hovered == table {
		u.refreshTableTip(table)
	}
	rows := render.TableRowsContent(content, table.MaxScroll())
	slots := u.tableColumnSlots(table, rows, table.MaxScroll())
	u.theme.DrawTableWithState(info, table, render.TableDrawState{
		HoveredRow:    u.tableHoverRowID(table),
		PressedRow:    u.tableArmedRow,
		HoveredColumn: u.tableHoverColumnID(table),
		PressedColumn: u.tableArmedColumn,
		ThumbState:    u.scrollThumbStateFor(table),
		ColumnSlots:   slots,
	})
	if needsBorder(table.Kind()) {
		u.theme.DrawWidgetPart(table.Kind(), skin.PartBorder, table.Bounds(), state, table.Class())
	}
}

// releaseTable commits only an identity-matched row or header release. A
// mismatched release is consumed but never selects or sorts anything.
func (u *UI) releaseTable(table *widgets.Table, pos core.Vec2) bool {
	armedRow, armedColumn := u.tableArmedRow, u.tableArmedColumn
	u.clearTableArm()
	if table == nil || !table.Enabled() {
		return true
	}
	if armedRow != "" {
		if rowID, ok := u.tableRowAt(table, pos); ok && rowID == armedRow {
			u.commitTableSelection(table, rowID)
		}
		return true
	}
	if armedColumn != "" {
		if columnID, ok := u.tableHeaderColumnAt(table, pos); ok && columnID == armedColumn {
			u.commitTableSort(table, columnID)
		}
	}
	return true
}

// commitTableSelection mutates selection before firing the change callback;
// every matched row release still fires the shared OnClick callback.
func (u *UI) commitTableSelection(table *widgets.Table, id string) {
	if table == nil {
		return
	}
	if table.Select(id) {
		u.fireOnTableSelect(table.Name(), id)
	}
	if u.Lookup(table.Name()) == table {
		u.fireOnClick(table.Name())
	}
}

// commitTableSort toggles the selected column between ascending and descending;
// every matched header release fires OnClick, while unsortable headers do not
// mutate the sorted view.
func (u *UI) commitTableSort(table *widgets.Table, columnID string) {
	if table == nil || columnID == "" {
		return
	}
	column, ok := tableColumnByID(table, columnID)
	if !ok {
		return
	}
	if !column.Sortable {
		// The header identity matched, so the generic click still observes
		// the release even though sorting is intentionally a no-op.
		if u.Lookup(table.Name()) == table {
			u.fireOnClick(table.Name())
		}
		return
	}
	current, direction := table.SortColumn()
	next := core.SortAsc
	if current == columnID && direction == core.SortAsc {
		next = core.SortDesc
	}
	if table.SetSort(columnID, next) {
		u.fireOnTableSort(table.Name(), columnID, next)
	}
	if u.Lookup(table.Name()) == table {
		u.fireOnClick(table.Name())
	}
}

// tableColumnByID finds a column without exposing table storage.
func tableColumnByID(table *widgets.Table, id string) (widgets.TableColumn, bool) {
	if table == nil || id == "" {
		return widgets.TableColumn{}, false
	}
	for index := 0; index < table.ColumnCount(); index++ {
		column, ok := table.ColumnAt(index)
		if ok && column.ID == id {
			return column, true
		}
	}
	return widgets.TableColumn{}, false
}

// lookupTable resolves a registered, available table for semantic operations.
func (u *UI) lookupTable(op, name string) (*widgets.Table, bool) {
	target := u.Lookup(name)
	if target == nil {
		u.diagnose("ui.%s: widget %q not found", op, name)
		return nil, false
	}
	table, ok := target.(*widgets.Table)
	if !ok {
		u.diagnose("ui.%s: widget %q is %v, expected WidgetTable", op, name, target.Kind())
		return nil, false
	}
	if !u.available(table) {
		u.diagnose("ui.%s: widget %q is disabled", op, name)
		return nil, false
	}
	return table, true
}

// lookupTableRow validates a stable row ID and reports actionable diagnostics.
func (u *UI) lookupTableRow(op, name, rowID string) (*widgets.Table, bool) {
	table, ok := u.lookupTable(op, name)
	if !ok {
		return nil, false
	}
	if rowID == "" {
		u.diagnose("ui.%s: empty row ID for %q", op, name)
		return nil, false
	}
	if table.IndexOfRow(rowID) < 0 {
		u.diagnose("ui.%s: row ID %q not found in %q", op, rowID, name)
		return nil, false
	}
	return table, true
}

// SelectTableRow changes a table selection through the shared semantic path.
// OnTableSelect fires only on a real ID change; OnClick fires for a valid row.
func (u *UI) SelectTableRow(name, rowID string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	table, ok := u.lookupTableRow("SelectTableRow", name, rowID)
	if !ok {
		return false
	}
	u.commitTableSelection(table, rowID)
	return true
}

// SetTableSort changes or clears a table sort without manufacturing pointer
// state. None is programmatic clearing and never produces an invalid column.
func (u *UI) SetTableSort(name, columnID string, direction core.SortDir) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	table, ok := u.lookupTable("SetTableSort", name)
	if !ok {
		return false
	}
	if direction == core.None {
		changed := table.ClearSort()
		if changed {
			u.fireOnTableSort(name, "", core.None)
		}
		return true
	}
	if direction != core.SortAsc && direction != core.SortDesc {
		u.diagnose("ui.SetTableSort: invalid direction %d for %q", direction, name)
		return false
	}
	column, ok := tableColumnByID(table, columnID)
	if !ok {
		u.diagnose("ui.SetTableSort: column ID %q not found in %q", columnID, name)
		return false
	}
	if !column.Sortable {
		u.diagnose("ui.SetTableSort: column ID %q in %q is not sortable", columnID, name)
		return false
	}
	changed := table.SetSort(columnID, direction)
	if changed {
		u.fireOnTableSort(name, columnID, direction)
	}
	return true
}

// commitTableActivate invokes the semantic activation callback for one stable
// row. It is deliberately separate from pointer selection: table releases
// never manufacture activation or double-click behavior.
func (u *UI) commitTableActivate(table *widgets.Table, rowID string) {
	if table == nil || rowID == "" {
		return
	}
	u.fireOnTableActivate(table.Name(), rowID)
}

// ActivateTableRow invokes only the semantic row activation callback. Pointer
// releases never call this path, and no double-click or keyboard path exists.
func (u *UI) ActivateTableRow(name, rowID string) bool {
	if u == nil {
		return false
	}
	u.reconcileInteraction()
	table, ok := u.lookupTableRow("ActivateTableRow", name, rowID)
	if !ok {
		return false
	}
	u.commitTableActivate(table, rowID)
	return true
}

// TableSelection returns the selected stable row ID, or false when unset or
// when name does not identify an available table.
func (u *UI) TableSelection(name string) (string, bool) {
	if u == nil {
		return "", false
	}
	u.reconcileInteraction()
	table, ok := u.lookupTable("TableSelection", name)
	if !ok {
		return "", false
	}
	return table.Selected()
}

// refreshTableTip updates the cell tooltip provider only when row, column, or
// table data revision changes. Static widget tooltip fallback remains in the
// common derived tooltip path.
func (u *UI) refreshTableTip(table *widgets.Table) {
	if table == nil {
		u.dismissLinkTip()
		return
	}
	rowID, columnID, ok := u.tableCellAt(table, u.pointer)
	if !ok {
		u.dismissLinkTip()
		return
	}
	revision := table.Revision()
	if u.tipWidget == table && u.tableTipRow == rowID && u.tableTipColumn == columnID && u.tipRevision == revision {
		return
	}
	if u.tipWidget != table || u.tableTipRow != rowID || u.tableTipColumn != columnID {
		u.hoverSince = u.tooltipNow()
	}
	u.tipWidget = table
	u.tipSeg = -1
	u.tipChatMsg = 0
	u.tableTipRow = rowID
	u.tableTipColumn = columnID
	u.tipRevision = revision
	u.tipText = ""
	if callbacks := table.Callbacks(); callbacks != nil && callbacks.TableCellTooltip != nil {
		u.tipText = callbacks.TableCellTooltip(rowID, columnID)
	}
}
