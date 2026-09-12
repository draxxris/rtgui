package widgets

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// tableContractColumns returns columns that exercise text and numeric sorting.
func tableContractColumns() []TableColumn {
	return []TableColumn{
		{ID: "name", Title: "Name", Width: 2, Sortable: true},
		{ID: "amount", Title: "Amount", Width: 1, Sortable: true, Numeric: true},
	}
}

// tableContractRows returns stable insertion data with numeric ties and misses.
func tableContractRows() []TableRow {
	return []TableRow{
		{ID: "first", Cells: []TableCell{{Text: "first"}, {Text: "2", SortValue: 2, HasSortValue: true}}},
		{ID: "tie-a", Cells: []TableCell{{Text: "tie-a"}, {Text: "2", SortValue: 2, HasSortValue: true}}},
		{ID: "low", Cells: []TableCell{{Text: "low"}, {Text: "1", SortValue: 1, HasSortValue: true}}},
		{ID: "missing-z", Cells: []TableCell{{Text: "missing-z"}, {Text: "z"}}},
		{ID: "tie-b", Cells: []TableCell{{Text: "tie-b"}, {Text: "2", SortValue: 2, HasSortValue: true}}},
		{ID: "missing-a", Cells: []TableCell{{Text: "missing-a"}, {Text: "a"}}},
	}
}

// tableViewIDs returns stable IDs in sorted-view order for concise assertions.
func tableViewIDs(t *testing.T, table *Table) []string {
	t.Helper()
	ids := make([]string, table.ViewRowCount())
	for index := range ids {
		id, ok := table.ViewRowIDAt(index)
		if !ok {
			t.Fatalf("view row %d is unavailable", index)
		}
		ids[index] = id
	}
	return ids
}

// TestTableSortKeepsInsertionOrderAndStableTies checks the separate view.
func TestTableSortKeepsInsertionOrderAndStableTies(t *testing.T) {
	table := NewTable("orders", core.Rect{W: 240, H: 120})
	if !table.SetColumns(tableContractColumns()) || !table.SetRows(tableContractRows()) {
		t.Fatal("table fixture rejected")
	}
	if !table.SetSort("amount", core.SortAsc) {
		t.Fatal("ascending sort was not armed")
	}
	wantAsc := []string{"low", "first", "tie-a", "tie-b", "missing-a", "missing-z"}
	if got := tableViewIDs(t, table); !sameStrings(got, wantAsc) {
		t.Fatalf("ascending view = %v, want %v", got, wantAsc)
	}
	if row, ok := table.RowAt(0); !ok || row.ID != "first" {
		t.Fatalf("RowAt changed insertion order: %+v/%v", row, ok)
	}
	if table.IndexOfRow("tie-b") != 4 {
		t.Fatalf("insertion index changed: %d", table.IndexOfRow("tie-b"))
	}
	if !table.SetSort("amount", core.SortDesc) {
		t.Fatal("descending sort was not armed")
	}
	wantDesc := []string{"first", "tie-a", "tie-b", "low", "missing-z", "missing-a"}
	if got := tableViewIDs(t, table); !sameStrings(got, wantDesc) {
		t.Fatalf("descending view = %v, want %v", got, wantDesc)
	}
}

// newColumnRemapTable creates short and excess cell rows for remap tests.
func newColumnRemapTable(t *testing.T) *Table {
	t.Helper()
	table := NewTable("columns", core.Rect{W: 240, H: 120})
	columns := []TableColumn{{ID: "name"}, {ID: "price"}, {ID: "stock"}}
	rows := []TableRow{
		{ID: "short", Cells: []TableCell{{Text: "s-name"}}},
		{ID: "excess", Cells: []TableCell{{Text: "e-name"}, {Text: "e-price"}, {Text: "e-stock"}, {Text: "discarded"}}},
	}
	if !table.SetColumns(columns) || !table.SetRows(rows) {
		t.Fatal("column remap fixture rejected")
	}
	return table
}

// TestTableColumnRemapNormalizesShortAndExcess checks cell count bounds.
func TestTableColumnRemapNormalizesShortAndExcess(t *testing.T) {
	table := newColumnRemapTable(t)
	if row, ok := table.RowAt(0); !ok || len(row.Cells) != 3 || row.Cells[1] != (TableCell{}) {
		t.Fatalf("short row normalization = %+v/%v", row, ok)
	}
	if row, ok := table.RowAt(1); !ok || len(row.Cells) != 3 || row.Cells[2].Text != "e-stock" {
		t.Fatalf("excess row normalization = %+v/%v", row, ok)
	}
}

// TestTableColumnRemapByID checks reordering preserves stable cell identity.
func TestTableColumnRemapByID(t *testing.T) {
	table := newColumnRemapTable(t)
	if !table.SetColumns([]TableColumn{{ID: "stock"}, {ID: "name"}, {ID: "new"}}) {
		t.Fatal("reordered columns rejected")
	}
	checks := []struct{ row, column, text string }{
		{"excess", "stock", "e-stock"}, {"excess", "name", "e-name"}, {"short", "name", "s-name"},
	}
	for _, check := range checks {
		cell, ok := table.CellAt(check.row, check.column)
		if !ok || cell.Text != check.text {
			t.Fatalf("%s/%s = %+v/%v, want %q", check.row, check.column, cell, ok, check.text)
		}
	}
	if cell, ok := table.CellAt("short", "new"); !ok || cell != (TableCell{}) {
		t.Fatalf("new column padding = %+v/%v", cell, ok)
	}
}

// TestTableNilColumnsAndRowsClearData checks nil replacement semantics.
func TestTableNilColumnsAndRowsClearData(t *testing.T) {
	table := newColumnRemapTable(t)
	if !table.SetColumns(nil) || table.ColumnCount() != 0 {
		t.Fatal("nil columns did not clear columns")
	}
	if row, ok := table.RowAt(0); !ok || len(row.Cells) != 0 {
		t.Fatalf("nil column remap retained cells = %+v/%v", row, ok)
	}
	if !table.SetRows(nil) || table.RowCount() != 0 {
		t.Fatal("nil rows did not clear rows")
	}
}

// TestTableSetCellTextOnlyRebuildsActiveSortColumn checks sort invalidation.
func TestTableSetCellTextOnlyRebuildsActiveSortColumn(t *testing.T) {
	table := NewTable("cell", core.Rect{W: 240, H: 120})
	if !table.SetColumns(tableContractColumns()) || !table.SetRows(tableContractRows()[:3]) || !table.SetSort("amount", core.SortAsc) {
		t.Fatal("cell fixture rejected")
	}
	before := &table.order[0]
	want := []string{"low", "first", "tie-a"}
	if got := tableViewIDs(t, table); !sameStrings(got, want) {
		t.Fatalf("initial order = %v, want %v", got, want)
	}
	if !table.SetCellText("first", "name", "first updated", 2, true) {
		t.Fatal("non-sort cell did not change")
	}
	if &table.order[0] != before || !sameStrings(tableViewIDs(t, table), want) {
		t.Fatal("non-sort cell rebuilt or changed the view")
	}
	if !table.SetCellText("first", "amount", "0", 0, true) {
		t.Fatal("active sort cell did not change")
	}
	if got := tableViewIDs(t, table); !sameStrings(got, []string{"first", "low", "tie-a"}) {
		t.Fatalf("active sort order = %v", got)
	}
}

// TestTableSnapshotsAreDefensiveAndEqualUpdatesAreNoops checks copy semantics.
func TestTableSnapshotsAreDefensiveAndEqualUpdatesAreNoops(t *testing.T) {
	table := NewTable("copies", core.Rect{W: 240, H: 120})
	columns := tableContractColumns()
	rows := tableContractRows()[:2]
	if !table.SetColumns(columns) || !table.SetRows(rows) {
		t.Fatal("table fixture rejected")
	}
	columns[0].Title = "mutated"
	if got, _ := table.ColumnAt(0); got.Title == "mutated" {
		t.Fatal("column input aliased table")
	}
	snapshot := table.Rows()
	snapshot[0].Cells[0].Text = "mutated snapshot"
	if cell, _ := table.CellAt("first", "name"); cell.Text != "first" {
		t.Fatal("Rows returned aliased cells")
	}
	row, _ := table.RowAt(0)
	row.Cells[0].Text = "mutated row"
	if cell, _ := table.CellAt("first", "name"); cell.Text != "first" {
		t.Fatal("RowAt returned aliased cells")
	}
	cell, _ := table.CellAt("first", "name")
	cell.Text = "mutated cell"
	if current, _ := table.CellAt("first", "name"); current.Text != "first" {
		t.Fatal("CellAt returned mutable storage")
	}
	if table.SetRows(table.Rows()) {
		t.Fatal("equal normalized rows reported a change")
	}
	if table.SetColumns(table.Columns()) {
		t.Fatal("equal columns reported a change")
	}
}

// sameStrings compares short expected ID slices without pulling in helpers.
func sameStrings(left, right []string) bool {
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
