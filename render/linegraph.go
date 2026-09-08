package render

import (
	"image/color"
	"math"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Line graph draw records reuse generic skin parts scoped by the graph kind:
// PartBackground for the plot shell, PartTrack for grid lines, PartOverlay
// for series segments, PartSpark for isolated-point markers, and PartText
// for axis labels. Grid and label colors are fixed v1 fallbacks; series
// colors and thickness come from the widget. UI draws the optional border last.
var lineGraphGridTint = color.RGBA{R: 200, G: 205, B: 215, A: 255}

// lineGraphLabelSize is the fixed axis label pixel size.
const lineGraphLabelSize = 12

// DrawLineGraph renders a line graph headlessly or windowed without
// per-draw allocations: geometry, ticks, and labels come from the widget
// cache, and every loop below uses indexed reads only. Segments are
// mathematically clipped to the plot and additionally scissored while a
// window is ready so thick strokes never bleed outside.
func (t *Theme) DrawLineGraph(info core.WidgetInfo, graph *widgets.LineGraph) {
	if t == nil || graph == nil {
		return
	}
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	plot := t.LineGraphPlot(info, graph)
	if plot.W <= 0 || plot.H <= 0 {
		return
	}
	graph.EnsureGeometry(plot)
	scissored := false
	if rl.IsWindowReady() {
		t.PushClip(plot)
		scissored = true
	}
	t.drawLineGraphGrid(info, graph, plot)
	t.drawLineGraphSeries(info, graph, plot)
	if scissored {
		t.PopClip()
	}
	if graph.ShowLabels() {
		t.PushClip(info.Bounds)
		t.drawLineGraphLabels(info, graph, plot)
		t.PopClip()
	}
}

// LineGraphPlot resolves the same skin-aware plot for drawing and nearest-point queries.
func (t *Theme) LineGraphPlot(info core.WidgetInfo, graph *widgets.LineGraph) core.Rect {
	background, _ := t.resolveDescriptor(info.Kind, skin.PartBackground, info.State)
	border, _ := t.resolveDescriptor(info.Kind, skin.PartBorder, info.State)
	return t.snap(graph.PlotRect(ContentRect(info.Bounds, background, border)))
}

// drawLineGraphGrid records and draws cached tick grid lines.
func (t *Theme) drawLineGraphGrid(info core.WidgetInfo, graph *widgets.LineGraph, plot core.Rect) {
	if !graph.ShowGrid() {
		return
	}
	t.lineGraphVerticals(info, graph, plot)
	t.lineGraphHorizontals(info, graph, plot)
}

// lineGraphVerticals draws one-pixel vertical lines at cached x ticks.
func (t *Theme) lineGraphVerticals(info core.WidgetInfo, graph *widgets.LineGraph, plot core.Rect) {
	for i := 0; i < graph.XTickCount(); i++ {
		_, pixel, _, ok := graph.XTickAt(i)
		if !ok {
			continue
		}
		line := core.Rect{X: pixel, Y: plot.Y, W: 1, H: plot.H}
		t.logDrawCall(info.Kind, skin.PartTrack, info.State, line, line, skin.SkinDescriptor{}, lineGraphGridTint, false)
		if rl.IsWindowReady() {
			rl.DrawRectangleRec(toRaylibRect(line), lineGraphGridTint)
		}
	}
}

// lineGraphHorizontals draws one-pixel horizontal lines at cached y ticks.
func (t *Theme) lineGraphHorizontals(info core.WidgetInfo, graph *widgets.LineGraph, plot core.Rect) {
	for i := 0; i < graph.YTickCount(); i++ {
		_, pixel, _, ok := graph.YTickAt(i)
		if !ok {
			continue
		}
		line := core.Rect{X: plot.X, Y: pixel, W: plot.W, H: 1}
		t.logDrawCall(info.Kind, skin.PartTrack, info.State, line, line, skin.SkinDescriptor{}, lineGraphGridTint, false)
		if rl.IsWindowReady() {
			rl.DrawRectangleRec(toRaylibRect(line), lineGraphGridTint)
		}
	}
}

// drawLineGraphSeries renders every series in linear or step mode with
// clipped segments and markers for points without a drawable neighbor.
func (t *Theme) drawLineGraphSeries(info core.WidgetInfo, graph *widgets.LineGraph, plot core.Rect) {
	step := graph.Interpolation() == widgets.LineGraphStep
	for s := 0; s < graph.SeriesCount(); s++ {
		if !graph.SeriesVisible(s) {
			continue
		}
		tint, ok := graph.SeriesColor(s)
		if !ok {
			continue
		}
		thickness, _ := graph.SeriesThickness(s)
		t.drawGraphSeries(info, graph, s, plot, tint.RGBA(), thickness, step)
	}
}

// drawGraphSeries renders one series, breaking runs at non-finite points.
func (t *Theme) drawGraphSeries(info core.WidgetInfo, graph *widgets.LineGraph, series int, plot core.Rect, tint color.RGBA, thickness float32, step bool) {
	previous, hasPrevious := core.Vec2{}, false
	count := graph.MappedLen(series)
	for i := 0; i < count; i++ {
		current, ok := graph.MappedPointAt(series, i)
		if !ok {
			hasPrevious = false
			continue
		}
		if !hasPrevious {
			if _, nextOK := graph.MappedPointAt(series, i+1); !nextOK {
				t.drawGraphMarker(info, current, plot, tint, thickness)
			}
			previous, hasPrevious = current, true
			continue
		}
		if step {
			corner := core.Vec2{X: current.X, Y: previous.Y}
			t.drawGraphSegment(info, previous, corner, plot, tint, thickness)
			t.drawGraphSegment(info, corner, current, plot, tint, thickness)
		} else {
			t.drawGraphSegment(info, previous, current, plot, tint, thickness)
		}
		previous = current
	}
}

// drawGraphSegment clips one segment to the plot, then records and draws it.
func (t *Theme) drawGraphSegment(info core.WidgetInfo, a, b core.Vec2, plot core.Rect, tint color.RGBA, thickness float32) {
	clippedA, clippedB, ok := clipGraphSegment(a, b, plot)
	if !ok {
		return
	}
	span := spanningRect(clippedA, clippedB)
	t.logDrawCall(info.Kind, skin.PartOverlay, info.State, span, span, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawLineEx(rl.NewVector2(clippedA.X, clippedA.Y), rl.NewVector2(clippedB.X, clippedB.Y), thickness, tint)
	}
}

// spanningRect returns the normalized bounds covering two clipped endpoints.
// Screen Y flips relative to data Y, so raw endpoint differences go negative
// for rising segments; normalization keeps recorded bounds valid.
func spanningRect(a, b core.Vec2) core.Rect {
	minX, maxX := a.X, b.X
	if maxX < minX {
		minX, maxX = maxX, minX
	}
	minY, maxY := a.Y, b.Y
	if maxY < minY {
		minY, maxY = maxY, minY
	}
	return core.Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

// drawGraphMarker records and draws one isolated-point square. Points outside
// the plot stay invisible with or without an active scissor.
func (t *Theme) drawGraphMarker(info core.WidgetInfo, point core.Vec2, plot core.Rect, tint color.RGBA, thickness float32) {
	if !plot.Contains(point) {
		return
	}
	half := thickness/2 + 2
	rect := core.Rect{X: point.X - half, Y: point.Y - half, W: thickness + 4, H: thickness + 4}
	t.logDrawCall(info.Kind, skin.PartSpark, info.State, rect, rect, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(rect), tint)
	}
}

// drawLineGraphLabels records and draws cached axis tick labels. Edge labels
// clamp inside the widget bounds so first and last ticks never bleed into
// neighboring widgets regardless of label width.
func (t *Theme) drawLineGraphLabels(info core.WidgetInfo, graph *widgets.LineGraph, plot core.Rect) {
	tint := defaultWidgetTextColor(info, info.State)
	for i := 0; i < graph.XTickCount(); i++ {
		_, pixel, label, ok := graph.XTickAt(i)
		if !ok || label == "" {
			continue
		}
		width := t.MeasureText(label, lineGraphLabelSize, false)
		x := clampLabel(pixel-width/2, width, info.Bounds.X, info.Bounds.X+info.Bounds.W)
		y := clampLabel(plot.Y+plot.H+4, lineGraphLabelSize, info.Bounds.Y, info.Bounds.Y+info.Bounds.H)
		t.drawGraphLabel(info, label, x, y, width, tint)
	}
	for i := 0; i < graph.YTickCount(); i++ {
		_, pixel, label, ok := graph.YTickAt(i)
		if !ok || label == "" {
			continue
		}
		width := t.MeasureText(label, lineGraphLabelSize, false)
		x := clampLabel(plot.X-4-width, width, info.Bounds.X, info.Bounds.X+info.Bounds.W)
		y := clampLabel(pixel-lineGraphLabelSize/2, lineGraphLabelSize, info.Bounds.Y, info.Bounds.Y+info.Bounds.H)
		t.drawGraphLabel(info, label, x, y, width, tint)
	}
}

// clampLabel keeps a label of size inside [lo, hi], preferring its natural
// position and then its leading edge when the label exceeds the span.
func clampLabel(pos, size, lo, hi float32) float32 {
	if pos < lo {
		return lo
	}
	if pos+size > hi {
		pos = hi - size
		if pos < lo {
			pos = lo
		}
	}
	return pos
}

// drawGraphLabel records one label row and draws its cached text.
func (t *Theme) drawGraphLabel(info core.WidgetInfo, text string, x, y, width float32, tint color.RGBA) {
	row := core.Rect{X: x, Y: y, W: width, H: lineGraphLabelSize}
	t.logDrawCall(info.Kind, skin.PartText, info.State, row, row, skin.SkinDescriptor{}, tint, false)
	t.DrawText(text, x, y, lineGraphLabelSize, false, tint)
}

// clipGraphSegment clips segment ab to plot with Liang-Barsky, reporting
// false when the segment lies fully outside. All math is stack-local.
func clipGraphSegment(a, b core.Vec2, plot core.Rect) (core.Vec2, core.Vec2, bool) {
	if math.IsNaN(float64(a.X+a.Y+b.X+b.Y)) || math.IsInf(float64(a.X+a.Y+b.X+b.Y), 0) {
		return core.Vec2{}, core.Vec2{}, false
	}
	x0, y0 := float64(a.X), float64(a.Y)
	dx := float64(b.X - a.X)
	dy := float64(b.Y - a.Y)
	xMin := float64(plot.X)
	xMax := float64(plot.X + plot.W)
	yMin := float64(plot.Y)
	yMax := float64(plot.Y + plot.H)
	// Each edge holds p and q: entering edges raise t0, leaving edges lower t1.
	edges := [4][2]float64{
		{-dx, x0 - xMin},
		{dx, xMax - x0},
		{-dy, y0 - yMin},
		{dy, yMax - y0},
	}
	enter, leave := 0.0, 1.0
	for _, edge := range edges {
		p, q := edge[0], edge[1]
		if p == 0 {
			if q < 0 {
				return core.Vec2{}, core.Vec2{}, false
			}
			continue
		}
		ratio := q / p
		if p < 0 {
			if ratio > leave {
				return core.Vec2{}, core.Vec2{}, false
			}
			if ratio > enter {
				enter = ratio
			}
			continue
		}
		if ratio < enter {
			return core.Vec2{}, core.Vec2{}, false
		}
		if ratio < leave {
			leave = ratio
		}
	}
	if enter > leave {
		return core.Vec2{}, core.Vec2{}, false
	}
	clippedA := core.Vec2{X: float32(x0 + enter*dx), Y: float32(y0 + enter*dy)}
	clippedB := core.Vec2{X: float32(x0 + leave*dx), Y: float32(y0 + leave*dy)}
	return clippedA, clippedB, true
}
