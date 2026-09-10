package skin

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

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
		"Slider::track":            {"Slider", PartTrack},
		"Slider::thumb":            {"Slider", PartThumb},
		"Dropdown::arrow":          {"Dropdown", PartArrow},
		"Checkbox::checkmark":      {"Checkbox", PartCheckmark},
		"Checkbox::box":            {"Checkbox", PartIcon},
		"Slider::thumb:hover":      {"Slider", PartThumb},
		"Checkbox::box:disabled":   {"Checkbox", PartIcon},
		"ProgressBar::fill":        {"ProgressBar", PartOverlay},
		"Dropdown::highlight":      {"Dropdown", PartOverlay},
		"ScrollPanel::track":       {"ScrollPanel", PartTrack},
		"ScrollPanel::thumb":       {"ScrollPanel", PartThumb},
		"ScrollPanel::thumb:hover": {"ScrollPanel", PartThumb},
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
		`ProgressBar::fill { border-image-source: url("a.png"); }`,
		`Dropdown::highlight { padding: 2; }`,
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

// assertSliceRule verifies a parsed SkinRule matches the expected kind, part, and slice.
func assertSliceRule(t *testing.T, rule SkinRule, kind core.WidgetKind, part SkinPart, slice int32) {
	t.Helper()
	if rule.Kind != kind || rule.Part != part || rule.Slice != slice {
		t.Fatalf("slice rule mismatch: %+v", rule)
	}
}

// TestParseScrollbarCSS verifies ScrollPanel track and thumb declarations with slicing.
func TestParseScrollbarCSS(t *testing.T) {
	rules, err := ParseCSS(`
		ScrollPanel {
			border-image-source: url("panel.png");
			border-image-slice: 8px;
			padding: 8px;
		}
		ScrollPanel::track {
			border-image-source: url("track.png");
			border-image-slice: 8px;
		}
		ScrollPanel::thumb {
			border-image-source: url("thumb.png");
			border-image-slice: 8px;
		}
		ScrollPanel::thumb:hover {
			border-image-source-tint: #E1F2FF;
		}
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 5 {
		t.Fatalf("expected 5 rules, got %d", len(rules))
	}
	assertSliceRule(t, rules[0], core.WidgetScrollPanel, PartBorder, 8)
	if rules[1].Kind != core.WidgetScrollPanel || rules[1].Part != PartBackground || !rules[1].HasPadding || rules[1].Padding[0] != 8 {
		t.Fatalf("padding rule mismatch: %+v", rules[1])
	}
	assertSliceRule(t, rules[2], core.WidgetScrollPanel, PartTrack, 8)
	assertSliceRule(t, rules[3], core.WidgetScrollPanel, PartThumb, 8)
	if rules[4].Kind != core.WidgetScrollPanel || rules[4].Part != PartThumb || rules[4].State != core.StateHovered || !rules[4].HasTint {
		t.Fatalf("thumb hover rule mismatch: %+v", rules[4])
	}
}

// TestParseCSSBackgroundColor verifies background-color hex parsing with and without alpha.
func TestParseCSSBackgroundColor(t *testing.T) {
	rules, err := ParseCSS(`
		Button { background-color: #334455; }
		Frame { background-color: #AABBCC80; }
		Slider::track { background-color: #112233; }
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}
	if !rules[0].HasBackgroundColor || rules[0].BackgroundColor != (core.Color{R: 0x33, G: 0x44, B: 0x55, A: 255}) {
		t.Fatalf("Button background-color mismatch: %+v", rules[0])
	}
	if !rules[1].HasBackgroundColor || rules[1].BackgroundColor != (core.Color{R: 0xaa, G: 0xbb, B: 0xcc, A: 0x80}) {
		t.Fatalf("Frame background-color mismatch: %+v", rules[1])
	}
	if rules[2].Part != PartTrack || !rules[2].HasBackgroundColor || rules[2].BackgroundColor != (core.Color{R: 0x11, G: 0x22, B: 0x33, A: 255}) {
		t.Fatalf("Slider::track background-color mismatch: %+v", rules[2])
	}
}

// TestParseCSSLinearGradientDefault verifies linear-gradient default direction and stops.
func TestParseCSSLinearGradientDefault(t *testing.T) {
	rules, err := ParseCSS(`Button { background-image: linear-gradient(#112233, #445566); }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || !rules[0].HasGradient() || rules[0].Gradients[0].Direction != GradientToBottom {
		t.Fatalf("expected GradientToBottom, got %+v", rules)
	}
	grad := rules[0].Gradients[0]
	stops := grad.Stops
	if stops[0].Color != (core.Color{R: 0x11, G: 0x22, B: 0x33, A: 255}) || stops[1].Color != (core.Color{R: 0x44, G: 0x55, B: 0x66, A: 255}) {
		t.Fatalf("unexpected stops: %+v", stops)
	}
}

// TestParseCSSLinearGradientCardinal verifies cardinal direction linear-gradient with alpha stops.
func TestParseCSSLinearGradientCardinal(t *testing.T) {
	rules, err := ParseCSS(`Frame { background-image: linear-gradient(to right, #11223380, #445566CC); }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || !rules[0].HasGradient() || rules[0].Gradients[0].Direction != GradientToRight {
		t.Fatalf("expected GradientToRight, got %+v", rules)
	}
	grad := rules[0].Gradients[0]
	stops := grad.Stops
	if stops[0].Color.A != 0x80 || stops[1].Color.A != 0xcc {
		t.Fatalf("expected alpha stops, got %+v", stops)
	}
}

// TestParseCSSLinearGradientCorner verifies diagonal corner linear-gradient with positions.
func TestParseCSSLinearGradientCorner(t *testing.T) {
	rules, err := ParseCSS(`Slider::thumb { background-image: linear-gradient(to bottom right, #112233 0%, #445566 100%); }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || !rules[0].HasGradient() || rules[0].Gradients[0].Direction != GradientToBottomRight {
		t.Fatalf("expected GradientToBottomRight, got %+v", rules)
	}
	grad := rules[0].Gradients[0]
	stops := grad.Stops
	if stops[0].Position != 0.0 || stops[1].Position != 1.0 {
		t.Fatalf("expected percentage positions, got %+v", stops)
	}
}

// TestParseCSSLinearGradientToTop verifies to top linear-gradient direction.
func TestParseCSSLinearGradientToTop(t *testing.T) {
	rules, err := ParseCSS(`ProgressBar::fill { background-image: linear-gradient(to top, #001122, #334455); }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || !rules[0].HasGradient() || rules[0].Gradients[0].Direction != GradientToTop {
		t.Fatalf("expected GradientToTop, got %+v", rules)
	}
}

// TestParseCSSLinearGradientErrors verifies malformed gradient syntax produces hard errors.
func TestParseCSSLinearGradientErrors(t *testing.T) {
	cases := []string{
		`Button { background-image: linear-gradient(); }`,
		`Button { background-image: linear-gradient(#112233); }`,
		`Button { background-image: linear-gradient(to nowhere, #112233, #445566); }`,
		`Button { background-image: linear-gradient(to bottom, red, #445566); }`,
		`Button { background-image: linear-gradient(to bottom, #112233, blue); }`,
		`Button { background-image: linear-gradient(to bottom, #112233 150%, #445566); }`,
		`Button { background-image: linear-gradient(to bottom, #112233, #445566, #778899, #aabbcc, #ddeeff); }`,
		`Button { background-color: red; }`,
		`Button { background-color: #123; }`,
		`Button { background-color: #12345; }`,
	}
	for _, text := range cases {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("expected error for %q", text)
		}
	}
}

// TestParseCSSBackgroundImageNone verifies none clears image and gradient declarations.
func TestParseCSSBackgroundImageNone(t *testing.T) {
	rules, err := ParseCSS(`Button { background-image: none; }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].HasImage || rules[0].HasGradient() || !rules[0].NoTexture {
		t.Fatalf("expected clear with no image or gradient, got %+v", rules[0])
	}
}

// TestParseCSSBorderImageSourceNone verifies none drops the inherited ring.
func TestParseCSSBorderImageSourceNone(t *testing.T) {
	rules, err := ParseCSS(`Frame.titled-titlebar { border-image-source: none; }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].HasImage || !rules[0].NoTexture {
		t.Fatalf("expected border clear with no image, got %+v", rules[0])
	}
	if rules[0].Kind != core.WidgetFrame || rules[0].Class != "titled-titlebar" || rules[0].Part != PartBorder {
		t.Fatalf("none rule key = %+v", rules[0])
	}
}

// TestParseCSSTitledVariantSelectors verifies the five TitledFrame class
// selectors plus pseudos parse to their widget kinds.
func TestParseCSSTitledVariantSelectors(t *testing.T) {
	cases := []struct {
		css  string
		kind core.WidgetKind
	}{
		{`Frame.titled-frame { padding: 8; }`, core.WidgetFrame},
		{`Frame.titled-frame:focus { padding: 8; }`, core.WidgetFrame},
		{`Frame.titled-frame:disabled { padding: 8; }`, core.WidgetFrame},
		{`Frame.titled-titlebar { background-color: #24354c; }`, core.WidgetFrame},
		{`Label.titled-title { color: #e6f0ff; }`, core.WidgetLabel},
		{`Button.titled-close { background-color: #5c1d24; }`, core.WidgetButton},
		{`Button.titled-close:hover { background-color: #8b2635; }`, core.WidgetButton},
		{`Button.titled-close:active { background-color: #3b1015; }`, core.WidgetButton},
		{`Frame.titled-content { padding: 8; }`, core.WidgetFrame},
	}
	for _, tc := range cases {
		rules, err := ParseCSS(tc.css)
		if err != nil {
			t.Fatalf("%s: %v", tc.css, err)
		}
		if len(rules) == 0 || rules[0].Kind != tc.kind || rules[0].Class == "" {
			t.Fatalf("%s parsed as %+v", tc.css, rules)
		}
	}
}

// TestParseCSSClassSelectors verifies universal and kind-scoped class selectors.
func TestParseCSSClassSelectors(t *testing.T) {
	cases := []struct {
		css       string
		wantKind  core.WidgetKind
		wantClass string
		wantState core.WidgetState
	}{
		{".danger { padding: 4; }", core.WidgetAny, "danger", core.StateNormal},
		{".danger:hover { padding: 6; }", core.WidgetAny, "danger", core.StateHovered},
		{".danger:active { padding: 2; }", core.WidgetAny, "danger", core.StatePressed},
		{"Button.primary { padding: 8; }", core.WidgetButton, "primary", core.StateNormal},
		{"Button.primary:hover { padding: 10; }", core.WidgetButton, "primary", core.StateHovered},
		{"Slider.special::thumb { background-color: #112233; }", core.WidgetSlider, "special", core.StateNormal},
	}
	for _, tc := range cases {
		rules, err := ParseCSS(tc.css)
		if err != nil {
			t.Fatalf("%s: %v", tc.css, err)
		}
		if len(rules) == 0 {
			t.Fatalf("%s: no rules parsed", tc.css)
		}
		if rules[0].Kind != tc.wantKind || rules[0].Class != tc.wantClass || rules[0].State != tc.wantState {
			t.Fatalf("%s: got kind=%v class=%q state=%v, want kind=%v class=%q state=%v",
				tc.css, rules[0].Kind, rules[0].Class, rules[0].State, tc.wantKind, tc.wantClass, tc.wantState)
		}
	}
}

// TestParseCSSClassErrors verifies invalid class syntax produces descriptive hard errors.
func TestParseCSSClassErrors(t *testing.T) {
	cases := []string{
		`. { padding: 1; }`,
		`Button. { padding: 1; }`,
		`.123 { padding: 1; }`,
		`Button.123 { padding: 1; }`,
		`.foo bar { padding: 1; }`,
		`Unknown.danger { padding: 1; }`,
		`.danger::thumb { background-color: #123456; }`,
	}
	for _, text := range cases {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("expected error for %q", text)
		}
	}
}

// TestParseCSSUniversalSelector verifies '*' and '*:pseudo' selectors parse to WidgetAny with empty class.
func TestParseCSSUniversalSelector(t *testing.T) {
	cases := []struct {
		css       string
		wantKind  core.WidgetKind
		wantClass string
		wantState core.WidgetState
	}{
		{"* { padding: 4; }", core.WidgetAny, "", core.StateNormal},
		{"*:hover { padding: 6; }", core.WidgetAny, "", core.StateHovered},
		{"*:active { padding: 2; }", core.WidgetAny, "", core.StatePressed},
		{"*.danger { padding: 8; }", core.WidgetAny, "danger", core.StateNormal},
	}
	for _, tc := range cases {
		rules, err := ParseCSS(tc.css)
		if err != nil {
			t.Fatalf("%s: %v", tc.css, err)
		}
		if len(rules) == 0 {
			t.Fatalf("%s: no rules parsed", tc.css)
		}
		if rules[0].Kind != tc.wantKind || rules[0].Class != tc.wantClass || rules[0].State != tc.wantState {
			t.Fatalf("%s: got kind=%v class=%q state=%v, want kind=%v class=%q state=%v",
				tc.css, rules[0].Kind, rules[0].Class, rules[0].State, tc.wantKind, tc.wantClass, tc.wantState)
		}
	}
}

// TestParseCSSTextAndFontProperties verifies color, font-size, font-family, and font-italic-family declarations.
func TestParseCSSTextAndFontProperties(t *testing.T) {
	cssText := `
		* {
			color: #aabbcc;
			font-size: 18px;
			font-family: url("../fonts/Custom.ttf");
			font-italic-family: url("../fonts/Custom-Italic.ttf");
		}
		Button {
			font-family: "ValleySans";
			color: #11223344;
		}
	`
	rules, err := ParseCSS(cssText)
	if err != nil {
		t.Fatalf("ParseCSS failed: %v", err)
	}
	if len(rules) < 2 {
		t.Fatalf("expected at least 2 rules, got %d", len(rules))
	}
	assertUniversalTextRule(t, rules[0])
	assertButtonTextRule(t, rules[1])
}

// assertUniversalTextRule checks parsed declarations for the universal selector rule.
func assertUniversalTextRule(t *testing.T, r SkinRule) {
	if !r.HasTextColor || r.TextColor != (core.Color{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}) {
		t.Errorf("r0 text color mismatch: %+v", r)
	}
	if !r.HasFontSize || r.FontSize != 18 {
		t.Errorf("r0 font size mismatch: %+v", r)
	}
	if !r.HasFont || r.Font != "../fonts/Custom.ttf" {
		t.Errorf("r0 font mismatch: %+v", r)
	}
	if !r.HasItalicFont || r.ItalicFont != "../fonts/Custom-Italic.ttf" {
		t.Errorf("r0 italic font mismatch: %+v", r)
	}
}

// assertButtonTextRule checks parsed declarations for the Button text rule.
func assertButtonTextRule(t *testing.T, r SkinRule) {
	if !r.HasFont || r.Font != "ValleySans" {
		t.Errorf("r1 font mismatch: %+v", r)
	}
	if !r.HasTextColor || r.TextColor != (core.Color{R: 0x11, G: 0x22, B: 0x33, A: 0x44}) {
		t.Errorf("r1 text color mismatch: %+v", r)
	}
}

// TestParseCSSTextPropertyErrors verifies errors on invalid text or font syntax.
func TestParseCSSTextPropertyErrors(t *testing.T) {
	cases := []string{
		`*::part { color: #fff; }`,
		`* { font-size: -5px; }`,
		`* { font-size: 0; }`,
		`* { font-size: abc; }`,
		`* { font-family: ""; }`,
		`* { color: not-a-color; }`,
	}
	for _, text := range cases {
		if _, err := ParseCSS(text); err == nil {
			t.Fatalf("expected error for %q", text)
		}
	}
}
