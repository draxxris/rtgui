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
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State, info.Class)
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
	background, _ := t.resolveDescriptor(info.Kind, skin.PartBackground, info.State, info.Class)
	border, _ := t.resolveDescriptor(info.Kind, skin.PartBorder, info.State, info.Class)
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
// clipped segments, optional area fills and glows, and markers for points
// without a drawable neighbor. Fills draw under all strokes of the series.
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
		fx, _ := graph.SeriesFX(s)
		rgba := tint.RGBA()
		if fx.FillEnabled {
			t.drawGraphFill(info, graph, s, plot, rgba, fx)
		}
		t.drawGraphSeries(info, graph, s, plot, rgba, thickness, step, fx)
	}
}

// drawGraphSeries renders one series, breaking runs at non-finite points.
// Halo passes derive once per series so every segment shares one clip.
func (t *Theme) drawGraphSeries(info core.WidgetInfo, graph *widgets.LineGraph, series int, plot core.Rect, tint color.RGBA, thickness float32, step bool, fx widgets.LineSeriesFX) {
	previous, hasPrevious := core.Vec2{}, false
	count := graph.MappedLen(series)
	glow := newGraphGlow(tint, thickness, fx)
	for i := 0; i < count; i++ {
		current, ok := graph.MappedPointAt(series, i)
		if !ok {
			hasPrevious = false
			continue
		}
		if !hasPrevious {
			if _, nextOK := graph.MappedPointAt(series, i+1); !nextOK {
				t.drawGraphMarker(info, current, plot, tint, thickness, glow)
			}
			previous, hasPrevious = current, true
			continue
		}
		if step {
			corner := core.Vec2{X: current.X, Y: previous.Y}
			t.drawGraphSegment(info, previous, corner, plot, tint, thickness, glow)
			t.drawGraphSegment(info, corner, current, plot, tint, thickness, glow)
		} else {
			t.drawGraphSegment(info, previous, current, plot, tint, thickness, glow)
		}
		previous = current
	}
}

// graphGlow holds one series' precomputed halo passes so every segment
// shares a single clip and markers reuse the same alphas and widths.
type graphGlow struct {
	// enabled selects halo passes under the core stroke when true.
	enabled bool
	// outer and mid are the halo tints, faint wide then stronger narrow.
	outer, mid color.RGBA
	// outerW and midW are the halo widths derived from the core thickness.
	outerW, midW float32
}

// haloColors derives the outer and mid halo tints from the series color.
// The outer pass is fainter (a quarter of GlowAlpha) while the mid pass
// is stronger (half of GlowAlpha).
func haloColors(tint color.RGBA, fx widgets.LineSeriesFX) (outer, mid color.RGBA) {
	return color.RGBA{R: tint.R, G: tint.G, B: tint.B, A: fx.GlowAlpha / 4},
		color.RGBA{R: tint.R, G: tint.G, B: tint.B, A: fx.GlowAlpha / 2}
}

// newGraphGlow derives the halo passes from the core stroke style once per
// series. A disabled effect returns the zero value, which draws core only.
func newGraphGlow(tint color.RGBA, thickness float32, fx widgets.LineSeriesFX) graphGlow {
	if !fx.GlowEnabled {
		return graphGlow{}
	}
	outer, mid := haloColors(tint, fx)
	return graphGlow{
		enabled: true,
		outer:    outer,
		mid:      mid,
		outerW:   thickness + fx.GlowWidth,
		midW:     thickness + fx.GlowWidth/2,
	}
}

// drawGraphSegment clips one segment to the plot once, then records and
// draws the halo passes under the core line. Stroking all three passes
// from the same clipped endpoints keeps halos from overdrawing the
// neighboring core at segment joints. A disabled glow draws core only.
func (t *Theme) drawGraphSegment(info core.WidgetInfo, a, b core.Vec2, plot core.Rect, tint color.RGBA, thickness float32, glow graphGlow) {
	clippedA, clippedB, ok := clipGraphSegment(a, b, plot)
	if !ok {
		return
	}
	span := spanningRect(clippedA, clippedB)
	from := rl.NewVector2(clippedA.X, clippedA.Y)
	to := rl.NewVector2(clippedB.X, clippedB.Y)
	if glow.enabled {
		t.logDrawCall(info.Kind, skin.PartOverlay, info.State, span, span, skin.SkinDescriptor{}, glow.outer, false)
		if rl.IsWindowReady() {
			rl.DrawLineEx(from, to, glow.outerW, glow.outer)
		}
		t.logDrawCall(info.Kind, skin.PartOverlay, info.State, span, span, skin.SkinDescriptor{}, glow.mid, false)
		if rl.IsWindowReady() {
			rl.DrawLineEx(from, to, glow.midW, glow.mid)
		}
	}
	t.logDrawCall(info.Kind, skin.PartOverlay, info.State, span, span, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawLineEx(from, to, thickness, tint)
	}
}

// drawGraphFill shades the area from the polyline down to the plot bottom
// with a GPU-interpolated vertical fade. One recorder call per continuous
// run keeps headless diagnostics bounded; the windowed mesh uses per-vertex
// alpha so the fade stays smooth without banding. Step series fill the
// stepped corners and gaps split runs like the stroke path.
func (t *Theme) drawGraphFill(info core.WidgetInfo, graph *widgets.LineGraph, series int, plot core.Rect, tint color.RGBA, fx widgets.LineSeriesFX) {
	count := graph.MappedLen(series)
	if count == 0 || plot.W <= 0 || plot.H <= 0 {
		return
	}
	step := graph.Interpolation() == widgets.LineGraphStep
	top := color.RGBA{R: tint.R, G: tint.G, B: tint.B, A: fx.FillTopAlpha}
	bottom := color.RGBA{R: tint.R, G: tint.G, B: tint.B}
	baseY := plot.Y + plot.H
	u0, v0, u1, v1, ready := beginFillMesh()
	previous, hasPrevious := core.Vec2{}, false
	runMinX, runMaxX, runTop, runEmitted := float32(0), float32(0), float32(0), false
	for i := 0; i < count; i++ {
		current, ok := graph.MappedPointAt(series, i)
		if !ok {
			if runEmitted {
				t.logFillRun(info, plot, runMinX, runMaxX, runTop, baseY, top)
				runEmitted = false
			}
			hasPrevious = false
			continue
		}
		if !hasPrevious {
			previous, hasPrevious = current, true
			runMinX, runMaxX, runTop = current.X, current.X, current.Y
			continue
		}
		if step {
			corner := core.Vec2{X: current.X, Y: previous.Y}
			runMinX, runMaxX, runTop = trackFillRun(runMinX, runMaxX, runTop, runEmitted, previous, corner)
			if ready {
				emitFillQuad(previous, corner, baseY, top, bottom, u0, v0, u1, v1)
			}
			runEmitted = true
			// The corner-to-current edge is vertical, so it adds no fill
			// area; only fold current into the run bounds.
			runMinX, runMaxX, runTop = trackFillRun(runMinX, runMaxX, runTop, runEmitted, corner, current)
		} else {
			runMinX, runMaxX, runTop = trackFillRun(runMinX, runMaxX, runTop, runEmitted, previous, current)
			if ready {
				emitFillQuad(previous, current, baseY, top, bottom, u0, v0, u1, v1)
			}
			runEmitted = true
		}
		previous = current
	}
	if ready {
		endFillMesh()
	}
	if runEmitted {
		t.logFillRun(info, plot, runMinX, runMaxX, runTop, baseY, top)
	}
}

// beginFillMesh binds the white shapes pixel and opens a triangle batch for
// fill emission. It reports ready=false without touching GL headlessly, so
// callers emit nothing and only record headless diagnostics. Texcoords stamp
// the shapes pixel like DrawTriangle does; without them the sampler hits
// stale font texels and the fill stays invisible.
func beginFillMesh() (u0, v0, u1, v1 float32, ready bool) {
	if !rl.IsWindowReady() {
		return 0, 0, 0, 0, false
	}
	shapeTex := rl.GetShapesTexture()
	shapeRec := rl.GetShapesTextureRectangle()
	if shapeTex.Width > 0 && shapeTex.Height > 0 {
		u0 = shapeRec.X / float32(shapeTex.Width)
		v0 = shapeRec.Y / float32(shapeTex.Height)
		u1 = (shapeRec.X + shapeRec.Width) / float32(shapeTex.Width)
		v1 = (shapeRec.Y + shapeRec.Height) / float32(shapeTex.Height)
	}
	rl.SetTexture(shapeTex.ID)
	rl.Begin(rl.Triangles)
	return u0, v0, u1, v1, true
}

// endFillMesh closes the batch opened by beginFillMesh and unbinds the
// shapes texture. It is a no-op unless ready reports a windowed batch.
func endFillMesh() {
	rl.End()
	rl.SetTexture(0)
}

// trackFillRun folds segment endpoints into the active run bounds.
func trackFillRun(runMinX, runMaxX, runTop float32, started bool, a, b core.Vec2) (float32, float32, float32) {
	if !started {
		runMinX, runMaxX, runTop = a.X, a.X, a.Y
	}
	if a.X < runMinX {
		runMinX = a.X
	}
	if a.X > runMaxX {
		runMaxX = a.X
	}
	if a.Y < runTop {
		runTop = a.Y
	}
	if b.X < runMinX {
		runMinX = b.X
	}
	if b.X > runMaxX {
		runMaxX = b.X
	}
	if b.Y < runTop {
		runTop = b.Y
	}
	return runMinX, runMaxX, runTop
}

// logFillRun records one fill run clamped to the plot for headless tests.
// It returns immediately without diagnostics, skipping all clamping math.
func (t *Theme) logFillRun(info core.WidgetInfo, plot core.Rect, minX, maxX, topY, baseY float32, tint color.RGBA) {
	if t == nil || t.recorder == nil {
		return
	}
	if minX < plot.X {
		minX = plot.X
	}
	if maxX > plot.X+plot.W {
		maxX = plot.X + plot.W
	}
	if topY < plot.Y {
		topY = plot.Y
	}
	if maxX <= minX || baseY <= topY {
		return
	}
	dest := core.Rect{X: minX, Y: topY, W: maxX - minX, H: baseY - topY}
	t.logDrawCall(info.Kind, skin.PartOverlay, info.State, dest, dest, skin.SkinDescriptor{}, tint, false)
}

// emitFillQuad emits the two triangles covering the quad from segment ab
// down to baseY. Top vertices carry the line alpha while base vertices are
// transparent, so the rasterizer fades the column smoothly. Zero-width
// vertical steps emit nothing. Texcoords stamp the white shapes pixel so
// the batch sampler keeps the vertex colors intact. Vertices run in the
// counter-clockwise order raylib shapes use, since backface culling stays
// enabled; the mirrored order is culled and the fill stays invisible.
func emitFillQuad(a, b core.Vec2, baseY float32, top, bottom color.RGBA, u0, v0, u1, v1 float32) {
	if a.X == b.X {
		return
	}
	rl.TexCoord2f(u1, v1)
	rl.Color4ub(bottom.R, bottom.G, bottom.B, bottom.A)
	rl.Vertex2f(b.X, baseY)
	rl.TexCoord2f(u1, v0)
	rl.Color4ub(top.R, top.G, top.B, top.A)
	rl.Vertex2f(b.X, b.Y)
	rl.TexCoord2f(u0, v0)
	rl.Color4ub(top.R, top.G, top.B, top.A)
	rl.Vertex2f(a.X, a.Y)
	rl.TexCoord2f(u0, v1)
	rl.Color4ub(bottom.R, bottom.G, bottom.B, bottom.A)
	rl.Vertex2f(a.X, baseY)
	rl.TexCoord2f(u1, v1)
	rl.Color4ub(bottom.R, bottom.G, bottom.B, bottom.A)
	rl.Vertex2f(b.X, baseY)
	rl.TexCoord2f(u0, v0)
	rl.Color4ub(top.R, top.G, top.B, top.A)
	rl.Vertex2f(a.X, a.Y)
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
// the plot stay invisible with or without an active scissor. Glow draws two
// translucent halos under the core square when enabled.
func (t *Theme) drawGraphMarker(info core.WidgetInfo, point core.Vec2, plot core.Rect, tint color.RGBA, thickness float32, glow graphGlow) {
	if !plot.Contains(point) {
		return
	}
	if glow.enabled {
		t.drawMarkerHalo(info, point, glow)
	}
	half := thickness/2 + 2
	rect := core.Rect{X: point.X - half, Y: point.Y - half, W: thickness + 4, H: thickness + 4}
	t.logDrawCall(info.Kind, skin.PartSpark, info.State, rect, rect, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(rect), tint)
	}
}

// drawMarkerHalo records and draws the two glow squares under an isolated
// point marker, reusing the series halo alphas with matching half widths.
func (t *Theme) drawMarkerHalo(info core.WidgetInfo, point core.Vec2, glow graphGlow) {
	halos := [2]struct {
		half float32
		tint color.RGBA
	}{
		{glow.outerW/2 + 2, glow.outer},
		{glow.midW/2 + 2, glow.mid},
	}
	for _, halo := range halos {
		rect := core.Rect{X: point.X - halo.half, Y: point.Y - halo.half, W: halo.half * 2, H: halo.half * 2}
		t.logDrawCall(info.Kind, skin.PartSpark, info.State, rect, rect, skin.SkinDescriptor{}, halo.tint, false)
		if rl.IsWindowReady() {
			rl.DrawRectangleRec(toRaylibRect(rect), halo.tint)
		}
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
