package skin

import (
	"math"

	"github.com/draxxris/rtgui/core"
)

// NewLinearGradient builds a linear gradient from 2-4 stops.
// Stops with negative positions are auto-distributed evenly.
func NewLinearGradient(dir GradientDirection, stops ...ColorStop) (Gradient, bool) {
	return makeGradient(GradientLinear, dir, false, 0, 0.5, 0.5, stops)
}

// NewAngleGradient builds a linear gradient from a CSS angle in degrees.
// Zero degrees points to the top and angles grow clockwise.
func NewAngleGradient(angleDeg float32, stops ...ColorStop) (Gradient, bool) {
	return makeGradient(GradientLinear, GradientToBottom, true, angleDeg, 0.5, 0.5, stops)
}

// NewRadialGradient builds a radial gradient centered at cx, cy in [0, 1].
// Distance 0 is the center and 1 is the farthest unit-square corner.
func NewRadialGradient(cx, cy float32, stops ...ColorStop) (Gradient, bool) {
	return makeGradient(GradientRadial, GradientToBottom, false, 0, cx, cy, stops)
}

// makeGradient validates stop count and center then normalizes positions.
func makeGradient(kind GradientKind, dir GradientDirection, useAngle bool, angle, cx, cy float32, stops []ColorStop) (Gradient, bool) {
	if len(stops) < 2 || len(stops) > MaxGradientStops {
		return Gradient{}, false
	}
	if kind == GradientRadial && !validCenter(cx, cy) {
		return Gradient{}, false
	}
	out := Gradient{Kind: kind, Direction: dir, UseAngle: useAngle, AngleDeg: angle, CenterX: cx, CenterY: cy}
	for i := range stops {
		out.Stops[i] = stops[i]
	}
	out.StopCount = len(stops)
	normalizeStopPositions(&out)
	return out, true
}

// validCenter reports whether a radial center lies in the unit square.
func validCenter(cx, cy float32) bool {
	return cx >= 0 && cx <= 1 && cy >= 0 && cy <= 1
}

// normalizeStopPositions resolves auto slots per CSS Images 3: a missing
// first stop anchors at 0, a missing last at 1, and interior runs spread
// evenly between explicit neighbors. Positions then clamp forward so
// sampling never runs backwards.
func normalizeStopPositions(g *Gradient) {
	clampExplicitPositions(g)
	if !hasExplicitPosition(g) {
		spreadEvenly(g)
		return
	}
	anchorEnds(g)
	bridgeRuns(g)
	clampForward(g)
}

// clampExplicitPositions confines authored positions above to 1.
// Negative positions select auto distribution and pass through untouched.
func clampExplicitPositions(g *Gradient) {
	for i := 0; i < g.StopCount; i++ {
		if g.Stops[i].Position > 1 {
			g.Stops[i].Position = 1
		}
	}
}

// hasExplicitPosition reports whether any stop carries an authored position.
func hasExplicitPosition(g *Gradient) bool {
	for i := 0; i < g.StopCount; i++ {
		if g.Stops[i].Position >= 0 {
			return true
		}
	}
	return false
}

// spreadEvenly assigns 0..1 positions across every stop.
func spreadEvenly(g *Gradient) {
	if g.StopCount <= 1 {
		return
	}
	denom := float32(g.StopCount - 1)
	for i := 0; i < g.StopCount; i++ {
		g.Stops[i].Position = float32(i) / denom
	}
}

// anchorEnds pins a missing first stop at 0 and a missing last at 1.
func anchorEnds(g *Gradient) {
	if g.StopCount == 0 {
		return
	}
	if g.Stops[0].Position < 0 {
		g.Stops[0].Position = 0
	}
	if g.Stops[g.StopCount-1].Position < 0 {
		g.Stops[g.StopCount-1].Position = 1
	}
}

// bridgeRuns spreads each auto run evenly between defined neighbors.
// Stops[0] is always defined after anchorEnds, so every run closes.
func bridgeRuns(g *Gradient) {
	prev := 0
	for i := 1; i < g.StopCount; i++ {
		if g.Stops[i].Position >= 0 {
			spreadBetween(g, prev, i)
			prev = i
		}
	}
}

// spreadBetween assigns autos strictly between two defined endpoints.
func spreadBetween(g *Gradient, prev, next int) {
	gap := next - prev - 1
	if gap <= 0 {
		return
	}
	start := g.Stops[prev].Position
	end := g.Stops[next].Position
	for k := 1; k <= gap; k++ {
		g.Stops[prev+k].Position = start + (end-start)*float32(k)/float32(gap+1)
	}
}

// clampForward enforces non-decreasing positions for stable sampling.
func clampForward(g *Gradient) {
	for i := 1; i < g.StopCount; i++ {
		if g.Stops[i].Position < g.Stops[i-1].Position {
			g.Stops[i].Position = g.Stops[i-1].Position
		}
	}
}

// sampleGradient returns the color at normalized distance t in [0, 1].
func sampleGradient(g Gradient, t float32) core.Color {
	if g.StopCount <= 0 {
		return core.Color{}
	}
	if t <= g.Stops[0].Position {
		return g.Stops[0].Color
	}
	last := g.StopCount - 1
	if t >= g.Stops[last].Position {
		return g.Stops[last].Color
	}
	return sampleInnerStop(g, t)
}

// sampleInnerStop interpolates the segment containing t.
func sampleInnerStop(g Gradient, t float32) core.Color {
	for i := 0; i+1 < g.StopCount; i++ {
		lo := g.Stops[i].Position
		hi := g.Stops[i+1].Position
		if t < lo || t > hi {
			continue
		}
		span := hi - lo
		if span <= 0 {
			return g.Stops[i+1].Color
		}
		return lerpCoreColor(g.Stops[i].Color, g.Stops[i+1].Color, (t-lo)/span)
	}
	return g.Stops[g.StopCount-1].Color
}

// lerpCoreColor blends two colors by t in [0, 1] with rounding.
func lerpCoreColor(a, b core.Color, t float32) core.Color {
	return core.Color{
		R: lerpChannel8(a.R, b.R, t),
		G: lerpChannel8(a.G, b.G, t),
		B: lerpChannel8(a.B, b.B, t),
		A: lerpChannel8(a.A, b.A, t),
	}
}

// lerpChannel8 blends two channels by t in [0, 1] with rounding.
func lerpChannel8(a, b uint8, t float32) uint8 {
	return uint8(float32(a) + (float32(b)-float32(a))*t + 0.5)
}

// gradientVector returns the unit direction of a linear gradient.
// Angles follow CSS: 0 points up and values grow clockwise.
func gradientVector(g Gradient) (dx, dy float32) {
	if g.UseAngle {
		return angleVector(g.AngleDeg)
	}
	return directionVector(g.Direction)
}

// angleVector converts CSS degrees to a unit vector.
func angleVector(deg float32) (float32, float32) {
	rad := float64(deg) * math.Pi / 180
	return float32(math.Sin(rad)), float32(-math.Cos(rad))
}

// directionVector returns the unit vector for a named direction.
func directionVector(dir GradientDirection) (float32, float32) {
	const diag = 0.70710678
	switch dir {
	case GradientToTop:
		return 0, -1
	case GradientToRight:
		return 1, 0
	case GradientToLeft:
		return -1, 0
	case GradientToBottomRight:
		return diag, diag
	case GradientToBottomLeft:
		return -diag, diag
	case GradientToTopRight:
		return diag, -diag
	case GradientToTopLeft:
		return -diag, -diag
	default:
		return 0, 1
	}
}

// linearT projects normalized (u, v) onto the gradient line in [0, 1].
func linearT(g Gradient, u, v float32) float32 {
	dx, dy := gradientVector(g)
	denom := abs32(dx) + abs32(dy)
	if denom <= 0 {
		return v
	}
	t := 0.5 + (dx*(u-0.5)+dy*(v-0.5))/denom
	return clamp01f(t)
}

// abs32 returns the absolute value of f.
func abs32(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

// clamp01f confines f to [0, 1].
func clamp01f(f float32) float32 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// RadialMaxDist returns the farthest unit-square corner distance.
func RadialMaxDist(cx, cy float32) float32 {
	best := float32(0)
	for _, corner := range [4][2]float32{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		dx := corner[0] - cx
		dy := corner[1] - cy
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if dist > best {
			best = dist
		}
	}
	if best <= 0 {
		return 1
	}
	return best
}

// sampleAt returns the gradient color at normalized (u, v).
func sampleAt(g Gradient, u, v float32) core.Color {
	if g.Kind == GradientRadial {
		return SampleRadialAt(g, u, v, RadialMaxDist(g.CenterX, g.CenterY))
	}
	return sampleGradient(g, linearT(g, u, v))
}

// SampleRadialAt samples a radial gradient with a precomputed max distance.
func SampleRadialAt(g Gradient, u, v, maxDist float32) core.Color {
	if maxDist <= 0 {
		return sampleGradient(g, 0)
	}
	dx := u - g.CenterX
	dy := v - g.CenterY
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	return sampleGradient(g, clamp01f(dist/maxDist))
}

// SampleLinearAt samples a linear gradient at normalized (u, v).
func SampleLinearAt(g Gradient, u, v float32) core.Color {
	return sampleGradient(g, linearT(g, u, v))
}
