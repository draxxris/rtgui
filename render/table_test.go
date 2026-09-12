package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
	"github.com/draxxris/rtgui/widgets"
)

// tableRenderFixture builds a small table with an active stable sort.
func tableRenderFixture(t *testing.T) *widgets.Table {
	t.Helper()
	table := widgets.NewTable("table", core.Rect{X: 10, Y: 20, W: 240, H: 120})
	if !table.SetColumns([]widgets.TableColumn{
		{ID: "name", Title: "Name", Width: 2},
		{ID: "amount", Title: "Amount", Width: 1, Sortable: true, Numeric: true},
	}) || !table.SetRows([]widgets.TableRow{
		{ID: "one", Cells: []widgets.TableCell{{Text: "One"}, {Text: "2", SortValue: 2, HasSortValue: true}}},
		{ID: "two", Cells: []widgets.TableCell{{Text: "Two"}, {Text: "1", SortValue: 1, HasSortValue: true}}},
		{ID: "three", Cells: []widgets.TableCell{{Text: "Three"}, {Text: "3", SortValue: 3, HasSortValue: true}}},
	}) || !table.SetSort("amount", core.SortAsc) || !table.Select("two") {
		t.Fatal("table fixture rejected")
	}
	return table
}

// TestTableAndListShareFixedRowsGeometry checks one track/content contract.
func TestTableAndListShareFixedRowsGeometry(t *testing.T) {
	content := core.Rect{X: 10, Y: 20, W: 200, H: 100}
	for _, maxScroll := range []float32{0, 40} {
		list := ListRowsContent(content, maxScroll)
		table := TableRowsContent(content, maxScroll)
		if list != table {
			t.Fatalf("max=%v list rows=%+v table rows=%+v", maxScroll, list, table)
		}
	}
	bounds := core.Rect{X: 10, Y: 20, W: 200, H: 100}
	theme := NewTheme(transform.New(core.Viewport{}))
	if got, want := theme.ListContent(bounds, core.StateNormal), theme.TableContent(bounds, core.StateNormal); got != want {
		t.Fatalf("unskinned shell parity = %+v/%+v", got, want)
	}
}

// TestTableDrawKeepsSpecificHeaderAndSharedRowDecoration checks draw ownership.
func TestTableDrawKeepsSpecificHeaderAndSharedRowDecoration(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	table := tableRenderFixture(t)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTable, Part: skin.PartHeader, State: core.StateNormal}, skin.SkinDescriptor{
		HasBackgroundColor: true, BackgroundColor: core.Color{R: 30, G: 38, B: 54, A: 255},
	})
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTable, Part: skin.PartStripe, State: core.StateNormal}, skin.SkinDescriptor{
		HasBackgroundColor: true, BackgroundColor: core.Color{R: 32, G: 36, B: 44, A: 255},
	})
	recorder := newTestRecorder(t, 128)
	theme.SetDrawRecorder(recorder)
	info := table.Snapshot(core.StateNormal)
	content := theme.TableContent(info.Bounds, info.State, info.Class)
	slots := TableLayoutColumns(TableRowsContent(content, table.MaxScroll()).W, table.Columns())
	theme.BeginFrame()
	theme.DrawTableWithState(info, table, TableDrawState{ColumnSlots: slots})
	calls := recorder.Calls()
	if !hasRenderPart(calls, core.WidgetTable, skin.PartHeader) || !hasRenderPart(calls, core.WidgetTable, skin.PartStripe) || !hasRenderPart(calls, core.WidgetTable, skin.PartArrow) {
		t.Fatalf("table-specific parts missing: %v", calls)
	}
	if !hasRenderPartState(calls, core.WidgetTable, skin.PartOverlay, core.StateSelected) {
		t.Fatal("table did not use shared selected-row decoration")
	}
}

// hasRenderPart reports whether a recorded widget part exists.
func hasRenderPart(calls []DrawCall, kind core.WidgetKind, part skin.SkinPart) bool {
	for _, call := range calls {
		if call.Kind == kind && call.Part == part {
			return true
		}
	}
	return false
}

// hasRenderPartState reports whether a recorded part has the requested state.
func hasRenderPartState(calls []DrawCall, kind core.WidgetKind, part skin.SkinPart, state core.WidgetState) bool {
	for _, call := range calls {
		if call.Kind == kind && call.Part == part && call.State == state {
			return true
		}
	}
	return false
}
