package render

import (
	"strconv"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
	"github.com/draxxris/rtgui/widgets"
)

// BenchmarkDrawWidgetPart measures the normal no-recorder draw path.
func BenchmarkDrawWidgetPart(b *testing.B) {
	theme := NewTheme(transform.New(core.Viewport{Viewport: core.Rect{W: 800, H: 600}}))
	theme.SetDrawRecorder(nil)
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		AtlasRegion: core.Rect{W: 64, H: 32}, Tint: core.Color{R: 255, G: 255, B: 255, A: 255}, HasTexture: true,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 100, H: 30}, core.StateNormal)
	}
}

// BenchmarkNinePatch measures deterministic nine-patch geometry.
func BenchmarkNinePatch(b *testing.B) {
	config := NinePatchConfig{Left: 8, Top: 8, Right: 8, Bottom: 8}
	destination := core.Rect{W: 120, H: 40}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NinePatchRects(config, destination)
	}
}

// benchmarkTableFixture builds and warms a representative 1,000-row table.
// Setup, initial column resolution, and the first sort stay outside measured
// iterations so the benchmarks describe steady-state draw, hover, and sort
// behavior rather than construction costs.
func benchmarkTableFixture(tb testing.TB) (*Theme, *widgets.Table) {
	tb.Helper()
	theme := NewTheme(transform.New(core.Viewport{Viewport: core.Rect{W: 800, H: 600}}))
	theme.SetDrawRecorder(nil)
	table := widgets.NewTable("benchmarkTable", core.Rect{W: 600, H: 240})
	if !table.SetColumns([]widgets.TableColumn{
		{ID: "name", Title: "Name", Width: 3},
		{ID: "level", Title: "Lvl", Width: 1, Numeric: true, Sortable: true},
		{ID: "buyout", Title: "Buyout", Width: 2, Numeric: true, Sortable: true},
	}) {
		tb.Fatal("benchmark table columns rejected")
	}
	rows := make([]widgets.TableRow, 1000)
	for index := range rows {
		level := float64(index % 80)
		buyout := float64((index*7919)%1000000 + index)
		id := "row-" + strconv.Itoa(index)
		rows[index] = widgets.TableRow{ID: id, Cells: []widgets.TableCell{
			{Text: "Auction item " + strconv.Itoa(index)},
			{Text: strconv.Itoa(int(level)), SortValue: level, HasSortValue: true},
			{Text: strconv.Itoa(int(buyout)) + "c", SortValue: buyout, HasSortValue: true},
		}}
	}
	if !table.SetRows(rows) {
		tb.Fatal("benchmark table rows rejected")
	}
	table.SetSort("buyout", core.SortAsc)
	table.EnsureScrollBounds(240)
	info := table.Snapshot(core.StateNormal)
	theme.DrawTableWithState(info, table, TableDrawState{HoveredRow: "row-517", ColumnSlots: benchmarkTableSlots(theme, table)})
	return theme, table
}

// benchmarkTableSlots resolves the caller-owned slots used by table draws.
func benchmarkTableSlots(theme *Theme, table *widgets.Table) []core.Rect {
	info := table.Snapshot(core.StateNormal)
	content := theme.TableContent(info.Bounds, info.State, info.Class)
	rows := TableRowsContent(content, table.MaxScroll())
	return TableLayoutColumns(rows.W, table.Columns())
}

// BenchmarkTableDraw1000 measures a warmed virtual table draw with 1,000 rows.
func BenchmarkTableDraw1000(b *testing.B) {
	theme, table := benchmarkTableFixture(b)
	info := table.Snapshot(core.StateNormal)
	drawState := TableDrawState{PressedRow: "row-17", ColumnSlots: benchmarkTableSlots(theme, table)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		theme.DrawTableWithState(info, table, drawState)
	}
}

// BenchmarkTableHover1000 measures warmed row hit resolution plus a draw for a
// moving pointer over a 1,000-row virtual table.
func BenchmarkTableHover1000(b *testing.B) {
	theme, table := benchmarkTableFixture(b)
	info := table.Snapshot(core.StateNormal)
	content := theme.TableContent(info.Bounds, info.State, info.Class)
	slots := benchmarkTableSlots(theme, table)
	pos := core.Vec2{X: content.X + 40, Y: content.Y + table.HeaderHeight() + 17*table.RowHeight() + 4}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index, header := TableRowAt(content, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset(), pos)
		if header {
			continue
		}
		rowID, _ := table.ViewRowIDAt(index)
		theme.DrawTableWithState(info, table, TableDrawState{HoveredRow: rowID, ColumnSlots: slots})
	}
}

// BenchmarkTableSort1000 measures alternating warmed numeric sort-view
// rebuilds without replacing the table's insertion-order rows.
func BenchmarkTableSort1000(b *testing.B) {
	_, table := benchmarkTableFixture(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		direction := core.SortAsc
		if i&1 == 1 {
			direction = core.SortDesc
		}
		table.SetSort("buyout", direction)
	}
}

// TestTableWarmedDrawHoverAllocations protects the steady-state virtual draw
// and pointer path from rebuilding 1,000-row or column-layout storage.
func TestTableWarmedDrawHoverAllocations(t *testing.T) {
	theme, table := benchmarkTableFixture(t)
	info := table.Snapshot(core.StateNormal)
	slots := benchmarkTableSlots(theme, table)
	draws := testing.AllocsPerRun(100, func() {
		theme.DrawTableWithState(info, table, TableDrawState{HoveredRow: "row-517", ColumnSlots: slots})
	})
	if draws != 0 {
		t.Fatalf("warmed table draw allocations = %g, want 0", draws)
	}
	content := theme.TableContent(info.Bounds, info.State, info.Class)
	pos := core.Vec2{X: content.X + 40, Y: content.Y + table.HeaderHeight() + 17*table.RowHeight() + 4}
	hover := testing.AllocsPerRun(100, func() {
		index, _ := TableRowAt(content, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset(), pos)
		rowID, _ := table.ViewRowIDAt(index)
		theme.DrawTableWithState(info, table, TableDrawState{HoveredRow: rowID, ColumnSlots: slots})
	})
	if hover != 0 {
		t.Fatalf("warmed table hover allocations = %g, want 0", hover)
	}
}
