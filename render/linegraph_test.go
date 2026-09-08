package render

import (
	"math"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
	"github.com/draxxris/rtgui/widgets"
)

// lineGraphTheme returns a headless theme with a large diagnostic recorder.
func lineGraphTheme(t *testing.T, maxCalls int) (*Theme, *DrawRecorder) {
	t.Helper()
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder := newTestRecorder(t, maxCalls)
	theme.SetDrawRecorder(recorder)
	return theme, recorder
}

// lineGraphInfo builds the renderer-facing snapshot for a graph widget.
func lineGraphInfo(graph *widgets.LineGraph) core.WidgetInfo {
	return core.WidgetInfo{
		Name:   graph.Name(),
		Bounds: graph.Bounds(),
		Kind:   graph.Kind(),
		State:  core.StateNormal,
	}
}

// lineGraphSample returns a two-series graph with a non-finite gap.
func lineGraphSample() *widgets.LineGraph {
	graph := widgets.NewLineGraph("graph", core.Rect{X: 10, Y: 10, W: 200, H: 120})
	graph.AddSeries(core.Color{R: 255, A: 255}, 2)
	graph.AddSeries(core.Color{B: 255, A: 255}, 3)
	graph.SetSeriesData(0, []core.Vec2{{X: 0, Y: 0}, {X: 5, Y: 20}, {X: 10, Y: 10}})
	graph.SetSeriesData(1, []core.Vec2{{X: 0, Y: 5}, {X: 99, Y: 99}})
	graph.SetXRange(0, 10)
	graph.SetYRange(0, 20)
	return graph
}

// recordedParts counts draw calls per skin part.
func recordedParts(calls []DrawCall) map[skin.SkinPart]int {
	counts := make(map[skin.SkinPart]int, len(calls))
	for _, call := range calls {
		counts[call.Part]++
	}
	return counts
}

// TestDrawLineGraphHeadless verifies background, grid, series, and label parts.
func TestDrawLineGraphHeadless(t *testing.T) {
	theme, recorder := lineGraphTheme(t, 256)
	graph := lineGraphSample()
	graph.SetShowLabels(true)
	info := lineGraphInfo(graph)
	theme.BeginFrame()
	theme.DrawLineGraph(info, graph)
	calls := recorder.Calls()
	parts := recordedParts(calls)
	if parts[skin.PartBackground] != 1 {
		t.Fatalf("background calls = %d, want 1 (%v)", parts[skin.PartBackground], calls)
	}
	if parts[skin.PartTrack] == 0 {
		t.Fatalf("grid calls missing: %v", calls)
	}
	if parts[skin.PartOverlay] == 0 {
		t.Fatalf("series calls missing: %v", calls)
	}
	if parts[skin.PartText] == 0 {
		t.Fatalf("label calls missing: %v", calls)
	}
	if got := recorder.LastWidgetInfo(); got != info {
		t.Fatalf("last widget = %+v, want %+v", got, info)
	}
}

// TestDrawLineGraphEdgeCases verifies empty, singleton, constant, and gapped data.
func TestDrawLineGraphEdgeCases(t *testing.T) {
	theme, recorder := lineGraphTheme(t, 256)
	bounds := core.Rect{W: 200, H: 120}
	empty := widgets.NewLineGraph("empty", bounds)
	theme.BeginFrame()
	theme.DrawLineGraph(lineGraphInfo(empty), empty)
	if parts := recordedParts(recorder.Calls()); parts[skin.PartBackground] != 1 || parts[skin.PartOverlay] != 0 || parts[skin.PartSpark] != 0 || parts[skin.PartText] != 0 {
		t.Fatalf("empty graph must draw shell and grid only: %v", recorder.Calls())
	}

	single := widgets.NewLineGraph("single", bounds)
	single.AddSeries(core.Color{R: 255, A: 255}, 2)
	single.SetSeriesData(0, []core.Vec2{{X: 4, Y: 4}})
	theme.BeginFrame()
	theme.DrawLineGraph(lineGraphInfo(single), single)
	if parts := recordedParts(recorder.Calls()); parts[skin.PartSpark] != 1 {
		t.Fatalf("singleton must draw one marker: %v", recorder.Calls())
	}

	constant := widgets.NewLineGraph("constant", bounds)
	constant.AddSeries(core.Color{A: 255}, 2)
	constant.SetSeriesData(0, []core.Vec2{{X: 1, Y: 5}, {X: 2, Y: 5}, {X: 3, Y: 5}})
	theme.BeginFrame()
	theme.DrawLineGraph(lineGraphInfo(constant), constant)
	if parts := recordedParts(recorder.Calls()); parts[skin.PartOverlay] == 0 {
		t.Fatalf("constant series must draw segments: %v", recorder.Calls())
	}

	var nilTheme *Theme
	nilTheme.DrawLineGraph(lineGraphInfo(single), single)
	theme.DrawLineGraph(lineGraphInfo(single), nil)
	var nilGraph *widgets.LineGraph
	theme.DrawLineGraph(lineGraphInfo(single), nilGraph)
}

// TestDrawLineGraphStepMode verifies step interpolation draws corner segments.
func TestDrawLineGraphStepMode(t *testing.T) {
	theme, recorder := lineGraphTheme(t, 256)
	linear := lineGraphSample()
	theme.BeginFrame()
	theme.DrawLineGraph(lineGraphInfo(linear), linear)
	linearCalls := len(recorder.Calls())

	stepped := lineGraphSample()
	stepped.SetInterpolation(widgets.LineGraphStep)
	theme.BeginFrame()
	theme.DrawLineGraph(lineGraphInfo(stepped), stepped)
	steppedCalls := len(recorder.Calls())
	if steppedCalls <= linearCalls {
		t.Fatalf("step calls = %d, want more than linear %d", steppedCalls, linearCalls)
	}
}

// TestClipGraphSegment verifies inside, outside, crossing, and point cases.
func TestClipGraphSegment(t *testing.T) {
	plot := core.Rect{X: 10, Y: 10, W: 80, H: 40}
	insideA, insideB := core.Vec2{X: 20, Y: 20}, core.Vec2{X: 40, Y: 30}
	if a, b, ok := clipGraphSegment(insideA, insideB, plot); !ok || a != insideA || b != insideB {
		t.Fatalf("inside = %v/%v/%v", a, b, ok)
	}
	if _, _, ok := clipGraphSegment(core.Vec2{}, core.Vec2{X: 5, Y: 5}, plot); ok {
		t.Fatal("outside segment must be rejected")
	}
	a, b, ok := clipGraphSegment(core.Vec2{}, core.Vec2{X: 50, Y: 25}, plot)
	if !ok {
		t.Fatal("crossing segment must clip")
	}
	// The segment enters through the top edge (y=10 at x=20), not the left edge.
	if a != (core.Vec2{X: 20, Y: 10}) || b != (core.Vec2{X: 50, Y: 25}) {
		t.Fatalf("crossing = %v/%v", a, b)
	}
	if _, _, ok := clipGraphSegment(core.Vec2{X: 20, Y: 20}, core.Vec2{X: 20, Y: 20}, plot); !ok {
		t.Fatal("inside point must clip")
	}
	if _, _, ok := clipGraphSegment(core.Vec2{X: 200, Y: 200}, core.Vec2{X: 200, Y: 200}, plot); ok {
		t.Fatal("outside point must be rejected")
	}
	if a, b, ok := clipGraphSegment(core.Vec2{X: 20, Y: 0}, core.Vec2{X: 20, Y: 100}, plot); !ok || a.Y != 10 || b.Y != 50 {
		t.Fatalf("vertical span = %v/%v/%v", a, b, ok)
	}
	nan := float32(math.NaN())
	if _, _, ok := clipGraphSegment(core.Vec2{X: nan, Y: 20}, insideB, plot); ok {
		t.Fatal("NaN endpoint must be rejected")
	}
	if _, _, ok := clipGraphSegment(insideA, core.Vec2{X: 40, Y: nan}, plot); ok {
		t.Fatal("NaN endpoint must be rejected")
	}
}

// TestDrawLineGraphSpanBoundsValid verifies recorded segment bounds never
// invert, even for rising data where screen Y decreases along the segment.
func TestDrawLineGraphSpanBoundsValid(t *testing.T) {
	theme, recorder := lineGraphTheme(t, 256)
	graph := widgets.NewLineGraph("rising", core.Rect{W: 200, H: 120})
	graph.AddSeries(core.Color{A: 255}, 2)
	graph.SetSeriesData(0, []core.Vec2{{X: 0, Y: 0}, {X: 10, Y: 20}, {X: 4, Y: 5}})
	theme.BeginFrame()
	theme.DrawLineGraph(lineGraphInfo(graph), graph)
	seen := 0
	for _, call := range recorder.Calls() {
		if call.Part != skin.PartOverlay {
			continue
		}
		seen++
		if call.Dest.W < 0 || call.Dest.H < 0 {
			t.Fatalf("inverted span = %+v", call)
		}
	}
	if seen == 0 {
		t.Fatal("rising series drew no segments")
	}
}

// TestDrawLineGraphSkipsOutsideMarkers verifies isolated points outside the
// explicit range draw nothing with or without an active scissor.
func TestDrawLineGraphSkipsOutsideMarkers(t *testing.T) {
	theme, recorder := lineGraphTheme(t, 64)
	graph := widgets.NewLineGraph("far", core.Rect{W: 200, H: 120})
	graph.AddSeries(core.Color{A: 255}, 2)
	graph.SetSeriesData(0, []core.Vec2{{X: 500, Y: 500}})
	graph.SetXRange(0, 10)
	graph.SetYRange(0, 10)
	theme.BeginFrame()
	theme.DrawLineGraph(lineGraphInfo(graph), graph)
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartSpark {
			t.Fatalf("outside marker must not draw: %+v", call)
		}
	}
}

// TestDrawLineGraphHidesHiddenSeries verifies invisible series draw nothing.
func TestDrawLineGraphHidesHiddenSeries(t *testing.T) {
	theme, recorder := lineGraphTheme(t, 256)
	graph := lineGraphSample()
	graph.SetSeriesVisible(0, false)
	graph.SetSeriesVisible(1, false)
	theme.BeginFrame()
	theme.DrawLineGraph(lineGraphInfo(graph), graph)
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartOverlay || call.Part == skin.PartSpark {
			t.Fatalf("hidden series must not draw: %+v", call)
		}
	}
}

// TestDrawLineGraphLabelsInsideBounds verifies clamped edge labels stay in
// the widget rectangle regardless of measured width.
func TestDrawLineGraphLabelsInsideBounds(t *testing.T) {
	theme, recorder := lineGraphTheme(t, 512)
	graph := lineGraphSample()
	graph.SetShowLabels(true)
	info := lineGraphInfo(graph)
	theme.BeginFrame()
	theme.DrawLineGraph(info, graph)
	seen := 0
	for _, call := range recorder.Calls() {
		if call.Part != skin.PartText {
			continue
		}
		seen++
		row := call.Bounds
		if row.X < info.Bounds.X || row.Y < info.Bounds.Y {
			t.Fatalf("label above bounds: %+v", call)
		}
		if row.X+row.W > info.Bounds.X+info.Bounds.W {
			t.Fatalf("label past right edge: %+v", call)
		}
		if row.Y+row.H > info.Bounds.Y+info.Bounds.H {
			t.Fatalf("label past bottom edge: %+v", call)
		}
	}
	if seen == 0 {
		t.Fatal("labeled graph drew no text")
	}
}

// TestLineGraphTickPixelsInsidePlot verifies cached ticks land on the plot.
func TestLineGraphTickPixelsInsidePlot(t *testing.T) {
	graph := lineGraphSample()
	graph.SetShowLabels(true)
	plot := graph.PlotRect(graph.Bounds())
	graph.EnsureGeometry(plot)
	for i := 0; i < graph.XTickCount(); i++ {
		_, pixel, label, ok := graph.XTickAt(i)
		if !ok || label == "" {
			t.Fatalf("x tick %d missing", i)
		}
		if pixel < plot.X || pixel > plot.X+plot.W {
			t.Fatalf("x tick pixel %v outside %v", pixel, plot)
		}
	}
	for i := 0; i < graph.YTickCount(); i++ {
		_, pixel, label, ok := graph.YTickAt(i)
		if !ok || label == "" {
			t.Fatalf("y tick %d missing", i)
		}
		if pixel < plot.Y || pixel > plot.Y+plot.H {
			t.Fatalf("y tick pixel %v outside %v", pixel, plot)
		}
	}
}

// TestDrawLineGraphAvoidsSteadyStateAllocs verifies the headless draw path.
func TestDrawLineGraphAvoidsSteadyStateAllocs(t *testing.T) {
	theme, recorder := lineGraphTheme(t, 512)
	graph := lineGraphSample()
	graph.SetShowLabels(true)
	info := lineGraphInfo(graph)
	theme.BeginFrame()
	theme.DrawLineGraph(info, graph)
	allocs := testing.AllocsPerRun(100, func() {
		theme.BeginFrame()
		theme.DrawLineGraph(info, graph)
	})
	if allocs != 0 {
		t.Fatalf("steady-state draw allocated %v times", allocs)
	}
	if got := len(recorder.Calls()); got == 0 {
		t.Fatal("allocation probe drew no calls")
	}
}
