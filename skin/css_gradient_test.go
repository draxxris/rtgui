package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestParseCSSMultiStopLinear verifies three stops with mixed positions.
func TestParseCSSMultiStopLinear(t *testing.T) {
	rules, err := ParseCSS(`Button { background-image: linear-gradient(to bottom, #000000 0%, #808080, #ffffff 100%); }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].GradientCount != 1 {
		t.Fatalf("expected single gradient, got %+v", rules)
	}
	grad := rules[0].Gradients[0]
	if grad.StopCount != 3 {
		t.Fatalf("stop count = %d", grad.StopCount)
	}
	if grad.Stops[1].Position != 0.5 {
		t.Fatalf("middle position = %v", grad.Stops[1].Position)
	}
}

// TestParseCSSAngleGradient verifies degree heads select angled fills.
func TestParseCSSAngleGradient(t *testing.T) {
	rules, err := ParseCSS(`Frame { background-image: linear-gradient(135deg, #112233, #445566); }`)
	if err != nil {
		t.Fatal(err)
	}
	grad := rules[0].Gradients[0]
	if !grad.UseAngle || grad.AngleDeg != 135 {
		t.Fatalf("angle = %+v", grad)
	}
	if grad.StopCount != 2 {
		t.Fatalf("stop count = %d", grad.StopCount)
	}
}

// TestParseCSSRadialGradient verifies center and stops parse.
func TestParseCSSRadialGradient(t *testing.T) {
	rules, err := ParseCSS(`Tooltip { background-image: radial-gradient(circle at 30% 20%, #3a5a7a80, #00000000 60%); }`)
	if err != nil {
		t.Fatal(err)
	}
	grad := rules[0].Gradients[0]
	if grad.Kind != GradientRadial {
		t.Fatalf("kind = %v", grad.Kind)
	}
	if grad.CenterX != 0.3 || grad.CenterY != 0.2 {
		t.Fatalf("center = %v %v", grad.CenterX, grad.CenterY)
	}
	if grad.StopCount != 2 || grad.Stops[1].Position != 0.6 {
		t.Fatalf("stops = %+v", grad.Stops)
	}
}

// TestParseCSSInnerGradient verifies one layer reads as four borders at
// once, with stops running border-to-center over the linear base.
func TestParseCSSInnerGradient(t *testing.T) {
	rules, err := ParseCSS(`List { background-image: inner-gradient(#7fb2f0B0, #1e3a5a00 70%), linear-gradient(to bottom, #3a6a9a, #1e3a5a); }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].GradientCount != 2 {
		t.Fatalf("expected inner over linear, got %+v", rules)
	}
	top := rules[0].Gradients[0]
	if top.Kind != GradientInner {
		t.Fatalf("top kind = %v", top.Kind)
	}
	if top.StopCount != 2 || top.Stops[0].Position != 0 || top.Stops[1].Position != 0.7 {
		t.Fatalf("inner stops = %+v", top.Stops)
	}
	if rules[0].Gradients[1].Kind != GradientLinear {
		t.Fatalf("base = %+v", rules[0].Gradients[1])
	}
	for _, text := range []string{
		`Button { background-image: inner-gradient(#112233); }`,
		`Button { background-image: inner-gradient(); }`,
		`Button { background-image: inner-gradient(#112233, #445566, #778899, #aabbcc, #ddeeff); }`,
	} {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("expected inner-gradient error for %q", text)
		}
	}
}

// TestParseCSSRadialDefaultCenter verifies bare circle centers at half.
func TestParseCSSRadialDefaultCenter(t *testing.T) {
	rules, err := ParseCSS(`Tooltip { background-image: radial-gradient(circle, #ffffff, #000000); }`)
	if err != nil {
		t.Fatal(err)
	}
	grad := rules[0].Gradients[0]
	if grad.CenterX != 0.5 || grad.CenterY != 0.5 {
		t.Fatalf("center = %v %v", grad.CenterX, grad.CenterY)
	}
}

// TestParseCSSLayeredGradients verifies first layer lands on top.
func TestParseCSSLayeredGradients(t *testing.T) {
	rules, err := ParseCSS(`Tooltip { background-image: radial-gradient(circle at 30% 20%, #3a5a7a80, #00000000 60%), linear-gradient(to bottom, #2b3d54 0%, #0b1524 100%); }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].GradientCount != 2 {
		t.Fatalf("expected layered gradients, got %+v", rules)
	}
	if rules[0].Gradients[0].Kind != GradientRadial {
		t.Fatalf("top kind = %v", rules[0].Gradients[0].Kind)
	}
	base := rules[0].Gradients[1]
	if base.Kind != GradientLinear || base.Direction != GradientToBottom {
		t.Fatalf("base = %+v", base)
	}
	_ = core.Color{}
}

// TestParseCSSThreeLayers verifies stacks beyond two layers parse in order.
func TestParseCSSThreeLayers(t *testing.T) {
	rules, err := ParseCSS(`Frame { background-image: radial-gradient(circle at 20% 20%, #ffffff80, #00000000 50%), linear-gradient(135deg, #112233 0%, #445566 50%, #778899 100%), linear-gradient(to bottom, #000000, #ffffff); }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].GradientCount != 3 {
		t.Fatalf("expected three layers, got %+v", rules)
	}
	if rules[0].Gradients[0].Kind != GradientRadial {
		t.Fatalf("top kind = %v", rules[0].Gradients[0].Kind)
	}
	mid := rules[0].Gradients[1]
	if !mid.UseAngle || mid.StopCount != 3 {
		t.Fatalf("middle = %+v", mid)
	}
	if rules[0].Gradients[2].Direction != GradientToBottom {
		t.Fatalf("base = %+v", rules[0].Gradients[2])
	}
}

// TestParseCSSLayeredGradientErrors verifies bad layers fail loudly.
func TestParseCSSLayeredGradientErrors(t *testing.T) {
	cases := []string{
		`Button { background-image: linear-gradient(#112233, #445566, #778899, #aabbcc, #ddeeff); }`,
		`Button { background-image: radial-gradient(circle at 30%, #112233, #445566); }`,
		`Button { background-image: radial-gradient(square, #112233, #445566); }`,
		`Button { background-image: linear-gradient(135turn, #112233, #445566); }`,
		`Button { background-image: linear-gradient(#112233, #445566), linear-gradient(#112233, #445566), linear-gradient(#112233, #445566), linear-gradient(#112233, #445566), linear-gradient(#112233, #445566); }`,
		`Button { background-image: conic-gradient(#112233, #445566); }`,
	}
	for _, text := range cases {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("expected error for %q", text)
		}
	}
}

// TestParseCSSNoneClearsOverlay verifies none drops both gradient layers.
func TestParseCSSNoneClearsOverlay(t *testing.T) {
	rules, err := ParseCSS(`Button { background-image: none; }`)
	if err != nil {
		t.Fatal(err)
	}
	if rules[0].HasGradient() || !rules[0].NoTexture {
		t.Fatalf("none must clear both layers, got %+v", rules[0])
	}
}
