package widgets

import (
	"fmt"
	"math"
	"strconv"

	"github.com/draxxris/rtgui/core"
)

// LineGraphKind identifies LineGraph widgets. It aliases core.WidgetLineGraph
// so snapshots always match theme and skin lookups registered for the kind.
const LineGraphKind core.WidgetKind = core.WidgetLineGraph

// LineGraphInterp selects how consecutive points in every series connect.
type LineGraphInterp int32

const (
	// LineGraphLinear connects consecutive points with straight segments.
	LineGraphLinear LineGraphInterp = iota
	// LineGraphStep connects consecutive points with horizontal-then-vertical steps.
	LineGraphStep
)

// Line graph layout and normalization constants.
const (
	// lineGraphContentPad is the unskinned inset kept on every plot side.
	lineGraphContentPad = 4.0
	// lineGraphLabelLeft reserves the y-axis label gutter when labels show.
	lineGraphLabelLeft = 44.0
	// lineGraphLabelBottom reserves the x-axis label gutter when labels show.
	lineGraphLabelBottom = 20.0
	// lineGraphDefaultDivisions is the default grid division count per axis.
	lineGraphDefaultDivisions = 4
	// lineGraphMaxDivisions bounds per-axis grid divisions.
	lineGraphMaxDivisions = 16
	// lineGraphDefaultThickness is used when a series thickness is not positive.
	lineGraphDefaultThickness = 2.0
	// lineGraphDefaultFillAlpha is the gradient top opacity for enabled fills.
	lineGraphDefaultFillAlpha = uint8(96)
	// lineGraphDefaultGlowAlpha is the outer glow opacity for enabled glows.
	lineGraphDefaultGlowAlpha = uint8(80)
	// lineGraphDefaultGlowWidth is the extra pixel width of the outer glow pass.
	lineGraphDefaultGlowWidth = float32(6)
)

// LineSeriesFX configures the area fill and line glow for one data series.
// The zero value disables both effects, preserving the flat stroke look.
// Fill shades the area from the polyline down to the plot bottom with a
// vertical fade; glow strokes translucent passes under the core line.
type LineSeriesFX struct {
	// FillEnabled shades the area under the line when true.
	FillEnabled bool
	// FillTopAlpha is the fill opacity at the line; 0 selects the default.
	FillTopAlpha uint8
	// GlowEnabled strokes soft halo passes under the core line when true.
	GlowEnabled bool
	// GlowAlpha is the halo opacity budget; the mid pass uses half and the
	// outer pass a quarter. Zero selects the default.
	GlowAlpha uint8
	// GlowWidth is the extra pixel width of the outer halo pass.
	GlowWidth float32
}

// lineGraphPalette supplies default series colors when AddSeries receives
// a zero color, so multi-series graphs stay distinguishable by default.
var lineGraphPalette = [...]core.Color{
	{R: 60, G: 120, B: 220, A: 255},
	{R: 220, G: 80, B: 60, A: 255},
	{R: 60, G: 170, B: 90, A: 255},
	{R: 230, G: 170, B: 40, A: 255},
	{R: 150, G: 90, B: 220, A: 255},
	{R: 40, G: 180, B: 180, A: 255},
}

// lineSeries is one stored data series with its own style.
type lineSeries struct {
	// points holds copied data in insertion order, oldest first.
	points []core.Vec2
	// color is the normalized series stroke color.
	color core.Color
	// thickness is the normalized stroke width in logical pixels.
	thickness float32
	// visible selects whether the series draws and answers queries.
	visible bool
	// fx holds the normalized fill and glow configuration.
	fx LineSeriesFX
}

// lineTickKey packs every input that tick values, pixels, and labels depend
// on. The struct is comparable so EnsureGeometry detects reusable ticks with
// one comparison instead of a branch chain.
type lineTickKey struct {
	// plot is the plot rectangle ticks were built for.
	plot core.Rect
	// xMin, xMax, yMin, and yMax are the data ranges ticks were built for.
	xMin, xMax, yMin, yMax float32
	// gridX and gridY are the division counts ticks were built for.
	gridX, gridY int
	// showGrid and showLabels are the display flags ticks were built for.
	showGrid, showLabels bool
	// formatVersion identifies the label formatter ticks were built with.
	formatVersion uint64
}

// LineGraph is a versatile multi-series line/step chart widget. Series data
// is always copied on write and truncated to MaxPoints, while reads are
// indexed value copies that never expose backing storage. Mapped geometry,
// ticks, and formatted labels are cached and rebuilt only when data, ranges,
// options, or the plot rectangle change, so steady-state draws allocate
// nothing. Non-finite points split segments and are skipped by range scans
// and nearest-point queries. Render draws through Theme.DrawLineGraph.
type LineGraph struct {
	base
	series        []lineSeries
	maxPoints     int
	interp        LineGraphInterp
	hasXRange     bool
	xMin          float32
	xMax          float32
	hasYRange     bool
	yMin          float32
	yMax          float32
	showGrid      bool
	gridX         int
	gridY         int
	showLabels    bool
	version       uint64
	cacheValid    bool
	cachePlot     core.Rect
	cacheVersion  uint64
	tickFormat    func(float32) string
	formatVersion uint64
	cacheTickKey  lineTickKey
	hasTickCache  bool
	mapped        [][]core.Vec2
	xTicks        []float32
	yTicks        []float32
	xPixels       []float32
	yPixels       []float32
	xLabels       []string
	yLabels       []string
}

// NewLineGraph returns a graph with automatic ranges, grid lines, and no labels.
func NewLineGraph(name string, bounds core.Rect) *LineGraph {
	return &LineGraph{
		base:       newBase(name, LineGraphKind, bounds),
		maxPoints:  1024,
		showGrid:   true,
		gridX:      lineGraphDefaultDivisions,
		gridY:      lineGraphDefaultDivisions,
		showLabels: false,
	}
}

// invalidate bumps the data version and drops cached geometry.
func (g *LineGraph) invalidate() {
	if g == nil {
		return
	}
	g.version++
	g.cacheValid = false
}

// finiteVec reports whether both components are finite numbers.
func finiteVec(p core.Vec2) bool {
	if math.IsNaN(float64(p.X)) || math.IsInf(float64(p.X), 0) {
		return false
	}
	return !math.IsNaN(float64(p.Y)) && !math.IsInf(float64(p.Y), 0)
}

// normalizeThickness replaces non-positive or non-finite widths with the default.
func normalizeThickness(thickness float32) float32 {
	if math.IsNaN(float64(thickness)) || math.IsInf(float64(thickness), 0) || thickness <= 0 {
		return lineGraphDefaultThickness
	}
	return thickness
}

// defaultSeriesColor returns color when set, else a palette entry by index.
func defaultSeriesColor(color core.Color, index int) core.Color {
	if color != (core.Color{}) {
		return color
	}
	return lineGraphPalette[index%len(lineGraphPalette)]
}

// SeriesCount returns the number of data series.
func (g *LineGraph) SeriesCount() int {
	if g == nil {
		return 0
	}
	return len(g.series)
}

// AddSeries appends an empty series and returns its index. A zero color
// selects a palette entry; a non-positive thickness uses the default width.
func (g *LineGraph) AddSeries(color core.Color, thickness float32) int {
	if g == nil {
		return -1
	}
	index := len(g.series)
	g.series = append(g.series, lineSeries{
		color:     defaultSeriesColor(color, index),
		thickness: normalizeThickness(thickness),
		visible:   true,
	})
	g.invalidate()
	return index
}

// RemoveSeries deletes one series and reports success.
func (g *LineGraph) RemoveSeries(series int) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	copy(g.series[series:], g.series[series+1:])
	g.series[len(g.series)-1] = lineSeries{}
	g.series = g.series[:len(g.series)-1]
	clear(g.mapped)
	g.mapped = g.mapped[:0]
	g.invalidate()
	return true
}

// SeriesLen returns the stored point count of one series.
func (g *LineGraph) SeriesLen(series int) int {
	if g == nil || series < 0 || series >= len(g.series) {
		return 0
	}
	return len(g.series[series].points)
}

// SeriesPointAt returns one stored data point by index without exposing storage.
func (g *LineGraph) SeriesPointAt(series, index int) (core.Vec2, bool) {
	if g == nil || series < 0 || series >= len(g.series) {
		return core.Vec2{}, false
	}
	points := g.series[series].points
	if index < 0 || index >= len(points) {
		return core.Vec2{}, false
	}
	return points[index], true
}

// SeriesColor returns one series stroke color.
func (g *LineGraph) SeriesColor(series int) (core.Color, bool) {
	if g == nil || series < 0 || series >= len(g.series) {
		return core.Color{}, false
	}
	return g.series[series].color, true
}

// SetSeriesColor replaces one series stroke color and reports success.
// A zero color reselects the palette entry for that index.
func (g *LineGraph) SetSeriesColor(series int, color core.Color) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	g.series[series].color = defaultSeriesColor(color, series)
	g.invalidate()
	return true
}

// SeriesThickness returns one series stroke width in logical pixels.
func (g *LineGraph) SeriesThickness(series int) (float32, bool) {
	if g == nil || series < 0 || series >= len(g.series) {
		return 0, false
	}
	return g.series[series].thickness, true
}

// SetSeriesThickness replaces one series stroke width and reports success.
// Non-positive or non-finite widths fall back to the default.
func (g *LineGraph) SetSeriesThickness(series int, thickness float32) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	g.series[series].thickness = normalizeThickness(thickness)
	g.invalidate()
	return true
}

// normalizeSeriesFX fills zero alphas and widths with defaults when the
// corresponding effect is enabled, so LineSeriesFX{} stays off while
// partially specified configs still render visibly.
func normalizeSeriesFX(fx LineSeriesFX) LineSeriesFX {
	if fx.FillEnabled && fx.FillTopAlpha == 0 {
		fx.FillTopAlpha = lineGraphDefaultFillAlpha
	}
	if fx.GlowEnabled {
		if fx.GlowAlpha == 0 {
			fx.GlowAlpha = lineGraphDefaultGlowAlpha
		}
		if !(fx.GlowWidth > 0) {
			fx.GlowWidth = lineGraphDefaultGlowWidth
		}
	}
	return fx
}

// SeriesFX returns one series fill and glow configuration.
func (g *LineGraph) SeriesFX(series int) (LineSeriesFX, bool) {
	if g == nil || series < 0 || series >= len(g.series) {
		return LineSeriesFX{}, false
	}
	return g.series[series].fx, true
}

// SetSeriesFX replaces one series fill and glow configuration and reports
// success. Zero alphas and widths fall back to defaults when enabled;
// a zero struct disables both effects.
func (g *LineGraph) SetSeriesFX(series int, fx LineSeriesFX) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	g.series[series].fx = normalizeSeriesFX(fx)
	g.invalidate()
	return true
}

// SeriesVisible reports whether one series draws and answers queries.
// Series are visible by default; unknown indices report false.
func (g *LineGraph) SeriesVisible(series int) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	return g.series[series].visible
}

// SetSeriesVisible toggles one series without deleting its data and reports
// success. Hidden series are skipped by drawing and nearest-point queries.
func (g *LineGraph) SetSeriesVisible(series int, visible bool) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	g.series[series].visible = visible
	g.invalidate()
	return true
}

// SetSeriesData copies points into one series, keeping only the newest
// MaxPoints entries when bounded, and reports success.
func (g *LineGraph) SetSeriesData(series int, points []core.Vec2) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	data := points
	if g.maxPoints > 0 && len(data) > g.maxPoints {
		data = data[len(data)-g.maxPoints:]
	}
	stored := append(g.series[series].points[:0], data...)
	g.series[series].points = stored
	g.invalidate()
	return true
}

// AppendPoint copies one point onto a series, dropping the oldest entry
// while bounded, and reports success.
func (g *LineGraph) AppendPoint(series int, point core.Vec2) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	stored := &g.series[series]
	if g.maxPoints > 0 && len(stored.points) >= g.maxPoints {
		copy(stored.points, stored.points[1:])
		stored.points = stored.points[:len(stored.points)-1]
	}
	stored.points = append(stored.points, point)
	g.invalidate()
	return true
}

// AppendPoints copies a batch onto one series with a single invalidation,
// keeping only the newest MaxPoints entries when bounded, and reports success.
func (g *LineGraph) AppendPoints(series int, points []core.Vec2) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	stored := &g.series[series]
	if g.maxPoints > 0 {
		if overflow := len(stored.points) + len(points) - g.maxPoints; overflow > 0 {
			if overflow >= len(stored.points) {
				stored.points = stored.points[:0]
			} else {
				copy(stored.points, stored.points[overflow:])
				stored.points = stored.points[:len(stored.points)-overflow]
			}
		}
		if len(points) >= g.maxPoints {
			points = points[len(points)-g.maxPoints:]
		}
	}
	stored.points = append(stored.points, points...)
	g.invalidate()
	return true
}

// AppendSeriesPoints copies one series into dst without exposing storage,
// matching the Dropdown and TabBar read idiom. Unknown series return dst.
func (g *LineGraph) AppendSeriesPoints(series int, dst []core.Vec2) []core.Vec2 {
	if g == nil || series < 0 || series >= len(g.series) {
		return dst
	}
	return append(dst, g.series[series].points...)
}

// ClearSeriesData drops every point in one series and reports success.
func (g *LineGraph) ClearSeriesData(series int) bool {
	if g == nil || series < 0 || series >= len(g.series) {
		return false
	}
	g.series[series].points = g.series[series].points[:0]
	g.invalidate()
	return true
}

// MaxPoints returns the per-series bound, or 0 when unbounded.
func (g *LineGraph) MaxPoints() int {
	if g == nil {
		return 0
	}
	return g.maxPoints
}

// SetMaxPoints bounds every series to the newest n points; n <= 0 unbounds.
func (g *LineGraph) SetMaxPoints(n int) *LineGraph {
	if g == nil {
		return nil
	}
	if n < 0 {
		n = 0
	}
	g.maxPoints = n
	if n > 0 {
		for i := range g.series {
			stored := &g.series[i]
			if len(stored.points) > n {
				copy(stored.points, stored.points[len(stored.points)-n:])
				stored.points = stored.points[:n]
			}
		}
	}
	g.invalidate()
	return g
}

// Interpolation returns the current segment connection mode.
func (g *LineGraph) Interpolation() LineGraphInterp {
	if g == nil {
		return LineGraphLinear
	}
	return g.interp
}

// SetInterpolation selects the segment connection mode for every series.
// Invalid modes are ignored. The cache is dropped so damage trackers
// observing the version see the change.
func (g *LineGraph) SetInterpolation(mode LineGraphInterp) *LineGraph {
	if g != nil && (mode == LineGraphLinear || mode == LineGraphStep) {
		g.interp = mode
		g.invalidate()
	}
	return g
}

// SetTickFormatter replaces the axis label formatter; nil restores the
// default compact format. The formatter runs only during tick rebuilds,
// never per draw, so windowed string costs stay off the hot path.
func (g *LineGraph) SetTickFormatter(format func(float32) string) *LineGraph {
	if g == nil {
		return nil
	}
	g.tickFormat = format
	g.formatVersion++
	g.invalidate()
	return g
}

// finiteRange validates an explicit axis range.
func finiteRange(lo, hi float32) bool {
	if hi <= lo {
		return false
	}
	for _, v := range [2]float32{lo, hi} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
	}
	return true
}

// SetXRange fixes the explicit x-axis range and reports success.
func (g *LineGraph) SetXRange(lo, hi float32) bool {
	if g == nil || !finiteRange(lo, hi) {
		return false
	}
	g.xMin, g.xMax, g.hasXRange = lo, hi, true
	g.invalidate()
	return true
}

// ClearXRange returns the x-axis to automatic ranging and reports a change.
func (g *LineGraph) ClearXRange() bool {
	if g == nil || !g.hasXRange {
		return false
	}
	g.hasXRange = false
	g.invalidate()
	return true
}

// HasXRange reports whether an explicit x-axis range is set.
func (g *LineGraph) HasXRange() bool {
	return g != nil && g.hasXRange
}

// XRange returns the explicit x-axis range when set.
func (g *LineGraph) XRange() (float32, float32, bool) {
	if g == nil || !g.hasXRange {
		return 0, 0, false
	}
	return g.xMin, g.xMax, true
}

// SetYRange fixes the explicit y-axis range and reports success.
func (g *LineGraph) SetYRange(lo, hi float32) bool {
	if g == nil || !finiteRange(lo, hi) {
		return false
	}
	g.yMin, g.yMax, g.hasYRange = lo, hi, true
	g.invalidate()
	return true
}

// ClearYRange returns the y-axis to automatic ranging and reports a change.
func (g *LineGraph) ClearYRange() bool {
	if g == nil || !g.hasYRange {
		return false
	}
	g.hasYRange = false
	g.invalidate()
	return true
}

// HasYRange reports whether an explicit y-axis range is set.
func (g *LineGraph) HasYRange() bool {
	return g != nil && g.hasYRange
}

// YRange returns the explicit y-axis range when set.
func (g *LineGraph) YRange() (float32, float32, bool) {
	if g == nil || !g.hasYRange {
		return 0, 0, false
	}
	return g.yMin, g.yMax, true
}

// scanFinite returns the bounding range of all finite points.
func (g *LineGraph) scanFinite() (xMin, xMax, yMin, yMax float32, found bool) {
	for si := range g.series {
		points := g.series[si].points
		for i := range points {
			p := points[i]
			if !finiteVec(p) {
				continue
			}
			if !found {
				xMin, xMax, yMin, yMax, found = p.X, p.X, p.Y, p.Y, true
				continue
			}
			if p.X < xMin {
				xMin = p.X
			}
			if p.X > xMax {
				xMax = p.X
			}
			if p.Y < yMin {
				yMin = p.Y
			}
			if p.Y > yMax {
				yMax = p.Y
			}
		}
	}
	return xMin, xMax, yMin, yMax, found
}

// expandConstant widens a degenerate range so singleton and constant
// series still map to visible geometry instead of dividing by zero.
func expandConstant(lo, hi float32) (float32, float32) {
	if hi < lo {
		lo, hi = hi, lo
	}
	if hi > lo {
		return lo, hi
	}
	half := float32(0.5)
	if lo != 0 {
		half = lo * 0.05
		if half < 0 {
			half = -half
		}
		if half == 0 {
			half = 0.5
		}
	}
	return lo - half, lo + half
}

// EffectiveRange resolves explicit ranges over automatic ones scanned from
// finite data. Constant axes are widened; ok is false when no finite data
// exists and an axis still needs automatic ranging.
func (g *LineGraph) EffectiveRange() (xMin, xMax, yMin, yMax float32, ok bool) {
	if g == nil {
		return 0, 1, 0, 1, false
	}
	xMin, xMax, xOK := g.xMin, g.xMax, g.hasXRange
	yMin, yMax, yOK := g.yMin, g.yMax, g.hasYRange
	if !xOK || !yOK {
		autoXMin, autoXMax, autoYMin, autoYMax, found := g.scanFinite()
		if !found {
			return 0, 1, 0, 1, false
		}
		if !xOK {
			xMin, xMax = autoXMin, autoXMax
		}
		if !yOK {
			yMin, yMax = autoYMin, autoYMax
		}
	}
	xMin, xMax = expandConstant(xMin, xMax)
	yMin, yMax = expandConstant(yMin, yMax)
	return xMin, xMax, yMin, yMax, true
}

// ShowGrid reports whether grid lines draw.
func (g *LineGraph) ShowGrid() bool {
	return g != nil && g.showGrid
}

// SetShowGrid toggles grid lines.
func (g *LineGraph) SetShowGrid(show bool) *LineGraph {
	if g != nil && g.showGrid != show {
		g.showGrid = show
		g.invalidate()
	}
	return g
}

// GridDivisions returns the per-axis grid division counts.
func (g *LineGraph) GridDivisions() (x, y int) {
	if g == nil {
		return 0, 0
	}
	return g.gridX, g.gridY
}

// clampDivisions keeps a division count inside the supported range.
func clampDivisions(n int) int {
	if n < 0 {
		return 0
	}
	if n > lineGraphMaxDivisions {
		return lineGraphMaxDivisions
	}
	return n
}

// SetGridDivisions configures per-axis grid divisions clamped to
// [0, lineGraphMaxDivisions]; zero disables that axis.
func (g *LineGraph) SetGridDivisions(x, y int) *LineGraph {
	if g != nil {
		g.gridX, g.gridY = clampDivisions(x), clampDivisions(y)
		g.invalidate()
	}
	return g
}

// ShowLabels reports whether axis labels draw.
func (g *LineGraph) ShowLabels() bool {
	return g != nil && g.showLabels
}

// SetShowLabels toggles axis tick labels.
func (g *LineGraph) SetShowLabels(show bool) *LineGraph {
	if g != nil && g.showLabels != show {
		g.showLabels = show
		g.invalidate()
	}
	return g
}

// SetTooltip attaches a hover tooltip string directly to the line graph.
func (g *LineGraph) SetTooltip(text string) *LineGraph {
	if g != nil {
		g.base.SetTooltip(text)
	}
	return g
}

// PlotRect returns the drawable plot area inside bounds, reserving label
// gutters when labels show. Render and hit queries must use this so drawn
// lines, ticks, and nearest-point results always agree.
func (g *LineGraph) PlotRect(bounds core.Rect) core.Rect {
	x := bounds.X + lineGraphContentPad
	y := bounds.Y + lineGraphContentPad
	w := bounds.W - 2*lineGraphContentPad
	h := bounds.H - 2*lineGraphContentPad
	if g != nil && g.showLabels {
		x += lineGraphLabelLeft
		w -= lineGraphLabelLeft
		h -= lineGraphLabelBottom
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return core.Rect{X: x, Y: y, W: w, H: h}
}

// mapGraphPoint converts one data point into plot coordinates. Non-finite
// input maps to NaN so segments break and queries skip it.
func mapGraphPoint(p core.Vec2, plot core.Rect, xMin, xSpan, yMin, ySpan float32) core.Vec2 {
	if !finiteVec(p) {
		nan := float32(math.NaN())
		return core.Vec2{X: nan, Y: nan}
	}
	fx := (p.X - xMin) / xSpan
	fy := (p.Y - yMin) / ySpan
	return core.Vec2{X: plot.X + fx*plot.W, Y: plot.Y + plot.H - fy*plot.H}
}

// formatTick renders one axis value without caller-visible allocation churn;
// results are cached during geometry rebuilds, never per draw.
func formatTick(v float32) string {
	return strconv.FormatFloat(float64(v), 'g', 4, 32)
}

// appendTicks adds divisions+1 uniform ticks with pixel positions and labels.
// Labels format through format so custom formatters apply at rebuild time.
func appendTicks(values, pixels *[]float32, labels *[]string, divisions int, lo, hi, origin, size float32, invert bool, format func(float32) string) {
	if divisions <= 0 {
		return
	}
	span := hi - lo
	if span <= 0 {
		return
	}
	for i := 0; i <= divisions; i++ {
		f := float32(i) / float32(divisions)
		*values = append(*values, lo+f*span)
		if invert {
			*pixels = append(*pixels, origin+size-f*size)
		} else {
			*pixels = append(*pixels, origin+f*size)
		}
		*labels = append(*labels, format(lo+f*span))
	}
}

// EnsureGeometry rebuilds cached mapped points, ticks, and labels for plot
// when data, ranges, options, or the plot changed. Repeated calls with an
// unchanged plot reuse every backing array and allocate nothing.
func (g *LineGraph) EnsureGeometry(plot core.Rect) {
	if g == nil {
		return
	}
	if g.cacheValid && g.cacheVersion == g.version && g.cachePlot == plot {
		return
	}
	xMin, xMax, yMin, yMax, ok := g.EffectiveRange()
	if !ok {
		xMin, xMax, yMin, yMax = 0, 1, 0, 1
	}
	g.cachePlot = plot
	g.remapSeries(plot, xMin, xMax, yMin, yMax)
	key := g.tickKey(plot, xMin, xMax, yMin, yMax)
	if !g.hasTickCache || key != g.cacheTickKey {
		g.rebuildTicks(plot, xMin, xMax, yMin, yMax)
		g.cacheTickKey = key
		g.hasTickCache = true
	}
	g.cacheVersion = g.version
	g.cacheValid = true
}

// tickKey packs every tick input so streaming data that leaves the range,
// plot, divisions, and formatter unchanged reuses labels without formatting.
func (g *LineGraph) tickKey(plot core.Rect, xMin, xMax, yMin, yMax float32) lineTickKey {
	return lineTickKey{
		plot:          plot,
		xMin:          xMin,
		xMax:          xMax,
		yMin:          yMin,
		yMax:          yMax,
		gridX:         g.gridX,
		gridY:         g.gridY,
		showGrid:      g.showGrid,
		showLabels:    g.showLabels,
		formatVersion: g.formatVersion,
	}
}

// remapSeries refreshes cached plot coordinates reusing backing arrays.
func (g *LineGraph) remapSeries(plot core.Rect, xMin, xMax, yMin, yMax float32) {
	for len(g.mapped) < len(g.series) {
		g.mapped = append(g.mapped, nil)
	}
	g.mapped = g.mapped[:len(g.series)]
	xSpan := xMax - xMin
	ySpan := yMax - yMin
	if xSpan <= 0 {
		xSpan = 1
	}
	if ySpan <= 0 {
		ySpan = 1
	}
	for s := range g.series {
		points := g.series[s].points
		dst := g.mapped[s]
		if cap(dst) < len(points) {
			dst = make([]core.Vec2, len(points))
		} else {
			dst = dst[:len(points)]
		}
		for i := range points {
			dst[i] = mapGraphPoint(points[i], plot, xMin, xSpan, yMin, ySpan)
		}
		g.mapped[s] = dst
	}
}

// rebuildTicks refreshes cached tick values, pixels, and labels.
func (g *LineGraph) rebuildTicks(plot core.Rect, xMin, xMax, yMin, yMax float32) {
	g.xTicks = g.xTicks[:0]
	g.yTicks = g.yTicks[:0]
	g.xPixels = g.xPixels[:0]
	g.yPixels = g.yPixels[:0]
	clear(g.xLabels)
	clear(g.yLabels)
	g.xLabels = g.xLabels[:0]
	g.yLabels = g.yLabels[:0]
	if !g.showGrid && !g.showLabels {
		return
	}
	format := formatTick
	if g.tickFormat != nil {
		format = g.tickFormat
	}
	appendTicks(&g.xTicks, &g.xPixels, &g.xLabels, g.gridX, xMin, xMax, plot.X, plot.W, false, format)
	appendTicks(&g.yTicks, &g.yPixels, &g.yLabels, g.gridY, yMin, yMax, plot.Y, plot.H, true, format)
}

// MappedLen returns the cached mapped point count of one series.
func (g *LineGraph) MappedLen(series int) int {
	if g == nil || !g.cacheValid || series < 0 || series >= len(g.mapped) {
		return 0
	}
	return len(g.mapped[series])
}

// MappedPointAt returns one cached plot coordinate, or false for stale
// geometry, bad indices, or non-finite data. Call EnsureGeometry first.
func (g *LineGraph) MappedPointAt(series, index int) (core.Vec2, bool) {
	if g == nil || !g.cacheValid || series < 0 || series >= len(g.mapped) {
		return core.Vec2{}, false
	}
	mapped := g.mapped[series]
	if index < 0 || index >= len(mapped) || !finiteVec(mapped[index]) {
		return core.Vec2{}, false
	}
	return mapped[index], true
}

// tickReading returns one cached tick triple for shared accessors.
func (g *LineGraph) tickReading(values, pixels []float32, labels []string, index int) (float32, float32, string, bool) {
	if g == nil || !g.cacheValid || index < 0 || index >= len(values) {
		return 0, 0, "", false
	}
	return values[index], pixels[index], labels[index], true
}

// XTickCount returns the cached x-axis tick count.
func (g *LineGraph) XTickCount() int {
	if g == nil || !g.cacheValid {
		return 0
	}
	return len(g.xTicks)
}

// XTickAt returns one cached x-axis tick value, pixel, and label.
func (g *LineGraph) XTickAt(index int) (value, pixel float32, label string, ok bool) {
	if g == nil {
		return 0, 0, "", false
	}
	return g.tickReading(g.xTicks, g.xPixels, g.xLabels, index)
}

// YTickCount returns the cached y-axis tick count.
func (g *LineGraph) YTickCount() int {
	if g == nil || !g.cacheValid {
		return 0
	}
	return len(g.yTicks)
}

// YTickAt returns one cached y-axis tick value, pixel, and label.
func (g *LineGraph) YTickAt(index int) (value, pixel float32, label string, ok bool) {
	if g == nil {
		return 0, 0, "", false
	}
	return g.tickReading(g.yTicks, g.yPixels, g.yLabels, index)
}

// NearestPoint returns the stored point nearest to pos in plot coordinates
// with its series, index, data, and mapped values. Hidden series and
// non-finite points are skipped; ok is false for empty graphs. Geometry is
// ensured first so callers never observe stale results. Pass the same plot
// rectangle used for drawing so the cache stays hot between draw and query.
func (g *LineGraph) NearestPoint(plot core.Rect, pos core.Vec2) (series, index int, data, mapped core.Vec2, ok bool) {
	if g == nil {
		return 0, 0, core.Vec2{}, core.Vec2{}, false
	}
	g.EnsureGeometry(plot)
	best := float32(0)
	for s := range g.series {
		if !g.series[s].visible {
			continue
		}
		points := g.series[s].points
		mappedSet := g.mapped[s]
		for i := range mappedSet {
			mp := mappedSet[i]
			if !finiteVec(mp) {
				continue
			}
			dx := mp.X - pos.X
			dy := mp.Y - pos.Y
			dist := dx*dx + dy*dy
			if !ok || dist < best {
				best = dist
				series, index, mapped, ok = s, i, mp, true
				data = points[i]
			}
		}
	}
	return series, index, data, mapped, ok
}

// NearestPointAt resolves the stored point nearest to pos using the widget's
// own plot area, for hover queries that do not draw first. Prefer
// NearestPoint with a shared plot rectangle when drawing and querying in the
// same frame so one cached geometry serves both.
func (g *LineGraph) NearestPointAt(pos core.Vec2) (series, index int, data, mapped core.Vec2, ok bool) {
	if g == nil {
		return 0, 0, core.Vec2{}, core.Vec2{}, false
	}
	return g.NearestPoint(g.PlotRect(g.Bounds()), pos)
}

// AsLineGraph asserts that w is a *LineGraph.
func AsLineGraph(w Widget) (*LineGraph, error) {
	if w == nil {
		return nil, ErrNilWidget
	}
	if g, ok := w.(*LineGraph); ok {
		return g, nil
	}
	return nil, fmt.Errorf("%w: widget %q is %v, expected %v", ErrKindMismatch, w.Name(), w.Kind(), core.WidgetLineGraph)
}
