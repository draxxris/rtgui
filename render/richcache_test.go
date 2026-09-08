package render

import (
	"strings"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// richCacheSource is an indexed test source without defensive copies.
type richCacheSource struct {
	segments []core.RichSegment
	revision uint64
}

// RichSegmentCount reports the source length without allocating.
func (s *richCacheSource) RichSegmentCount() int {
	if s == nil {
		return 0
	}
	return len(s.segments)
}

// RichSegmentAt returns one segment copy without allocating.
func (s *richCacheSource) RichSegmentAt(index int) (core.RichSegment, bool) {
	if s == nil || index < 0 || index >= len(s.segments) {
		return core.RichSegment{}, false
	}
	return s.segments[index], true
}

// RichRevision reports the manual revision for cache tests.
func (s *richCacheSource) RichRevision() uint64 {
	if s == nil {
		return 0
	}
	return s.revision
}

// richCacheSegments returns a stable mixed message for cache tests.
func richCacheSegments() []core.RichSegment {
	return []core.RichSegment{
		{Text: "See "},
		{Text: "Thunderfury Blessed Blade", Link: core.Link{Kind: core.LinkItem, Target: "item:19019"}},
		{Text: " ok"},
	}
}

// richCacheBounds returns stable bounds for cache tests.
func richCacheBounds() core.Rect {
	return core.Rect{X: 10, Y: 10, W: 90, H: 200}
}

// TestRichLayoutCacheMatchesLegacy checks cached spans equal LayoutRichSpans.
func TestRichLayoutCacheMatchesLegacy(t *testing.T) {
	theme := richLayoutTheme()
	segments := richCacheSegments()
	bounds := richCacheBounds()
	legacy := theme.LayoutRichSpans(bounds, segments, core.StateNormal)
	var cache RichLayoutCache
	if !cache.UpdateSegments(theme, bounds, segments, core.StateNormal) {
		t.Fatal("first update must rebuild")
	}
	assertSpansEqual(t, cache.Spans(), legacy)
	if cache.UpdateSegments(theme, bounds, segments, core.StateNormal) {
		t.Fatal("identical update must not rebuild")
	}
}

// assertSpansEqual compares cached and legacy spans element by element.
func assertSpansEqual(t *testing.T, cached, legacy []RichSpanLayout) {
	t.Helper()
	if len(cached) != len(legacy) {
		t.Fatalf("cached %d spans, legacy %d", len(cached), len(legacy))
	}
	for i := range legacy {
		if cached[i] != legacy[i] {
			t.Fatalf("span %d = %+v, legacy %+v", i, cached[i], legacy[i])
		}
	}
}

// TestRichLayoutCacheIndexedReadsWidgetWithoutCopies checks indexed updates.
func TestRichLayoutCacheIndexedReadsWidgetWithoutCopies(t *testing.T) {
	theme := richLayoutTheme()
	bounds := richCacheBounds()
	source := &richCacheSource{segments: richCacheSegments(), revision: 1}
	var cache RichLayoutCache
	if !cache.Update(theme, bounds, source, core.StateNormal) {
		t.Fatal("first indexed update must rebuild")
	}
	if cache.Len() == 0 {
		t.Fatal("no cached spans")
	}
	if cache.Update(theme, bounds, source, core.StateNormal) {
		t.Fatal("steady indexed update must not rebuild")
	}
	source.segments[1].Text = "changed"
	if cache.Update(theme, bounds, source, core.StateNormal) {
		t.Fatal("same revision must trust revision without deep compare")
	}
	source.revision = 2
	if !cache.Update(theme, bounds, source, core.StateNormal) {
		t.Fatal("revision bump must rebuild")
	}
	if cache.Len() == 0 {
		t.Fatal("rebuilt spans empty")
	}
	other := &richCacheSource{segments: richCacheSegments(), revision: 2}
	if !cache.Update(theme, bounds, other, core.StateNormal) {
		t.Fatal("source identity change must rebuild")
	}
}

// TestRichLayoutCacheIgnoresBareState checks hover without inset change.
func TestRichLayoutCacheIgnoresBareState(t *testing.T) {
	theme := richLayoutTheme()
	segments := richCacheSegments()
	bounds := richCacheBounds()
	var cache RichLayoutCache
	if !cache.UpdateSegments(theme, bounds, segments, core.StateNormal) {
		t.Fatal("first update must rebuild")
	}
	if cache.UpdateSegments(theme, bounds, segments, core.StateHovered) {
		t.Fatal("bare state change with identical insets must not rebuild")
	}
}

// TestRichLayoutCacheInvalidatesOnKeys checks bounds/skin/segment keys.
func TestRichLayoutCacheInvalidatesOnKeys(t *testing.T) {
	theme := richLayoutTheme()
	segments := richCacheSegments()
	bounds := richCacheBounds()
	var cache RichLayoutCache
	if !cache.UpdateSegments(theme, bounds, segments, core.StateNormal) {
		t.Fatal("first update must rebuild")
	}
	moved := core.Rect{X: 11, Y: 10, W: 90, H: 200}
	if !cache.UpdateSegments(theme, moved, segments, core.StateNormal) {
		t.Fatal("bounds change must rebuild")
	}
	if cache.UpdateSegments(theme, moved, segments, core.StateNormal) {
		t.Fatal("steady update must not rebuild")
	}
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetRichText, Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{PaddingLeft: 9})
	if !cache.UpdateSegments(theme, moved, segments, core.StateNormal) {
		t.Fatal("skin revision change must rebuild")
	}
	changed := []core.RichSegment{{Text: "other"}}
	if !cache.UpdateSegments(theme, moved, changed, core.StateNormal) {
		t.Fatal("segment change must rebuild")
	}
	if cache.Len() == 0 {
		t.Fatal("changed layout empty")
	}
	cache.Invalidate()
	if cache.Len() != 0 {
		t.Fatal("invalidate must drop spans")
	}
	if _, ok := cache.SpanAt(core.Vec2{X: 12, Y: 12}); ok {
		t.Fatal("invalidated cache must miss")
	}
}

// TestRichLayoutCacheHitTestMatchesLegacy checks cached hit testing.
func TestRichLayoutCacheHitTestMatchesLegacy(t *testing.T) {
	theme := richLayoutTheme()
	segments := richCacheSegments()
	bounds := richCacheBounds()
	var cache RichLayoutCache
	cache.UpdateSegments(theme, bounds, segments, core.StateNormal)
	for _, span := range cache.Spans() {
		middle := core.Vec2{X: span.Bounds.X + span.Bounds.W/2, Y: span.Bounds.Y + span.Bounds.H/2}
		legacy, okLegacy := RichSpanAt(theme.LayoutRichSpans(bounds, segments, core.StateNormal), middle)
		cached, okCached := cache.SpanAt(middle)
		if !okLegacy || !okCached || cached != legacy {
			t.Fatalf("hit %+v legacy %+v/%v cached %+v/%v", middle, legacy, okLegacy, cached, okCached)
		}
	}
	if _, ok := cache.SpanAt(core.Vec2{}); ok {
		t.Fatal("outside miss reported a fragment")
	}
	assertNilCacheSafe(t)
}

// assertNilCacheSafe checks nil cache accessors never allocate or hit.
func assertNilCacheSafe(t *testing.T) {
	t.Helper()
	var nilCache *RichLayoutCache
	if _, ok := nilCache.SpanAt(core.Vec2{}); ok {
		t.Fatal("nil cache hit")
	}
	if nilCache.Spans() != nil || nilCache.Len() != 0 {
		t.Fatal("nil cache must read empty")
	}
	if nilCache.Update(nil, core.Rect{}, nil, core.StateNormal) {
		t.Fatal("nil cache update must not rebuild")
	}
	if nilCache.UpdateSegments(nil, core.Rect{}, nil, core.StateNormal) {
		t.Fatal("nil slice update must not rebuild")
	}
}

// TestRichLayoutCacheSteadyStateAllocatesNothing protects draw/hit paths.
func TestRichLayoutCacheSteadyStateAllocatesNothing(t *testing.T) {
	theme := richLayoutTheme()
	theme.SetDrawRecorder(nil)
	segments := richCacheSegments()
	bounds := core.Rect{X: 10, Y: 10, W: 300, H: 200}
	var cache RichLayoutCache
	source := &richCacheSource{segments: segments, revision: 1}
	cache.Update(theme, bounds, source, core.StateNormal)
	cache.UpdateSegments(theme, bounds, segments, core.StateNormal)
	point := core.Vec2{X: 20, Y: 20}
	info := core.WidgetInfo{Name: "chat", Bounds: bounds, Kind: core.WidgetRichText, State: core.StateNormal}
	allocations := testing.AllocsPerRun(200, func() {
		cache.Update(theme, bounds, source, core.StateNormal)
		_ = cache.Spans()
		_, _ = cache.SpanAt(point)
		theme.DrawRichTextLayout(info, &cache, -1)
	})
	if allocations != 0 {
		t.Fatalf("cache steady-state allocations = %v, want 0", allocations)
	}
	sliceAllocs := testing.AllocsPerRun(200, func() {
		cache.UpdateSegments(theme, bounds, segments, core.StateNormal)
	})
	if sliceAllocs != 0 {
		t.Fatalf("slice steady-state allocations = %v, want 0", sliceAllocs)
	}
}

// TestRichLayoutCacheDrawMatchesLegacy checks cached draw records.
func TestRichLayoutCacheDrawMatchesLegacy(t *testing.T) {
	theme := richLayoutTheme()
	legacyRecorder, err := NewDrawRecorder(128)
	if err != nil {
		t.Fatal(err)
	}
	cachedRecorder, err := NewDrawRecorder(128)
	if err != nil {
		t.Fatal(err)
	}
	segments := []core.RichSegment{
		{Text: "hi "},
		{Text: "item", Link: core.Link{Kind: core.LinkItem, Target: "item:1"}},
	}
	info := core.WidgetInfo{Name: "chat", Bounds: core.Rect{W: 300, H: 60}, Kind: core.WidgetRichText, State: core.StateNormal}
	theme.SetDrawRecorder(legacyRecorder)
	theme.BeginFrame()
	theme.DrawRichText(info, segments, 1)
	legacy := legacyRecorder.Calls()
	var cache RichLayoutCache
	cache.UpdateSegments(theme, info.Bounds, segments, info.State)
	theme.SetDrawRecorder(cachedRecorder)
	theme.BeginFrame()
	theme.DrawRichTextLayout(info, &cache, 1)
	cached := cachedRecorder.Calls()
	if len(cached) != len(legacy) {
		t.Fatalf("cached %d calls, legacy %d", len(cached), len(legacy))
	}
	for i := range legacy {
		if cached[i] != legacy[i] {
			t.Fatalf("call %d cached %+v legacy %+v", i, cached[i], legacy[i])
		}
	}
	theme.SetDrawRecorder(nil)
	theme.DrawRichTextLayout(info, nil, -1)
	var nilTheme *Theme
	nilTheme.DrawRichTextLayout(info, &cache, -1)
}

// TestRichTokenizerSpacesAndSplits checks whitespace and long words.
func TestRichTokenizerSpacesAndSplits(t *testing.T) {
	theme := richLayoutTheme()
	bounds := core.Rect{X: 0, Y: 0, W: 300, H: 200}
	single := theme.LayoutRichSpans(bounds, []core.RichSegment{{Text: "a b"}}, core.StateNormal)
	double := theme.LayoutRichSpans(bounds, []core.RichSegment{{Text: "a  b"}}, core.StateNormal)
	if len(single) != 2 || len(double) != 2 {
		t.Fatalf("word counts = %d/%d", len(single), len(double))
	}
	if double[1].Bounds.X <= single[1].Bounds.X {
		t.Fatalf("double space must advance further: %v vs %v", double[1].Bounds.X, single[1].Bounds.X)
	}
	broken := theme.LayoutRichSpans(bounds, []core.RichSegment{{Text: "a\nb"}}, core.StateNormal)
	if len(broken) != 2 || broken[1].Bounds.Y <= broken[0].Bounds.Y {
		t.Fatalf("newline must break rows: %+v", broken)
	}
	indented := theme.LayoutRichSpans(bounds, []core.RichSegment{{Text: "a\n    hi"}}, core.StateNormal)
	if len(indented) != 2 {
		t.Fatalf("indented spans = %+v", indented)
	}
	if indented[1].Bounds.X <= bounds.X {
		t.Fatalf("explicit newline indent stripped: %+v", indented[1].Bounds)
	}
	narrow := core.Rect{X: 0, Y: 0, W: 20, H: 1000}
	long := theme.LayoutRichSpans(narrow, []core.RichSegment{{Text: "supercalifragilistic"}}, core.StateNormal)
	if len(long) < 2 {
		t.Fatalf("long word must split, spans = %d", len(long))
	}
	var rebuilt strings.Builder
	for _, span := range long {
		if span.Bounds.W > narrow.W+0.001 {
			t.Fatalf("split width %v exceeds %v", span.Bounds.W, narrow.W)
		}
		rebuilt.WriteString(span.Text)
	}
	if rebuilt.String() != "supercalifragilistic" {
		t.Fatalf("split text = %q", rebuilt.String())
	}
	empty := theme.LayoutRichSpans(core.Rect{}, []core.RichSegment{{Text: "x"}}, core.StateNormal)
	if len(empty) != 0 {
		t.Fatalf("empty content spans = %d", len(empty))
	}
}

// TestRichWidgetIndexedReads checks count, indexing, and revision bumps.
func TestRichWidgetIndexedReads(t *testing.T) {
	message := widgets.NewRichText("chat", core.Rect{W: 100, H: 40}, richCacheSegments())
	if message.RichSegmentCount() != 3 {
		t.Fatalf("count = %d", message.RichSegmentCount())
	}
	first, ok := message.RichSegmentAt(0)
	if !ok || first.Text != "See " {
		t.Fatalf("segment 0 = %+v/%v", first, ok)
	}
	if _, ok := message.RichSegmentAt(3); ok {
		t.Fatal("out-of-range index accepted")
	}
	revision := message.RichRevision()
	if message.SetRichSegments(richCacheSegments()) {
		t.Fatal("identical segments bumped revision")
	}
	if message.RichRevision() != revision {
		t.Fatal("revision changed without content change")
	}
	if !message.SetRichSegments([]core.RichSegment{{Text: "other"}}) {
		t.Fatal("changed segments not reported")
	}
	if message.RichRevision() == revision {
		t.Fatal("revision did not bump on change")
	}
}

// TestRichWidgetCopyInto checks reusable copy semantics and nil safety.
func TestRichWidgetCopyInto(t *testing.T) {
	message := widgets.NewRichText("chat", core.Rect{W: 100, H: 40}, []core.RichSegment{{Text: "other"}})
	backing := make([]core.RichSegment, 0, 8)
	backing = message.CopyRichSegmentsInto(backing)
	if len(backing) != 1 || backing[0].Text != "other" {
		t.Fatalf("copy-into = %+v", backing)
	}
	var nilMessage *widgets.RichText
	if nilMessage.RichSegmentCount() != 0 || nilMessage.RichRevision() != 0 {
		t.Fatal("nil widget indexed access must be zero")
	}
	if _, ok := nilMessage.RichSegmentAt(0); ok {
		t.Fatal("nil widget index must fail")
	}
	if got := nilMessage.CopyRichSegmentsInto(nil); len(got) != 0 {
		t.Fatalf("nil copy-into = %+v", got)
	}
}
