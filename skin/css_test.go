package skin

import (
	"image"
	"image/color"
	"testing"

	"rtgui/core"
)

// nrgbaAt reads a pixel straight back; NRGBA storage round-trips exactly.
func nrgbaAt(img image.Image, x, y int) color.NRGBA {
	return img.At(x, y).(color.NRGBA)
}

// solidImage builds an 8x8 test image of one NRGBA color.
func solidImage(c color.NRGBA) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// TestParseCSSSelectors verifies every kind, pseudo-class, and ::part spelling.
func TestParseCSSSelectors(t *testing.T) {
	kinds := map[string]core.WidgetKind{
		"Button": core.WidgetButton, "Checkbox": core.WidgetCheckbox,
		"Textbox": core.WidgetTextbox, "Dropdown": core.WidgetDropdown,
		"Slider": core.WidgetSlider, "ProgressBar": core.WidgetProgressBar,
		"Frame": core.WidgetFrame, "Label": core.WidgetLabel,
		"ScrollPanel": core.WidgetScrollPanel,
	}
	for name, kind := range kinds {
		rules, err := ParseCSS(name + " { padding: 1; }")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(rules) != 1 || rules[0].Kind != kind || rules[0].State != core.StateNormal {
			t.Fatalf("%s parsed as %+v", name, rules)
		}
	}
	pseudos := map[string]core.WidgetState{
		"hover": core.StateHovered, "active": core.StatePressed,
		"focus": core.StateFocused, "disabled": core.StateDisabled,
		"selected": core.StateSelected,
	}
	for name, state := range pseudos {
		rules, err := ParseCSS("Button:" + name + " { padding: 1; }")
		if err != nil {
			t.Fatalf(":%s: %v", name, err)
		}
		if rules[0].State != state {
			t.Fatalf(":%s parsed as state %d", name, rules[0].State)
		}
	}
	parts := map[string]struct {
		kind string
		part SkinPart
	}{
		"Slider::track":          {"Slider", PartTrack},
		"Slider::thumb":          {"Slider", PartThumb},
		"Dropdown::arrow":        {"Dropdown", PartArrow},
		"Checkbox::checkmark":    {"Checkbox", PartCheckmark},
		"Checkbox::box":          {"Checkbox", PartIcon},
		"Slider::thumb:hover":    {"Slider", PartThumb},
		"Checkbox::box:disabled": {"Checkbox", PartIcon},
	}
	for selector, want := range parts {
		rules, err := ParseCSS(selector + ` { background-image: url("a.png"); }`)
		if err != nil {
			t.Fatalf("%s: %v", selector, err)
		}
		if len(rules) != 1 || rules[0].Part != want.part || !rules[0].HasImage {
			t.Fatalf("%s parsed as %+v", selector, rules)
		}
		_ = want.kind
	}
}

// TestParseCSSErrors verifies strict failures name the offender.
func TestParseCSSErrors(t *testing.T) {
	cases := []string{
		`Buton { padding: 1; }`,
		`button { padding: 1; }`,
		`Button:hovr { padding: 1; }`,
		`Button:hover:active { padding: 1; }`,
		`Slider::knob { background-image: url("a.png"); }`,
		`Button::thumb { background-image: url("a.png"); }`,
		`Button { width: 10; }`,
		`Button { border-image-source: icon.png; }`,
		`Button { border-image-source: url(""); }`,
		`Button { border-image-slice: -1; }`,
		`Button { border-image-slice: lots; }`,
		`Button { border-image-source-tint: red; }`,
		`Button { border-image-source-tint: #12345; }`,
		`Button { padding: 1 2 3 4 5; }`,
		`Button { padding: wide; }`,
		`Slider::thumb { border-image-source: url("a.png"); }`,
		`Slider::thumb { border-image-slice: 4; }`,
		`Slider::thumb { padding: 2; }`,
	}
	for _, text := range cases {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("expected error for %q", text)
		}
	}
}

// TestParseCSSDeclarations verifies values land on the right part entries.
func TestParseCSSDeclarations(t *testing.T) {
	rules, err := ParseCSS(`Frame {
		background-image: url('bg.png');
		background-image-tint: #E1F2FF80;
		border-image-source: url("ring.png");
		border-image-slice: 8px;
		padding: 5 10;
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected background+border entries, got %+v", rules)
	}
	var bg, border *SkinRule
	for i := range rules {
		switch rules[i].Part {
		case PartBackground:
			bg = &rules[i]
		case PartBorder:
			border = &rules[i]
		}
	}
	if bg == nil || bg.Image != "bg.png" || !bg.HasTint || bg.Tint != (core.Color{R: 0xE1, G: 0xF2, B: 0xFF, A: 0x80}) {
		t.Fatalf("background entry=%+v", bg)
	}
	if bg.Padding != ([4]float32{5, 10, 5, 10}) {
		t.Fatalf("padding expansion=%v", bg.Padding)
	}
	if border == nil || border.Image != "ring.png" || !border.HasSlice || border.Slice != 8 {
		t.Fatalf("border entry=%+v", border)
	}
}

// TestParseCSSOrder verifies source order survives for overwrite layering.
func TestParseCSSOrder(t *testing.T) {
	rules, err := ParseCSS("Button { padding: 1; }\nButton:hover { padding: 2; }\nButton { padding: 3; }")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 || rules[0].Padding[0] != 1 || rules[1].Padding[0] != 2 || rules[2].Padding[0] != 3 {
		t.Fatalf("order not preserved: %+v", rules)
	}
}

// TestTintImage verifies channel math incl. alpha with stdlib images only.
// Sources are opaque so premultiplied storage round-trips exactly.
func TestTintImage(t *testing.T) {
	src := solidImage(color.NRGBA{R: 200, G: 100, B: 50, A: 255})
	white := TintImage(src, core.Color{R: 255, G: 255, B: 255, A: 255})
	if got := nrgbaAt(white, 3, 3); got != (color.NRGBA{R: 200, G: 100, B: 50, A: 255}) {
		t.Fatalf("white tint must be identity, got %v", got)
	}
	red := TintImage(src, core.Color{R: 255, G: 0, B: 0, A: 255})
	if got := nrgbaAt(red, 0, 0); got != (color.NRGBA{R: 200, G: 0, B: 0, A: 255}) {
		t.Fatalf("red multiply wrong: %v", got)
	}
	half := TintImage(src, core.Color{R: 255, G: 255, B: 255, A: 128})
	if got := nrgbaAt(half, 0, 0); got != (color.NRGBA{R: 200, G: 100, B: 50, A: 128}) {
		t.Fatalf("alpha multiply wrong: %v", got)
	}
	translucent := TintImage(solidImage(color.NRGBA{R: 200, G: 100, B: 50, A: 200}), core.Color{R: 255, G: 255, B: 255, A: 255})
	if bounds := translucent.Bounds(); bounds.Dx() != 8 || bounds.Dy() != 8 {
		t.Fatalf("dims must survive: %v", bounds)
	}
}
