package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/widgets"
)

// tableUIFixture creates one populated table with a sortable numeric column.
func tableUIFixture(t *testing.T, bounds core.Rect) (*UI, *widgets.Table) {
	t.Helper()
	u := New(500, 400)
	table := widgets.NewTable("table", bounds)
	if !table.SetColumns([]widgets.TableColumn{
		{ID: "name", Title: "Name", Width: 2},
		{ID: "amount", Title: "Amount", Width: 1, Sortable: true, Numeric: true},
	}) || !table.SetRows([]widgets.TableRow{
		{ID: "first", Cells: []widgets.TableCell{{Text: "First"}, {Text: "2", SortValue: 2, HasSortValue: true}}},
		{ID: "second", Cells: []widgets.TableCell{{Text: "Second"}, {Text: "1", SortValue: 1, HasSortValue: true}}},
		{ID: "third", Cells: []widgets.TableCell{{Text: "Third"}, {Text: "3", SortValue: 3, HasSortValue: true}}},
	}) {
		t.Fatal("table fixture rejected")
	}
	mustAdd(t, u, table)
	return u, table
}

// tableUIRowCenter resolves a row center through the UI's cached geometry.
func tableUIRowCenter(u *UI, table *widgets.Table, index int) core.Vec2 {
	_, content, _ := u.tableGeometry(table)
	row, _ := render.TableRowRect(content, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset(), index)
	return core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
}

// tableUIHeaderCenter resolves a header cell center through shared slots.
func tableUIHeaderCenter(u *UI, table *widgets.Table, index int) core.Vec2 {
	_, content, slots := u.tableGeometry(table)
	header, _ := render.TableHeaderCellRect(content, slots, index, table.HeaderHeight())
	return core.Vec2{X: header.X + header.W/2, Y: header.Y + header.H/2}
}

// TestTableMatchedAndMismatchedPressRelease checks stable row arming.
func TestTableMatchedAndMismatchedPressRelease(t *testing.T) {
	u, table := tableUIFixture(t, core.Rect{X: 20, Y: 20, W: 260, H: 150})
	selected := []string{}
	clicks := 0
	u.OnTableSelect("table", func(id string) { selected = append(selected, id) })
	u.OnClick("table", func() { clicks++ })
	first := tableUIRowCenter(u, table, 0)
	second := tableUIRowCenter(u, table, 1)
	if !u.HandleMouse(MouseEvent{Pos: first, Pressed: true}) || !u.HandleMouse(MouseEvent{Pos: first, Released: true}) {
		t.Fatal("matched row release was not consumed")
	}
	if id, ok := table.Selected(); !ok || id != "first" || len(selected) != 1 || selected[0] != "first" || clicks != 1 {
		t.Fatalf("matched row result = %q/%v selected=%v clicks=%d", id, ok, selected, clicks)
	}
	if !u.HandleMouse(MouseEvent{Pos: first, Pressed: true}) || !u.HandleMouse(MouseEvent{Pos: second, Released: true}) {
		t.Fatal("mismatched row release was not consumed")
	}
	if id, _ := table.Selected(); id != "first" || len(selected) != 1 || clicks != 1 {
		t.Fatalf("mismatched row changed selection=%q selected=%v clicks=%d", id, selected, clicks)
	}
}

// TestTableUnsortableHeaderIsNoop checks an unsortable header release.
func TestTableUnsortableHeaderIsNoop(t *testing.T) {
	u, table := tableUIFixture(t, core.Rect{X: 20, Y: 20, W: 260, H: 150})
	sorts, clicks := 0, 0
	u.OnTableSort("table", func(string, core.SortDir) { sorts++ })
	u.OnClick("table", func() { clicks++ })
	pos := tableUIHeaderCenter(u, table, 0)
	if !u.HandleMouse(MouseEvent{Pos: pos, Pressed: true}) || !u.HandleMouse(MouseEvent{Pos: pos, Released: true}) {
		t.Fatal("unsortable header release was not consumed")
	}
	column, direction := table.SortColumn()
	if column != "" || direction != core.None || sorts != 0 || clicks != 1 {
		t.Fatalf("unsortable header mutated sort=%q/%v callbacks=%d/%d", column, direction, sorts, clicks)
	}
}

// TestTableSortableHeaderToggles checks the two-state header cycle.
func TestTableSortableHeaderToggles(t *testing.T) {
	u, table := tableUIFixture(t, core.Rect{X: 20, Y: 20, W: 260, H: 150})
	sorts, clicks := 0, 0
	u.OnTableSort("table", func(string, core.SortDir) { sorts++ })
	u.OnClick("table", func() { clicks++ })
	pos := tableUIHeaderCenter(u, table, 1)
	if !u.HandleMouse(MouseEvent{Pos: pos, Pressed: true}) || !u.HandleMouse(MouseEvent{Pos: pos, Released: true}) {
		t.Fatal("sortable header release was not consumed")
	}
	column, direction := table.SortColumn()
	if column != "amount" || direction != core.SortAsc || sorts != 1 || clicks != 1 {
		t.Fatalf("sortable header result=%q/%v callbacks=%d/%d", column, direction, sorts, clicks)
	}
	if !u.HandleMouse(MouseEvent{Pos: pos, Pressed: true}) || !u.HandleMouse(MouseEvent{Pos: pos, Released: true}) {
		t.Fatal("second sortable header release was not consumed")
	}
	_, direction = table.SortColumn()
	if direction != core.SortDesc || sorts != 2 || clicks != 2 {
		t.Fatalf("second header result=%v callbacks=%d/%d", direction, sorts, clicks)
	}
}

// TestTableMismatchedHeaderReleaseIsNoop checks stable column arming.
func TestTableMismatchedHeaderReleaseIsNoop(t *testing.T) {
	u, table := tableUIFixture(t, core.Rect{X: 20, Y: 20, W: 260, H: 150})
	sorts, clicks := 0, 0
	u.OnTableSort("table", func(string, core.SortDir) { sorts++ })
	u.OnClick("table", func() { clicks++ })
	amount, name := tableUIHeaderCenter(u, table, 1), tableUIHeaderCenter(u, table, 0)
	u.HandleMouse(MouseEvent{Pos: amount, Pressed: true})
	if !u.HandleMouse(MouseEvent{Pos: name, Released: true}) {
		t.Fatal("mismatched header release was not consumed")
	}
	column, direction := table.SortColumn()
	if column != "" || direction != core.None || sorts != 0 || clicks != 0 {
		t.Fatalf("mismatched header changed sort=%q/%v callbacks=%d/%d", column, direction, sorts, clicks)
	}
}

// TestTableNestedWheelLetsOverflowingOuterWin checks scroll ownership.
func TestTableNestedWheelLetsOverflowingOuterWin(t *testing.T) {
	u := New(500, 400)
	outer := widgets.NewScrollPanel("outer", core.Rect{X: 20, Y: 20, W: 320, H: 180}).SetMaxScroll(core.Vec2{Y: 120})
	table := widgets.NewTable("inner", core.Rect{X: 20, Y: 20, W: 260, H: 100})
	if !table.SetColumns([]widgets.TableColumn{{ID: "name", Title: "Name", Width: 1}}) || !table.SetRows([]widgets.TableRow{{ID: "one", Cells: []widgets.TableCell{{Text: "one"}}}}) {
		t.Fatal("nested table fixture rejected")
	}
	mustAdd(t, u, outer, table)
	mustParent(t, u, "inner", "outer")
	if err := layout.Arrange(outer.Frame(), core.Rect{}); err != nil {
		t.Fatal(err)
	}
	pos := centerOf(table)
	if table.MaxScroll() != 0 {
		t.Fatalf("fitting inner table max scroll = %v", table.MaxScroll())
	}
	if !u.HandleMouse(MouseEvent{Pos: pos, Wheel: -1}) {
		t.Fatal("nested wheel was not consumed")
	}
	if outer.Scroll().Y <= 0 || table.ScrollOffset() != 0 {
		t.Fatalf("nested wheel offsets outer=%v inner=%v", outer.Scroll().Y, table.ScrollOffset())
	}
}

// TestTableTooltipTracksCellIDAndRevision checks stationary hover refresh.
func TestTableTooltipTracksCellIDAndRevision(t *testing.T) {
	u, table := tableUIFixture(t, core.Rect{X: 20, Y: 20, W: 260, H: 150})
	requests := []string{}
	u.OnTableCellTooltip("table", func(rowID, columnID string) string {
		requests = append(requests, rowID+":"+columnID)
		return "tip-" + rowID + "-" + columnID
	})
	pos := tableUIRowCenter(u, table, 0)
	if u.HandleMouse(MouseEvent{Pos: pos}) {
		t.Fatal("hover alone must pass through")
	}
	if text, _, ok := u.derivedTooltip(); !ok || text != "tip-first-name" {
		t.Fatalf("initial tooltip=%q/%v", text, ok)
	}
	u.HandleMouse(MouseEvent{Pos: pos})
	if len(requests) != 1 {
		t.Fatalf("stationary hover requested %v times", len(requests))
	}
	if !table.SetSort("amount", core.SortAsc) {
		t.Fatal("sort did not change table revision")
	}
	u.HandleMouse(MouseEvent{Pos: pos})
	if len(requests) != 2 || requests[1] != "second:name" {
		t.Fatalf("sort tooltip requests=%v", requests)
	}
	if !table.SetCellText("second", "name", "Second updated", 0, false) {
		t.Fatal("cell text did not change")
	}
	u.HandleMouse(MouseEvent{Pos: pos})
	if len(requests) != 3 || requests[2] != "second:name" {
		t.Fatalf("revision tooltip requests=%v", requests)
	}
}

// TestTableLayoutCacheIsRemovedWithWidget checks cache lifetime ownership.
func TestTableLayoutCacheIsRemovedWithWidget(t *testing.T) {
	u, table := tableUIFixture(t, core.Rect{X: 20, Y: 20, W: 260, H: 150})
	_ = tableUIHeaderCenter(u, table, 0)
	if len(u.tableLayouts) != 1 {
		t.Fatalf("table cache count before removal=%d", len(u.tableLayouts))
	}
	if !u.Remove("table") {
		t.Fatal("table removal failed")
	}
	if len(u.tableLayouts) != 0 {
		t.Fatalf("table cache retained after removal: %d", len(u.tableLayouts))
	}
}
