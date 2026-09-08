package widgets_test

import (
	"math"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// lineGraphFixture returns a two-series graph with known data.
func lineGraphFixture(t *testing.T) *widgets.LineGraph {
	t.Helper()
	graph := widgets.NewLineGraph("graph", core.Rect{X: 10, Y: 10, W: 200, H: 120})
	first := graph.AddSeries(core.Color{R: 255, A: 255}, 2)
	second := graph.AddSeries(core.Color{B: 255, A: 255}, 3)
	if first != 0 || second != 1 {
		t.Fatalf("series indices = %d/%d", first, second)
	}
	if !graph.SetSeriesData(0, []core.Vec2{{X: 0, Y: 0}, {X: 10, Y: 10}, {X: 20, Y: 5}}) {
		t.Fatal("SetSeriesData(0) failed")
	}
	if !graph.SetSeriesData(1, []core.Vec2{{X: 0, Y: 5}, {X: 20, Y: 15}}) {
		t.Fatal("SetSeriesData(1) failed")
	}
	return graph
}

// TestLineGraphDefaults verifies constructor state and temporary kind tagging.
func TestLineGraphDefaults(t *testing.T) {
	graph := widgets.NewLineGraph("graph", core.Rect{W: 200, H: 120})
	if graph.Kind() != widgets.LineGraphKind {
		t.Fatalf("kind = %v", graph.Kind())
	}
	if graph.SeriesCount() != 0 || graph.MaxPoints() != 1024 {
		t.Fatalf("count=%d max=%d", graph.SeriesCount(), graph.MaxPoints())
	}
	if graph.Interpolation() != widgets.LineGraphLinear {
		t.Fatalf("interp = %v", graph.Interpolation())
	}
	if !graph.ShowGrid() || graph.ShowLabels() {
		t.Fatal("expected grid on and labels off")
	}
	if x, y := graph.GridDivisions(); x != 4 || y != 4 {
		t.Fatalf("divisions = %d/%d", x, y)
	}
	if graph.HasXRange() || graph.HasYRange() {
		t.Fatal("ranges must start automatic")
	}
	if _, _, _, _, ok := graph.EffectiveRange(); ok {
		t.Fatal("empty graph must not resolve a range")
	}
}

// TestLineGraphSeriesCopiesInput verifies writes copy and reads hide storage.
func TestLineGraphSeriesCopiesInput(t *testing.T) {
	graph := widgets.NewLineGraph("graph", core.Rect{W: 100, H: 60})
	input := []core.Vec2{{X: 1, Y: 2}, {X: 3, Y: 4}}
	if graph.AddSeries(core.Color{R: 9, G: 9, B: 9, A: 255}, 0) != 0 {
		t.Fatal("first series index must be 0")
	}
	if !graph.SetSeriesData(0, input) {
		t.Fatal("SetSeriesData failed")
	}
	input[0] = core.Vec2{X: 99, Y: 99}
	if got, _ := graph.SeriesPointAt(0, 0); got != (core.Vec2{X: 1, Y: 2}) {
		t.Fatalf("input mutation leaked into storage: %v", got)
	}
	if thickness, _ := graph.SeriesThickness(0); thickness != 2 {
		t.Fatalf("non-positive thickness must default, got %v", thickness)
	}
	if graph.SetSeriesData(9, input) || graph.AppendPoint(9, core.Vec2{}) {
		t.Fatal("out-of-range writes must fail")
	}
	if _, ok := graph.SeriesPointAt(0, 42); ok {
		t.Fatal("out-of-range read must fail")
	}
}

// TestLineGraphBoundedData verifies capacity keeps the newest points.
func TestLineGraphBoundedData(t *testing.T) {
	graph := widgets.NewLineGraph("graph", core.Rect{W: 100, H: 60})
	graph.AddSeries(core.Color{A: 255}, 2)
	graph.SetMaxPoints(3)
	for i := float32(0); i < 5; i++ {
		if !graph.AppendPoint(0, core.Vec2{X: i, Y: i}) {
			t.Fatalf("AppendPoint(%v) failed", i)
		}
	}
	if got := graph.SeriesLen(0); got != 3 {
		t.Fatalf("bounded length = %d", got)
	}
	if got, _ := graph.SeriesPointAt(0, 0); got.X != 2 {
		t.Fatalf("oldest kept = %v", got)
	}
	graph.SetMaxPoints(0)
	if !graph.SetSeriesData(0, []core.Vec2{{X: 1, Y: 1}, {X: 2, Y: 2}, {X: 3, Y: 3}, {X: 4, Y: 4}}) {
		t.Fatal("unbounded SetSeriesData failed")
	}
	if got := graph.SeriesLen(0); got != 4 {
		t.Fatalf("unbounded length = %d", got)
	}
	if !graph.ClearSeriesData(0) || graph.SeriesLen(0) != 0 {
		t.Fatal("ClearSeriesData must empty the series")
	}
	if !graph.RemoveSeries(0) || graph.SeriesCount() != 0 {
		t.Fatal("RemoveSeries must drop the series")
	}
}

// TestLineGraphExplicitRanges verifies fixed ranges apply and read back.
func TestLineGraphExplicitRanges(t *testing.T) {
	graph := lineGraphFixture(t)
	if !graph.SetXRange(0, 20) {
		t.Fatal("valid x range must apply")
	}
	if !graph.SetYRange(0, 15) {
		t.Fatal("valid y range must apply")
	}
	xMin, xMax, ok := graph.XRange()
	if !ok {
		t.Fatal("x range must be set")
	}
	if xMin != 0 || xMax != 20 {
		t.Fatalf("x range = %v/%v", xMin, xMax)
	}
	yMin, yMax, ok := graph.YRange()
	if !ok {
		t.Fatal("y range must be set")
	}
	if yMin != 0 || yMax != 15 {
		t.Fatalf("y range = %v/%v", yMin, yMax)
	}
}

// TestLineGraphRangeValidation verifies degenerate ranges are rejected.
func TestLineGraphRangeValidation(t *testing.T) {
	graph := lineGraphFixture(t)
	nan := float32(math.NaN())
	if graph.SetXRange(5, 5) {
		t.Fatal("equal bounds must be rejected")
	}
	if graph.SetXRange(9, 1) {
		t.Fatal("inverted bounds must be rejected")
	}
	if graph.SetXRange(nan, 1) {
		t.Fatal("NaN bound must be rejected")
	}
	if graph.SetYRange(0, nan) {
		t.Fatal("NaN bound must be rejected")
	}
	if graph.HasXRange() || graph.HasYRange() {
		t.Fatal("rejected ranges must leave automatic mode")
	}
}

// TestLineGraphRangeClear verifies returning axes to automatic ranging.
func TestLineGraphRangeClear(t *testing.T) {
	graph := lineGraphFixture(t)
	graph.SetXRange(0, 20)
	graph.SetYRange(0, 15)
	if !graph.ClearXRange() {
		t.Fatal("clearing x must report a change")
	}
	if !graph.ClearYRange() {
		t.Fatal("clearing y must report a change")
	}
	if graph.HasXRange() || graph.HasYRange() {
		t.Fatal("clearing must restore automatic ranges")
	}
	if graph.ClearXRange() {
		t.Fatal("clearing an automatic axis must report no change")
	}
	if graph.ClearYRange() {
		t.Fatal("clearing an automatic axis must report no change")
	}
}

// autoRangeGraph returns a single-series graph holding points.
func autoRangeGraph(points []core.Vec2) *widgets.LineGraph {
	graph := widgets.NewLineGraph("graph", core.Rect{W: 100, H: 60})
	graph.AddSeries(core.Color{A: 255}, 2)
	graph.SetSeriesData(0, points)
	return graph
}

// TestLineGraphAutoRangeSkipsNonfinite verifies scans ignore NaN and Inf.
func TestLineGraphAutoRangeSkipsNonfinite(t *testing.T) {
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))
	graph := autoRangeGraph([]core.Vec2{{X: nan, Y: 1}, {X: 4, Y: inf}, {X: 2, Y: 6}, {X: 8, Y: 10}})
	xMin, xMax, yMin, yMax, ok := graph.EffectiveRange()
	if !ok {
		t.Fatal("finite data must resolve a range")
	}
	if xMin != 2 || xMax != 8 {
		t.Fatalf("auto x = %v/%v", xMin, xMax)
	}
	if yMin != 6 || yMax != 10 {
		t.Fatalf("auto y = %v/%v", yMin, yMax)
	}
}

// TestLineGraphAutoRangeEmpty verifies all-non-finite data resolves nothing.
func TestLineGraphAutoRangeEmpty(t *testing.T) {
	nan := float32(math.NaN())
	graph := autoRangeGraph([]core.Vec2{{X: nan, Y: nan}})
	if _, _, _, _, ok := graph.EffectiveRange(); ok {
		t.Fatal("all-non-finite data must not resolve a range")
	}
}

// TestLineGraphAutoRangeWidensConstant verifies singleton data still maps.
func TestLineGraphAutoRangeWidensConstant(t *testing.T) {
	graph := autoRangeGraph([]core.Vec2{{X: 5, Y: 5}})
	xMin, xMax, yMin, yMax, ok := graph.EffectiveRange()
	if !ok {
		t.Fatal("constant data must resolve a range")
	}
	if xMin >= 5 || xMax <= 5 {
		t.Fatalf("constant x = %v/%v", xMin, xMax)
	}
	if yMin >= 5 || yMax <= 5 {
		t.Fatalf("constant y = %v/%v", yMin, yMax)
	}
}

// TestLineGraphAutoRangeMixed verifies explicit x pairs with scanned y.
func TestLineGraphAutoRangeMixed(t *testing.T) {
	graph := autoRangeGraph([]core.Vec2{{X: 5, Y: 5}})
	graph.SetXRange(0, 10)
	xMin, xMax, yMin, yMax, ok := graph.EffectiveRange()
	if !ok {
		t.Fatal("mixed ranges must resolve")
	}
	if xMin != 0 || xMax != 10 {
		t.Fatalf("explicit x = %v/%v", xMin, xMax)
	}
	if yMin >= 5 || yMax <= 5 {
		t.Fatalf("scanned y = %v/%v", yMin, yMax)
	}
}

// TestLineGraphGeometryMapping verifies plot mapping, ticks, and labels.
func TestLineGraphGeometryMapping(t *testing.T) {
	graph := lineGraphFixture(t)
	graph.SetXRange(0, 20)
	graph.SetYRange(0, 20)
	plot := graph.PlotRect(core.Rect{W: 200, H: 120})
	graph.EnsureGeometry(plot)
	if got := graph.MappedLen(0); got != 3 {
		t.Fatalf("mapped length = %d", got)
	}
	first, ok := graph.MappedPointAt(0, 0)
	if !ok {
		t.Fatal("mapped point missing")
	}
	if first.X != plot.X || first.Y != plot.Y+plot.H {
		t.Fatalf("origin mapped = %v plot = %v", first, plot)
	}
	last, ok := graph.MappedPointAt(0, 1)
	if !ok || last.X <= first.X || last.Y >= first.Y {
		t.Fatalf("rising point mapped = %v from %v", last, first)
	}
	if got := graph.XTickCount(); got != 5 {
		t.Fatalf("x ticks = %d", got)
	}
	value, pixel, label, ok := graph.XTickAt(0)
	if !ok || value != 0 || pixel != plot.X || label == "" {
		t.Fatalf("first x tick = %v/%v/%q/%v", value, pixel, label, ok)
	}
	if _, _, _, ok := graph.XTickAt(99); ok {
		t.Fatal("out-of-range tick must fail")
	}
	if yCount := graph.YTickCount(); yCount != 5 {
		t.Fatalf("y ticks = %d", yCount)
	}
}

// TestLineGraphNonfiniteGaps verifies NaN points break cached runs.
func TestLineGraphNonfiniteGaps(t *testing.T) {
	graph := widgets.NewLineGraph("graph", core.Rect{W: 100, H: 60})
	graph.AddSeries(core.Color{A: 255}, 2)
	nan := float32(math.NaN())
	graph.SetSeriesData(0, []core.Vec2{{X: 0, Y: 0}, {X: nan, Y: nan}, {X: 10, Y: 10}})
	plot := graph.PlotRect(graph.Bounds())
	graph.EnsureGeometry(plot)
	if _, ok := graph.MappedPointAt(0, 1); ok {
		t.Fatal("non-finite point must not map")
	}
	if _, ok := graph.MappedPointAt(0, 0); !ok {
		t.Fatal("finite neighbors must still map")
	}
	if _, _, _, _, ok := graph.NearestPoint(plot, core.Vec2{X: plot.X, Y: plot.Y + plot.H}); !ok {
		t.Fatal("nearest query must skip the gap")
	}
}

// TestLineGraphNearestPoint verifies closest stored point resolution.
func TestLineGraphNearestPoint(t *testing.T) {
	graph := lineGraphFixture(t)
	plot := graph.PlotRect(core.Rect{W: 200, H: 120})
	series, index, data, mapped, ok := graph.NearestPoint(plot, core.Vec2{X: plot.X + plot.W, Y: plot.Y})
	if !ok || series != 1 || index != 1 {
		t.Fatalf("nearest = series %d index %d ok=%v", series, index, ok)
	}
	if data != (core.Vec2{X: 20, Y: 15}) {
		t.Fatalf("nearest data = %v", data)
	}
	want, mapOK := graph.MappedPointAt(1, 1)
	if !mapOK || mapped != want {
		t.Fatalf("nearest mapped = %v want %v", mapped, want)
	}
	empty := widgets.NewLineGraph("empty", core.Rect{W: 50, H: 50})
	if _, _, _, _, ok := empty.NearestPoint(empty.PlotRect(empty.Bounds()), core.Vec2{}); ok {
		t.Fatal("empty graph must not resolve a nearest point")
	}
}

// TestLineGraphInterpolationAndGrid verifies modes and grid configuration.
func TestLineGraphInterpolationAndGrid(t *testing.T) {
	graph := lineGraphFixture(t)
	graph.SetInterpolation(widgets.LineGraphStep)
	if graph.Interpolation() != widgets.LineGraphStep {
		t.Fatal("step mode must stick")
	}
	plot := graph.PlotRect(core.Rect{W: 200, H: 120})
	graph.EnsureGeometry(plot)
	linear := widgets.NewLineGraph("linear", core.Rect{W: 200, H: 120})
	linear.AddSeries(core.Color{A: 255}, 2)
	linear.SetSeriesData(0, []core.Vec2{{X: 0, Y: 0}, {X: 10, Y: 10}})
	linear.EnsureGeometry(plot)
	stepPoint, _ := graph.MappedPointAt(0, 0)
	_ = stepPoint
	graph.SetInterpolation(999)
	if graph.Interpolation() != widgets.LineGraphStep {
		t.Fatal("invalid interpolation must be ignored")
	}
	graph.SetGridDivisions(0, 2)
	graph.EnsureGeometry(plot)
	if got := graph.XTickCount(); got != 0 {
		t.Fatalf("disabled x grid ticks = %d", got)
	}
	if got := graph.YTickCount(); got != 3 {
		t.Fatalf("y grid ticks = %d", got)
	}
	graph.SetGridDivisions(99, -3)
	if x, y := graph.GridDivisions(); x != 16 || y != 0 {
		t.Fatalf("clamped divisions = %d/%d", x, y)
	}
	if !graph.SetShowLabels(true).ShowLabels() {
		t.Fatal("labels must enable")
	}
	bare := graph.PlotRect(core.Rect{W: 200, H: 120})
	labeled := graph.SetShowLabels(true).PlotRect(core.Rect{W: 200, H: 120})
	_ = bare
	if labeled.W >= 200-8 || labeled.H >= 120-8 {
		t.Fatalf("labeled plot must reserve gutters: %v", labeled)
	}
}

// TestLineGraphStyleAndSingleton verifies style updates and single-point ranges.
func TestLineGraphStyleAndSingleton(t *testing.T) {
	graph := widgets.NewLineGraph("graph", core.Rect{W: 100, H: 60})
	graph.AddSeries(core.Color{}, 0)
	color, ok := graph.SeriesColor(0)
	if !ok || color == (core.Color{}) {
		t.Fatalf("zero color must select a palette entry: %v", color)
	}
	if !graph.SetSeriesColor(0, core.Color{R: 1, G: 2, B: 3, A: 255}) {
		t.Fatal("SetSeriesColor failed")
	}
	if got, _ := graph.SeriesColor(0); got.R != 1 {
		t.Fatalf("series color = %v", got)
	}
	if graph.SetSeriesColor(9, color) || graph.SetSeriesThickness(9, 2) {
		t.Fatal("out-of-range style writes must fail")
	}
	graph.SetSeriesData(0, []core.Vec2{{X: 7, Y: 7}})
	plot := graph.PlotRect(graph.Bounds())
	graph.EnsureGeometry(plot)
	if _, ok := graph.MappedPointAt(0, 0); !ok {
		t.Fatal("singleton must map after constant expansion")
	}
	series, index, _, _, ok := graph.NearestPoint(plot, core.Vec2{})
	if !ok || series != 0 || index != 0 {
		t.Fatal("singleton must resolve its only point")
	}
}

// TestLineGraphGeometryAvoidsSteadyStateAllocs verifies the hot read path.
func TestLineGraphGeometryAvoidsSteadyStateAllocs(t *testing.T) {
	graph := lineGraphFixture(t)
	plot := graph.PlotRect(core.Rect{W: 200, H: 120})
	graph.EnsureGeometry(plot)
	allocs := testing.AllocsPerRun(200, func() {
		graph.EnsureGeometry(plot)
		_ = graph.MappedLen(0)
		_, _ = graph.MappedPointAt(0, 1)
		_, _, _, _ = graph.XTickAt(0)
		_, _, _, _, _ = graph.NearestPoint(plot, core.Vec2{X: plot.X + 5, Y: plot.Y + 5})
	})
	if allocs != 0 {
		t.Fatalf("steady-state geometry allocated %v times", allocs)
	}
}

// TestLineGraphNilSafety verifies nil receivers never panic.
func TestLineGraphNilSafety(t *testing.T) {
	var graph *widgets.LineGraph
	graph.EnsureGeometry(core.Rect{W: 10, H: 10})
	if graph.SeriesCount() != 0 || graph.SeriesLen(0) != 0 {
		t.Fatal("nil counts must be zero")
	}
	if _, ok := graph.SeriesPointAt(0, 0); ok {
		t.Fatal("nil reads must fail")
	}
	if graph.AddSeries(core.Color{}, 1) != -1 || graph.SetSeriesData(0, nil) || graph.AppendPoint(0, core.Vec2{}) {
		t.Fatal("nil writes must fail")
	}
	if graph.SetXRange(0, 1) || graph.ClearXRange() || graph.SetYRange(0, 1) || graph.ClearYRange() {
		t.Fatal("nil range writes must fail")
	}
	if _, _, _, _, ok := graph.EffectiveRange(); ok {
		t.Fatal("nil range must not resolve")
	}
	if _, _, _, _, ok := graph.NearestPoint(core.Rect{}, core.Vec2{}); ok {
		t.Fatal("nil nearest query must fail")
	}
	if graph.PlotRect(core.Rect{W: -5, H: -5}) != (core.Rect{X: 4, Y: 4, W: 0, H: 0}) {
		t.Fatalf("nil plot rect = %v", graph.PlotRect(core.Rect{W: -5, H: -5}))
	}
}

// TestLineGraphNilChainedSetters verifies fluent setters stay nil-safe.
func TestLineGraphNilChainedSetters(t *testing.T) {
	var graph *widgets.LineGraph
	if graph.SetTooltip("tip") != nil {
		t.Fatal("nil SetTooltip must return nil")
	}
	if graph.SetSeriesVisible(0, false) {
		t.Fatal("nil visibility write must fail")
	}
	if graph.AppendPoints(0, []core.Vec2{{X: 1, Y: 1}}) {
		t.Fatal("nil bulk append must fail")
	}
	if got := graph.AppendSeriesPoints(0, nil); len(got) != 0 {
		t.Fatal("nil bulk read must stay empty")
	}
	if graph.SetTickFormatter(nil) != nil {
		t.Fatal("nil formatter setter must return nil")
	}
	if _, _, _, _, ok := graph.NearestPointAt(core.Vec2{}); ok {
		t.Fatal("nil convenience query must fail")
	}
}

// TestLineGraphKindMatchesCore verifies snapshots match registered lookups.
func TestLineGraphKindMatchesCore(t *testing.T) {
	if widgets.LineGraphKind != core.WidgetLineGraph {
		t.Fatalf("kind = %v, want %v", widgets.LineGraphKind, core.WidgetLineGraph)
	}
	graph := widgets.NewLineGraph("graph", core.Rect{W: 10, H: 10})
	if graph.Kind() != core.WidgetLineGraph {
		t.Fatalf("widget kind = %v", graph.Kind())
	}
}

// TestAsLineGraph verifies the widget assertion helper.
func TestAsLineGraph(t *testing.T) {
	graph := widgets.NewLineGraph("graph", core.Rect{})
	if _, err := widgets.AsLineGraph(graph); err != nil {
		t.Fatalf("AsLineGraph(graph) = %v", err)
	}
	button := widgets.NewButton("ok", core.Rect{}, "OK")
	if _, err := widgets.AsLineGraph(button); err == nil {
		t.Fatal("AsLineGraph(button) must fail")
	}
	if _, err := widgets.AsLineGraph(nil); err == nil {
		t.Fatal("AsLineGraph(nil) must fail")
	}
}

// TestLineGraphAppendPoints verifies batched writes copy and bound correctly.
func TestLineGraphAppendPoints(t *testing.T) {
	graph := widgets.NewLineGraph("graph", core.Rect{W: 100, H: 60})
	graph.AddSeries(core.Color{A: 255}, 2)
	input := []core.Vec2{{X: 1, Y: 1}, {X: 2, Y: 2}}
	if !graph.AppendPoints(0, input) {
		t.Fatal("AppendPoints failed")
	}
	input[0] = core.Vec2{X: 99, Y: 99}
	if got, _ := graph.SeriesPointAt(0, 0); got != (core.Vec2{X: 1, Y: 1}) {
		t.Fatalf("input mutation leaked: %v", got)
	}
	graph.SetMaxPoints(3)
	batch := []core.Vec2{{X: 3, Y: 3}, {X: 4, Y: 4}, {X: 5, Y: 5}, {X: 6, Y: 6}}
	if !graph.AppendPoints(0, batch) {
		t.Fatal("bounded AppendPoints failed")
	}
	if got := graph.SeriesLen(0); got != 3 {
		t.Fatalf("bounded length = %d", got)
	}
	if got, _ := graph.SeriesPointAt(0, 0); got.X != 4 {
		t.Fatalf("oldest kept = %v", got)
	}
	if graph.AppendPoints(9, batch) {
		t.Fatal("out-of-range bulk append must fail")
	}
}

// TestLineGraphSeriesVisibility verifies hiding skips draws and queries.
func TestLineGraphSeriesVisibility(t *testing.T) {
	graph := lineGraphFixture(t)
	if !graph.SeriesVisible(0) {
		t.Fatal("series must start visible")
	}
	if !graph.SetSeriesVisible(0, false) {
		t.Fatal("SetSeriesVisible failed")
	}
	if graph.SeriesVisible(0) {
		t.Fatal("series must hide")
	}
	plot := graph.PlotRect(core.Rect{W: 200, H: 120})
	series, _, _, _, ok := graph.NearestPoint(plot, core.Vec2{X: plot.X, Y: plot.Y})
	if !ok {
		t.Fatal("query must fall through to visible series")
	}
	if series != 1 {
		t.Fatalf("nearest visible series = %d", series)
	}
	if graph.SeriesVisible(9) {
		t.Fatal("unknown series must report hidden")
	}
	if graph.SetSeriesVisible(9, true) {
		t.Fatal("unknown visibility write must fail")
	}
}

// TestLineGraphAppendSeriesPoints verifies the idiomatic bulk read.
func TestLineGraphAppendSeriesPoints(t *testing.T) {
	graph := lineGraphFixture(t)
	got := graph.AppendSeriesPoints(0, nil)
	if len(got) != 3 {
		t.Fatalf("bulk read length = %d", len(got))
	}
	if got[0] != (core.Vec2{X: 0, Y: 0}) {
		t.Fatalf("bulk read first = %v", got[0])
	}
	got[0] = core.Vec2{X: 99, Y: 99}
	if stored, _ := graph.SeriesPointAt(0, 0); stored != (core.Vec2{X: 0, Y: 0}) {
		t.Fatal("bulk read mutation leaked into storage")
	}
	dst := make([]core.Vec2, 0, 8)
	if out := graph.AppendSeriesPoints(9, dst); len(out) != 0 {
		t.Fatal("unknown series must return dst unchanged")
	}
}

// TestLineGraphTickFormatter verifies custom and reset label formatting.
func TestLineGraphTickFormatter(t *testing.T) {
	graph := lineGraphFixture(t)
	graph.SetTickFormatter(func(v float32) string { return "tick" })
	plot := graph.PlotRect(core.Rect{W: 200, H: 120})
	graph.EnsureGeometry(plot)
	if _, _, label, ok := graph.XTickAt(0); !ok || label != "tick" {
		t.Fatalf("custom label = %q/%v", label, ok)
	}
	graph.SetTickFormatter(nil)
	graph.EnsureGeometry(plot)
	if _, _, label, ok := graph.XTickAt(0); !ok || label == "tick" || label == "" {
		t.Fatalf("reset label = %q/%v", label, ok)
	}
}

// TestLineGraphStreamingAvoidsAllocs verifies per-frame appends with fixed
// ranges reuse geometry backing and skip label reformatting entirely.
func TestLineGraphStreamingAvoidsAllocs(t *testing.T) {
	graph := widgets.NewLineGraph("stream", core.Rect{W: 200, H: 120})
	graph.AddSeries(core.Color{A: 255}, 2)
	graph.SetMaxPoints(8)
	graph.SetXRange(0, 100)
	graph.SetYRange(0, 100)
	for i := float32(0); i < 20; i++ {
		graph.AppendPoint(0, core.Vec2{X: i, Y: i})
	}
	plot := graph.PlotRect(graph.Bounds())
	graph.EnsureGeometry(plot)
	allocs := testing.AllocsPerRun(200, func() {
		graph.AppendPoint(0, core.Vec2{X: 50, Y: 50})
		graph.EnsureGeometry(plot)
		_, _, _, _, _ = graph.NearestPoint(plot, core.Vec2{X: plot.X + 5, Y: plot.Y + 5})
	})
	if allocs != 0 {
		t.Fatalf("streaming frame allocated %v times", allocs)
	}
}

// fxFixture returns a single-series graph for FX tests.
func fxFixture(t *testing.T) *widgets.LineGraph {
	t.Helper()
	graph := widgets.NewLineGraph("graph", core.Rect{W: 100, H: 60})
	graph.AddSeries(core.Color{R: 120, G: 210, B: 130, A: 255}, 2)
	return graph
}

// TestLineGraphSeriesFXDefaults verifies fresh series stay flat and
// unknown indices fail both reads and writes.
func TestLineGraphSeriesFXDefaults(t *testing.T) {
	graph := fxFixture(t)
	if fx, ok := graph.SeriesFX(0); !ok || fx.FillEnabled || fx.GlowEnabled {
		t.Fatalf("fresh series FX must be disabled: %+v/%v", fx, ok)
	}
	if _, ok := graph.SeriesFX(9); ok {
		t.Fatal("unknown series FX read must fail")
	}
	if graph.SetSeriesFX(9, widgets.LineSeriesFX{FillEnabled: true}) {
		t.Fatal("unknown series FX write must fail")
	}
}

// TestLineGraphSeriesFXNormalization verifies zero alphas and widths fall
// back to visible defaults while explicit values stick.
func TestLineGraphSeriesFXNormalization(t *testing.T) {
	graph := fxFixture(t)
	if !graph.SetSeriesFX(0, widgets.LineSeriesFX{FillEnabled: true, GlowEnabled: true}) {
		t.Fatal("SetSeriesFX failed")
	}
	fx, ok := graph.SeriesFX(0)
	if !ok || !fx.FillEnabled || !fx.GlowEnabled {
		t.Fatalf("FX flags lost: %+v/%v", fx, ok)
	}
	if fx.FillTopAlpha != 96 || fx.GlowAlpha != 80 || fx.GlowWidth != 6 {
		t.Fatalf("zero alphas/widths must default: %+v", fx)
	}
	explicit := widgets.LineSeriesFX{
		FillEnabled: true, FillTopAlpha: 40,
		GlowEnabled: true, GlowAlpha: 60, GlowWidth: 3,
	}
	if !graph.SetSeriesFX(0, explicit) {
		t.Fatal("explicit SetSeriesFX failed")
	}
	if fx, _ := graph.SeriesFX(0); fx != explicit {
		t.Fatalf("explicit FX must stick: %+v", fx)
	}
}

// TestLineGraphSeriesFXDisableAndNil verifies clearing effects and nil safety.
func TestLineGraphSeriesFXDisableAndNil(t *testing.T) {
	graph := fxFixture(t)
	graph.SetSeriesFX(0, widgets.LineSeriesFX{FillEnabled: true, GlowEnabled: true})
	if !graph.SetSeriesFX(0, widgets.LineSeriesFX{}) {
		t.Fatal("disabling FX failed")
	}
	if fx, _ := graph.SeriesFX(0); fx.FillEnabled || fx.GlowEnabled {
		t.Fatalf("FX must disable: %+v", fx)
	}
	var nilGraph *widgets.LineGraph
	if _, ok := nilGraph.SeriesFX(0); ok {
		t.Fatal("nil FX read must fail")
	}
	if nilGraph.SetSeriesFX(0, widgets.LineSeriesFX{FillEnabled: true}) {
		t.Fatal("nil FX write must fail")
	}
}
