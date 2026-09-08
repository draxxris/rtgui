package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
)

// TestInlineIconLayoutEmitsAtomicBox checks icon fragments wrap atomically.
func TestInlineIconLayoutEmitsAtomicBox(t *testing.T) {
	theme := richLayoutTheme()
	segments := []core.RichSegment{
		{Text: "a "},
		{Icon: "iron-plate", HasIcon: true},
		{Text: " b"},
	}
	spans := theme.LayoutRichSpans(core.Rect{X: 0, Y: 0, W: 300, H: 200}, segments, core.StateNormal)
	icon := 0
	for _, span := range spans {
		if span.IsIcon {
			icon++
			if span.Icon != "iron-plate" || span.Text != "" {
				t.Fatalf("icon span = %+v", span)
			}
			if span.Bounds.W <= 0 || span.Bounds.H != RichLineHeight {
				t.Fatalf("icon bounds = %+v", span.Bounds)
			}
		}
	}
	if icon != 1 {
		t.Fatalf("icon fragments = %d, spans %+v", icon, spans)
	}
}

// TestRegisteredLinkColorWinsOverDefault checks parent tint resolution.
func TestRegisteredLinkColorWinsOverDefault(t *testing.T) {
	theme := richLayoutTheme()
	span := RichSpanLayout{Link: core.Link{Kind: core.LinkItem, Target: "x"}}
	if got := theme.richFragmentTint(span); got != richLinkText {
		t.Fatalf("default link tint = %+v", got)
	}
	custom := core.Color{R: 255, G: 180, B: 70, A: 255}
	theme.SetLinkColor(core.LinkItem, custom)
	if got := theme.richFragmentTint(span); got != custom.RGBA() {
		t.Fatalf("registered tint = %+v", got)
	}
	explicit := RichSpanLayout{Link: core.Link{Kind: core.LinkItem}, HasColor: true, Color: core.Color{R: 1, G: 2, B: 3, A: 255}}
	if got := theme.richFragmentTint(explicit); got != explicit.Color.RGBA() {
		t.Fatalf("explicit tint = %+v", got)
	}
	theme.ClearLinkColor(core.LinkItem)
	if _, ok := theme.LinkColor(core.LinkItem); ok {
		t.Fatal("cleared link color survived")
	}
}

// TestInlineIconRegistryRoundTrips checks whitelist storage semantics.
func TestInlineIconRegistryRoundTrips(t *testing.T) {
	theme := richLayoutTheme()
	theme.RegisterInlineIcon("iron-plate", core.TooltipIcon{Width: 32, Height: 32})
	icon, ok := theme.LookupInlineIcon("iron-plate")
	if !ok || icon.Width != 32 {
		t.Fatalf("lookup = %+v/%v", icon, ok)
	}
	theme.UnregisterInlineIcon("iron-plate")
	if _, ok := theme.LookupInlineIcon("iron-plate"); ok {
		t.Fatal("unregistered icon survived")
	}
	var nilTheme *Theme
	nilTheme.RegisterInlineIcon("x", core.TooltipIcon{})
	if _, ok := nilTheme.LookupInlineIcon("x"); ok {
		t.Fatal("nil theme registry wrote")
	}
}

// TestSingleLineLayoutAligns checks centered single-line runs.
func TestSingleLineLayoutAligns(t *testing.T) {
	theme := richLayoutTheme()
	content := core.Rect{X: 0, Y: 0, W: 300, H: 40}
	left := theme.LayoutRichSingleLine(content, []core.RichSegment{{Text: "hi"}}, core.AlignLeft, nil)
	center := theme.LayoutRichSingleLine(content, []core.RichSegment{{Text: "hi"}}, core.AlignCenter, nil)
	if len(left) == 0 || len(center) == 0 {
		t.Fatal("single-line spans empty")
	}
	if center[0].Bounds.X <= left[0].Bounds.X {
		t.Fatalf("center %v not past left %v", center[0].Bounds.X, left[0].Bounds.X)
	}
	withIcon := theme.LayoutRichSingleLine(content, []core.RichSegment{{Icon: "iron-plate", HasIcon: true}, {Text: "hi"}}, core.AlignLeft, nil)
	if len(withIcon) != 2 || !withIcon[0].IsIcon {
		t.Fatalf("single-line icon = %+v", withIcon)
	}
}

// TestStyledSpansCarrySizeAndFace checks per-segment style flows to spans.
func TestStyledSpansCarrySizeAndFace(t *testing.T) {
	theme := richLayoutTheme()
	bounds := core.Rect{X: 0, Y: 0, W: 600, H: 400}
	segments := []core.RichSegment{
		{Text: "plain "},
		{Text: "bold", Bold: true},
		{Text: " big", HasFontSize: true, FontSize: 32},
		{Text: " valley", HasFont: true, Font: "ValleySans"},
	}
	spans := theme.LayoutRichSpans(bounds, segments, core.StateNormal)
	if len(spans) != 4 {
		t.Fatalf("styled spans = %+v", spans)
	}
	if !spans[1].Bold || spans[2].Size != 32 || spans[3].Font != "ValleySans" {
		t.Fatalf("style lost = %+v", spans)
	}
	if spans[2].Bounds.H <= RichLineHeight {
		t.Fatalf("big row not taller than default: %+v", spans[2].Bounds)
	}
	single := theme.LayoutRichSingleLine(core.Rect{X: 0, Y: 0, W: 600, H: 60}, segments, core.AlignLeft, nil)
	if len(single) != 4 || !single[1].Bold || single[2].Size != 32 {
		t.Fatalf("single-line style lost = %+v", single)
	}
}

// TestNamedFontRegistry checks ValleySans registration and clamping.
func TestNamedFontRegistry(t *testing.T) {
	theme := richLayoutTheme()
	if err := theme.RegisterFont("ValleySans", "../testdata/fonts/ValleySans-Regular.ttf"); err != nil {
		t.Fatalf("register ValleySans: %v", err)
	}
	if !theme.HasNamedFont("ValleySans") {
		t.Fatal("ValleySans not registered")
	}
	if err := theme.RegisterFont("bad", "../testdata/fonts/missing.ttf"); err == nil {
		t.Fatal("missing font accepted")
	}
	if got := clampRichFontSize(200); got != RichMaxFontSize {
		t.Fatalf("max clamp = %v", got)
	}
	if got := clampRichFontSize(2); got != RichMinFontSize {
		t.Fatalf("min clamp = %v", got)
	}
	if got := richSpanSize(core.RichSegment{}); got != RichFontSize {
		t.Fatalf("default size = %v", got)
	}
	theme.UnregisterFont("ValleySans")
	if theme.HasNamedFont("ValleySans") {
		t.Fatal("unregistered font survived")
	}
}

// TestNamedFontLifecycle checks overwrite, unregister, and empty names.
func TestNamedFontLifecycle(t *testing.T) {
	theme := richLayoutTheme()
	if err := theme.RegisterFont("", "../testdata/fonts/ValleySans-Regular.ttf"); err == nil {
		t.Fatal("empty font name accepted")
	}
	if err := theme.RegisterFont("ValleySans", "../testdata/fonts/ValleySans-Regular.ttf"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := theme.RegisterFont("ValleySans", "../testdata/fonts/ValleySans-Regular.ttf"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if !theme.HasNamedFont("ValleySans") {
		t.Fatal("overwrite dropped registration")
	}
	theme.UnregisterFont("ValleySans")
	if theme.HasNamedFont("ValleySans") {
		t.Fatal("unregistered font survived")
	}
	theme.UnregisterFont("ValleySans")
}

// TestSingleLineDrawSteadyStateAllocatesNothing covers warmed button draws.
func TestSingleLineDrawSteadyStateAllocatesNothing(t *testing.T) {
	theme := richLayoutTheme()
	info := core.WidgetInfo{Name: "ok", Bounds: core.Rect{W: 200, H: 40}, Kind: core.WidgetButton, State: core.StateNormal}
	content := core.Rect{X: 0, Y: 0, W: 200, H: 40}
	segments := []core.RichSegment{{Text: "hello world"}, {Text: " bold", Bold: true}}
	theme.DrawRichSingleLine(info, content, segments, core.AlignLeft)
	if allocations := testing.AllocsPerRun(200, func() {
		theme.DrawRichSingleLine(info, content, segments, core.AlignLeft)
	}); allocations != 0 {
		t.Fatalf("single-line steady-state allocations = %v, want 0", allocations)
	}
}
