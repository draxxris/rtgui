package skin

import (
	"math"
	"testing"

	"github.com/draxxris/rtgui/core"
)

// gradientTestColor builds an opaque test color from one channel value.
func gradientTestColor(v uint8) core.Color {
	return core.Color{R: v, G: v, B: v, A: 255}
}

// TestNewLinearGradientValidatesCount verifies 2-4 stops build and others fail.
func TestNewLinearGradientValidatesCount(t *testing.T) {
	two := []ColorStop{{Color: gradientTestColor(0), Position: 0}, {Color: gradientTestColor(255), Position: 1}}
	if _, ok := NewLinearGradient(GradientToBottom, two...); !ok {
		t.Fatal("two stops must build")
	}
	four := []ColorStop{
		{Color: gradientTestColor(0), Position: 0}, {Color: gradientTestColor(85), Position: -1},
		{Color: gradientTestColor(170), Position: -1}, {Color: gradientTestColor(255), Position: 1},
	}
	grad, ok := NewLinearGradient(GradientToBottom, four...)
	if !ok || grad.StopCount != 4 {
		t.Fatalf("four stops must build, got %+v ok=%v", grad, ok)
	}
	if _, ok := NewLinearGradient(GradientToBottom, two[:1]...); ok {
		t.Fatal("one stop must fail")
	}
	five := append(append([]ColorStop{}, four...), ColorStop{Color: gradientTestColor(255), Position: 1})
	if _, ok := NewLinearGradient(GradientToBottom, five...); ok {
		t.Fatal("five stops must fail")
	}
}

// TestNormalizeDistributesAuto verifies all-auto stops spread evenly.
func TestNormalizeDistributesAuto(t *testing.T) {
	grad, _ := NewLinearGradient(GradientToBottom,
		ColorStop{Color: gradientTestColor(0), Position: -1},
		ColorStop{Color: gradientTestColor(128), Position: -1},
		ColorStop{Color: gradientTestColor(255), Position: -1})
	if grad.Stops[0].Position != 0 || grad.Stops[1].Position != 0.5 || grad.Stops[2].Position != 1 {
		t.Fatalf("auto positions = %v %v %v", grad.Stops[0].Position, grad.Stops[1].Position, grad.Stops[2].Position)
	}
}

// TestNormalizeFillsGaps verifies missing ends anchor at 0 and 1.
func TestNormalizeFillsGaps(t *testing.T) {
	grad, _ := NewLinearGradient(GradientToBottom,
		ColorStop{Color: gradientTestColor(0), Position: -1},
		ColorStop{Color: gradientTestColor(128), Position: 0.5},
		ColorStop{Color: gradientTestColor(255), Position: -1})
	if grad.Stops[0].Position != 0 || grad.Stops[2].Position != 1 {
		t.Fatalf("gap positions = %v %v %v", grad.Stops[0].Position, grad.Stops[1].Position, grad.Stops[2].Position)
	}
}

// TestNormalizeSpreadsInteriorRun verifies autos split explicit neighbors.
func TestNormalizeSpreadsInteriorRun(t *testing.T) {
	grad, _ := NewLinearGradient(GradientToBottom,
		ColorStop{Color: gradientTestColor(0), Position: -1},
		ColorStop{Color: gradientTestColor(128), Position: -1},
		ColorStop{Color: gradientTestColor(255), Position: 0.6})
	if grad.Stops[0].Position != 0 || grad.Stops[1].Position != 0.3 {
		t.Fatalf("run positions = %v %v %v", grad.Stops[0].Position, grad.Stops[1].Position, grad.Stops[2].Position)
	}
}

// TestNormalizeClampsForward verifies decreasing positions hard-stop.
func TestNormalizeClampsForward(t *testing.T) {
	grad, _ := NewLinearGradient(GradientToBottom,
		ColorStop{Color: gradientTestColor(0), Position: 0.8},
		ColorStop{Color: gradientTestColor(255), Position: 0.2})
	if grad.Stops[1].Position != 0.8 {
		t.Fatalf("clamped position = %v", grad.Stops[1].Position)
	}
	if got := sampleGradient(grad, 0.5); got != gradientTestColor(0) {
		t.Fatalf("pre-clamp sample = %+v", got)
	}
}

// TestSampleGradientEdges verifies clamping outside [0, 1].
func TestSampleGradientEdges(t *testing.T) {
	grad, _ := NewLinearGradient(GradientToBottom,
		ColorStop{Color: gradientTestColor(10), Position: 0},
		ColorStop{Color: gradientTestColor(200), Position: 1})
	if got := sampleGradient(grad, -1); got != gradientTestColor(10) {
		t.Fatalf("low sample = %+v", got)
	}
	if got := sampleGradient(grad, 2); got != gradientTestColor(200) {
		t.Fatalf("high sample = %+v", got)
	}
	if got := sampleGradient(grad, 0.5); got.R != 105 {
		t.Fatalf("mid sample = %+v", got)
	}
}

// TestLinearTCardinal verifies projection for named directions.
func TestLinearTCardinal(t *testing.T) {
	base := []ColorStop{{Color: gradientTestColor(0), Position: 0}, {Color: gradientTestColor(255), Position: 1}}
	cases := []struct {
		dir      GradientDirection
		u, v     float32
		expected float32
	}{
		{GradientToBottom, 0.3, 0.25, 0.25},
		{GradientToTop, 0.3, 0.25, 0.75},
		{GradientToRight, 0.25, 0.7, 0.25},
		{GradientToLeft, 0.25, 0.7, 0.75},
		{GradientToBottomRight, 0, 0, 0},
		{GradientToBottomRight, 1, 1, 1},
		{GradientToBottomRight, 0.5, 0.5, 0.5},
	}
	for _, tc := range cases {
		grad, _ := NewLinearGradient(tc.dir, base...)
		if got := linearT(grad, tc.u, tc.v); math.Abs(float64(got-tc.expected)) > 1e-4 {
			t.Fatalf("dir %v at (%v,%v) = %v, want %v", tc.dir, tc.u, tc.v, got, tc.expected)
		}
	}
}

// TestLinearTAngle verifies CSS angles map to the right vectors.
func TestLinearTAngle(t *testing.T) {
	base := []ColorStop{{Color: gradientTestColor(0), Position: 0}, {Color: gradientTestColor(255), Position: 1}}
	top, _ := NewAngleGradient(0, base...)
	if got := linearT(top, 0.5, 0.25); math.Abs(float64(got-0.75)) > 1e-4 {
		t.Fatalf("0deg = %v, want 0.75", got)
	}
	right, _ := NewAngleGradient(90, base...)
	if got := linearT(right, 0.25, 0.5); math.Abs(float64(got-0.25)) > 1e-4 {
		t.Fatalf("90deg = %v, want 0.25", got)
	}
	down, _ := NewAngleGradient(180, base...)
	if got := linearT(down, 0.5, 0.25); math.Abs(float64(got-0.25)) > 1e-4 {
		t.Fatalf("180deg = %v, want 0.25", got)
	}
}

// TestNewRadialGradientValidatesCenter verifies centers stay in the square.
func TestNewRadialGradientValidatesCenter(t *testing.T) {
	stops := []ColorStop{{Color: gradientTestColor(0), Position: 0}, {Color: gradientTestColor(255), Position: 1}}
	if _, ok := NewRadialGradient(0.3, 0.2, stops...); !ok {
		t.Fatal("interior center must build")
	}
	if _, ok := NewRadialGradient(-0.1, 0.5, stops...); ok {
		t.Fatal("negative center must fail")
	}
	if _, ok := NewRadialGradient(0.5, 1.5, stops...); ok {
		t.Fatal("oversize center must fail")
	}
}

// TestRadialMaxDist verifies farthest-corner distances.
func TestRadialMaxDist(t *testing.T) {
	if got := RadialMaxDist(0.5, 0.5); math.Abs(float64(got)-math.Sqrt(0.5)) > 1e-4 {
		t.Fatalf("center max = %v", got)
	}
	if got := RadialMaxDist(0, 0); math.Abs(float64(got)-math.Sqrt(2)) > 1e-4 {
		t.Fatalf("corner max = %v", got)
	}
}

// TestSampleRadialAt verifies center maps to 0 and far corners to 1.
func TestSampleRadialAt(t *testing.T) {
	grad, _ := NewRadialGradient(0.5, 0.5,
		ColorStop{Color: gradientTestColor(255), Position: 0},
		ColorStop{Color: gradientTestColor(0), Position: 1})
	maxDist := RadialMaxDist(0.5, 0.5)
	if got := SampleRadialAt(grad, 0.5, 0.5, maxDist); got != gradientTestColor(255) {
		t.Fatalf("center = %+v", got)
	}
	if got := SampleRadialAt(grad, 0, 0, maxDist); got != gradientTestColor(0) {
		t.Fatalf("corner = %+v", got)
	}
}

// TestSampleAtDispatchesKind verifies linear and radial sampling agree.
func TestSampleAtDispatchesKind(t *testing.T) {
	linear, _ := NewLinearGradient(GradientToBottom,
		ColorStop{Color: gradientTestColor(0), Position: 0},
		ColorStop{Color: gradientTestColor(255), Position: 1})
	if got := sampleAt(linear, 0.2, 0.75); got.R != 191 {
		t.Fatalf("linear at = %+v", got)
	}
	radial, _ := NewRadialGradient(0, 0,
		ColorStop{Color: gradientTestColor(255), Position: 0},
		ColorStop{Color: gradientTestColor(0), Position: 1})
	if got := sampleAt(radial, 0, 0); got != gradientTestColor(255) {
		t.Fatalf("radial at center = %+v", got)
	}
}
