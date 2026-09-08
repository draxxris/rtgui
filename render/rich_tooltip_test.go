package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

// richTooltipTestTheme builds a theme with a logical viewport for tests.
func richTooltipTestTheme() *Theme {
	return NewTheme(transform.New(core.Viewport{
		Viewport:    core.Rect{W: 800, H: 600},
		LogicalSize: core.Vec2{X: 800, Y: 600},
	}))
}

// richTooltipTestData builds a titled payload with a colored body.
func richTooltipTestData() core.RichTooltip {
	return core.RichTooltip{
		Title:    "Thunderfury",
		Subtitle: "Legendary sword",
		Segments: []core.RichSegment{
			{Text: "Chance on hit to blast your enemy."},
			{Text: " Nature damage.", Color: core.Color{R: 140, G: 190, B: 255, A: 255}, HasColor: true},
		},
	}
}

// TestRichTooltipTitlePlacementFlipsAndClamps checks viewport positioning.
func TestRichTooltipTitlePlacementFlipsAndClamps(t *testing.T) {
	theme := richTooltipTestTheme()
	logical := core.Vec2{X: 800, Y: 600}
	var cache RichTooltipCache
	data := core.RichTooltip{Title: "Hi"}
	if !cache.Update(theme, data, core.Vec2{X: 10, Y: 10}, logical, 0) {
		t.Fatal("first update must rebuild")
	}
	bounds := cache.Bounds()
	if bounds.W <= 0 || bounds.H <= 0 {
		t.Fatalf("title bounds = %+v", bounds)
	}
	if bounds.X < 10 || bounds.Y < 10 {
		t.Fatalf("title not anchored below cursor: %+v", bounds)
	}
	var edge RichTooltipCache
	edge.Update(theme, data, core.Vec2{X: 790, Y: 590}, logical, 0)
	flipped := edge.Bounds()
	if flipped.X+flipped.W > 800 || flipped.Y+flipped.H > 600 {
		t.Fatalf("flipped bounds escape viewport: %+v", flipped)
	}
	if flipped.X >= 790 || flipped.Y >= 590 {
		t.Fatalf("edge tooltip did not flip: %+v", flipped)
	}
	if !edge.Valid() || edge.TitleLineCount() != 1 {
		t.Fatalf("edge cache invalid: valid=%v lines=%d", edge.Valid(), edge.TitleLineCount())
	}
}

// TestRichTooltipBodyWrapsInsideBoundedWidth checks multiline fragments.
func TestRichTooltipBodyWrapsInsideBoundedWidth(t *testing.T) {
	theme := richTooltipTestTheme()
	logical := core.Vec2{X: 800, Y: 600}
	data := core.RichTooltip{
		Title: "Item",
		Segments: []core.RichSegment{
			{Text: "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu"},
		},
		Width: 120,
	}
	var cache RichTooltipCache
	cache.Update(theme, data, core.Vec2{}, logical, 8)
	bounds := cache.Bounds()
	if bounds.W > 120+16+1 {
		t.Fatalf("body width unbounded: %+v", bounds)
	}
	if cache.BodySpanCount() < 2 {
		t.Fatalf("body did not wrap: spans=%d bounds=%+v", cache.BodySpanCount(), bounds)
	}
	for _, span := range cache.bodySpans {
		if span.Bounds.X+span.Bounds.W > bounds.X+bounds.W+1 {
			t.Fatalf("fragment overflows popup: %+v in %+v", span.Bounds, bounds)
		}
		if span.Text == "" {
			t.Fatal("empty body fragment cached")
		}
	}
	content := cache.Content()
	if content.W <= 0 || content.H <= 0 {
		t.Fatalf("content empty: %+v", content)
	}
}

// TestRichTooltipIconReservesTextColumn checks icon geometry.
func TestRichTooltipIconReservesTextColumn(t *testing.T) {
	theme := richTooltipTestTheme()
	logical := core.Vec2{X: 800, Y: 600}
	data := richTooltipTestData()
	data.HasIcon = true
	data.Icon = core.TooltipIcon{ID: 7, Width: 64, Height: 64}
	var cache RichTooltipCache
	cache.Update(theme, data, core.Vec2{X: 20, Y: 20}, logical, 8)
	if !cache.HasIcon() {
		t.Fatal("icon missing from cache")
	}
	dest := cache.IconDest()
	bounds := cache.Bounds()
	if dest.W <= 0 || dest.H <= 0 {
		t.Fatalf("icon dest empty: %+v", dest)
	}
	if dest.X < bounds.X || dest.Y < bounds.Y || dest.X+dest.W > bounds.X+bounds.W+1 {
		t.Fatalf("icon escapes popup: %+v in %+v", dest, bounds)
	}
	wantX := float32(RichTooltipIconSize + RichTooltipIconGap)
	for _, row := range cache.titleRows {
		if row.X != wantX {
			t.Fatalf("title not offset past icon: %v want %v", row.X, wantX)
		}
		if right := row.X + richTooltipMeasure(theme, row.Text, RichTooltipTitleSize); right > cache.Content().W+1 {
			t.Fatalf("icon title clipped: right %v in content %+v", right, cache.Content())
		}
	}
	var noIcon RichTooltipCache
	plain := richTooltipTestData()
	noIcon.Update(theme, plain, core.Vec2{X: 20, Y: 20}, logical, 8)
	if noIcon.HasIcon() {
		t.Fatal("plain tooltip reports an icon")
	}
}

// TestRichTooltipEmptyDataIsInvalid checks degenerate payloads.
func TestRichTooltipEmptyDataIsInvalid(t *testing.T) {
	theme := richTooltipTestTheme()
	var cache RichTooltipCache
	if !cache.Update(theme, core.RichTooltip{}, core.Vec2{X: 5, Y: 5}, core.Vec2{X: 800, Y: 600}, 8) {
		t.Fatal("first empty update must rebuild")
	}
	if cache.Valid() {
		t.Fatal("empty tooltip must be invalid")
	}
	if cache.Bounds() != (core.Rect{}) {
		t.Fatalf("empty bounds = %+v", cache.Bounds())
	}
}

// TestRichTooltipInvalidationKeys checks every cache key busts layout.
func TestRichTooltipInvalidationKeys(t *testing.T) {
	theme := richTooltipTestTheme()
	logical := core.Vec2{X: 800, Y: 600}
	data := richTooltipTestData()
	var cache RichTooltipCache
	anchor := core.Vec2{X: 30, Y: 40}
	cache.Update(theme, data, anchor, logical, 8)
	if cache.Update(theme, data, anchor, logical, 8) {
		t.Fatal("identical update must reuse layout")
	}
	changed := data
	changed.Title = "Renamed"
	if !cache.Update(theme, changed, anchor, logical, 8) {
		t.Fatal("title change must rebuild")
	}
	if cache.Update(theme, changed, anchor, logical, 8) {
		t.Fatal("repeated title must reuse layout")
	}
	segChanged := changed
	segChanged.Segments = []core.RichSegment{{Text: "different body"}}
	if !cache.Update(theme, segChanged, anchor, logical, 8) {
		t.Fatal("segment change must rebuild")
	}
	if !cache.Update(theme, segChanged, core.Vec2{X: 31, Y: 40}, logical, 8) {
		t.Fatal("anchor change must rebuild")
	}
	anchor2 := core.Vec2{X: 31, Y: 40}
	if !cache.Update(theme, segChanged, anchor2, core.Vec2{X: 400, Y: 300}, 8) {
		t.Fatal("viewport change must rebuild")
	}
	if !cache.Update(theme, segChanged, anchor2, core.Vec2{X: 400, Y: 300}, 12) {
		t.Fatal("padding change must rebuild")
	}
	wide := segChanged
	wide.Width = 200
	if !cache.Update(theme, wide, anchor2, core.Vec2{X: 400, Y: 300}, 12) {
		t.Fatal("width change must rebuild")
	}
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTooltip, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		PaddingLeft: 10, PaddingTop: 10, PaddingRight: 10, PaddingBottom: 10,
	})
	if !cache.Update(theme, wide, anchor2, core.Vec2{X: 400, Y: 300}, 12) {
		t.Fatal("text revision change must rebuild")
	}
}

// TestRichTooltipAnchorMoveRepositionsWithoutRewrap checks cursor motion.
func TestRichTooltipAnchorMoveRepositionsWithoutRewrap(t *testing.T) {
	theme := richTooltipTestTheme()
	logical := core.Vec2{X: 800, Y: 600}
	data := richTooltipTestData()
	var cache RichTooltipCache
	first := core.Vec2{X: 30, Y: 40}
	cache.Update(theme, data, first, logical, 8)
	before := cache.Bounds()
	relative := cache.bodySpans[0].Bounds
	second := core.Vec2{X: 34, Y: 44}
	if !cache.Update(theme, data, second, logical, 8) {
		t.Fatal("anchor move must translate the popup")
	}
	after := cache.Bounds()
	if after.X-before.X != 4 || after.Y-before.Y != 4 {
		t.Fatalf("popup did not follow anchor: %+v -> %+v", before, after)
	}
	if cache.bodySpans[0].Bounds != relative {
		t.Fatal("anchor move re-wrapped body text")
	}
	moves := testing.AllocsPerRun(200, func() {
		cache.Update(theme, data, second, logical, 8)
	})
	if moves != 0 {
		t.Fatalf("anchor steady allocations = %v, want 0", moves)
	}
	shift := testing.AllocsPerRun(100, func() {
		cache.Update(theme, data, first, logical, 8)
		cache.Update(theme, data, second, logical, 8)
	})
	if shift != 0 {
		t.Fatalf("anchor move allocations = %v, want 0", shift)
	}
}

// TestRichTooltipEmptyUpdateCaches checks degenerate payloads stay cached.
func TestRichTooltipEmptyUpdateCaches(t *testing.T) {
	theme := richTooltipTestTheme()
	logical := core.Vec2{X: 800, Y: 600}
	var cache RichTooltipCache
	anchor := core.Vec2{X: 5, Y: 5}
	if !cache.Update(theme, core.RichTooltip{}, anchor, logical, 8) {
		t.Fatal("first empty update must run layout")
	}
	if cache.Valid() || cache.Bounds() != (core.Rect{}) {
		t.Fatalf("empty cache = valid %v bounds %+v", cache.Valid(), cache.Bounds())
	}
	if cache.Update(theme, core.RichTooltip{}, anchor, logical, 8) {
		t.Fatal("identical empty update must reuse layout")
	}
	empty := testing.AllocsPerRun(200, func() {
		cache.Update(theme, core.RichTooltip{}, anchor, logical, 8)
	})
	if empty != 0 {
		t.Fatalf("empty steady allocations = %v, want 0", empty)
	}
}

// TestRichTooltipInvalidateFlushesLayout checks explicit cache resets.
func TestRichTooltipInvalidateFlushesLayout(t *testing.T) {
	theme := richTooltipTestTheme()
	logical := core.Vec2{X: 800, Y: 600}
	data := richTooltipTestData()
	var cache RichTooltipCache
	anchor := core.Vec2{X: 30, Y: 40}
	cache.Update(theme, data, anchor, logical, 8)
	if !cache.Valid() {
		t.Fatal("populated cache must be valid")
	}
	cache.Invalidate()
	if cache.Valid() || cache.Bounds() != (core.Rect{}) {
		t.Fatalf("invalidated cache = valid %v bounds %+v", cache.Valid(), cache.Bounds())
	}
	if cache.TitleLineCount() != 0 || cache.BodySpanCount() != 0 {
		t.Fatal("invalidated rows were retained")
	}
	if !cache.Update(theme, data, anchor, logical, 8) {
		t.Fatal("post-invalidate update must rebuild")
	}
	if !cache.Valid() {
		t.Fatal("rebuilt cache must be valid")
	}
	var nilCache *RichTooltipCache
	nilCache.Invalidate()
}

// TestRichTooltipSteadyStateAllocatesNothing checks hot-path reuse.
func TestRichTooltipSteadyStateAllocatesNothing(t *testing.T) {
	theme := richTooltipTestTheme()
	theme.SetDrawRecorder(nil)
	logical := core.Vec2{X: 800, Y: 600}
	data := richTooltipTestData()
	var cache RichTooltipCache
	cache.Update(theme, data, core.Vec2{X: 30, Y: 40}, logical, 8)
	info := core.WidgetInfo{Name: "tip", Kind: core.WidgetTooltip, State: core.StateNormal}
	theme.DrawRichTooltip(info, &cache)
	updates := testing.AllocsPerRun(200, func() {
		cache.Update(theme, data, core.Vec2{X: 30, Y: 40}, logical, 8)
	})
	if updates != 0 {
		t.Fatalf("cached update allocations = %v, want 0", updates)
	}
	draws := testing.AllocsPerRun(200, func() {
		theme.DrawRichTooltip(info, &cache)
	})
	if draws != 0 {
		t.Fatalf("cached draw allocations = %v, want 0", draws)
	}
	var nilCache *RichTooltipCache
	if nilCache.Update(theme, data, core.Vec2{}, logical, 8) {
		t.Fatal("nil cache update must not rebuild")
	}
	if got := nilCache.Bounds(); got != (core.Rect{}) {
		t.Fatalf("nil cache bounds = %+v", got)
	}
}

// TestRichTooltipDrawRecordsShellTextAndIcon checks recorder output.
func TestRichTooltipDrawRecordsShellTextAndIcon(t *testing.T) {
	theme := richTooltipTestTheme()
	recorder, err := NewDrawRecorder(256)
	if err != nil {
		t.Fatal(err)
	}
	theme.SetDrawRecorder(recorder)
	logical := core.Vec2{X: 800, Y: 600}
	data := richTooltipTestData()
	data.HasIcon = true
	data.Icon = core.TooltipIcon{ID: 9, Width: 32, Height: 32}
	var cache RichTooltipCache
	cache.Update(theme, data, core.Vec2{X: 30, Y: 40}, logical, 8)
	theme.BeginFrame()
	info := core.WidgetInfo{Name: "tip", Kind: core.WidgetTooltip, State: core.StateNormal}
	theme.DrawRichTooltip(info, &cache)
	calls := recorder.Calls()
	if !hasPart(calls, skin.PartBackground) || !hasPart(calls, skin.PartText) {
		t.Fatalf("rich tooltip missing shell or text: %+v", calls)
	}
	if !hasPart(calls, skin.PartIcon) {
		t.Fatalf("rich tooltip missing icon: %+v", calls)
	}
	before := len(calls)
	theme.BeginFrame()
	theme.DrawRichTooltip(info, nil)
	theme.DrawRichTooltip(info, &RichTooltipCache{})
	if got := len(recorder.Calls()); got != 0 {
		t.Fatalf("invalid draw recorded %d calls, want 0 (before %d)", got, before)
	}
}
