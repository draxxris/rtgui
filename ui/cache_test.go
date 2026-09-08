package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// cacheFixture constructs representative widgets outside the measurement loop.
func cacheFixture(t testing.TB) *UI {
	t.Helper()
	u := New(800, 600)
	tab := widgets.NewTabBar("tabs", core.Rect{W: 300, H: 40}, []string{"Bag", "Bank", "Equipment"}, 0)
	text := widgets.NewTextbox("text", core.Rect{Y: 50, W: 300, H: 40}, 128)
	text.SetText("An unchanged text buffer with a cached string")
	rich := widgets.NewRichText("chat", core.Rect{Y: 100, W: 600, H: 100}, []core.RichSegment{{Text: "A player found a legendary sword in the dungeon today", Link: core.Link{Kind: core.LinkItem, Target: "sword"}}})
	progress := widgets.NewProgressBar("progress", core.Rect{Y: 210, W: 300, H: 30}, .5).SetFormat("%.0f%%")
	if err := u.Add(tab, text, rich, progress); err != nil {
		t.Fatal(err)
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 5, Y: 105}})
	u.Draw()
	return u
}

// BenchmarkCachedUIFrame measures complete headless drawing and rich-text hover.
func BenchmarkCachedUIFrame(b *testing.B) {
	u := cacheFixture(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 5, Y: 105}})
		u.Draw()
	}
}

// TestSteadyUIFrameAllocatesNothing covers draw, rich hover, and cache lookups.
func TestSteadyUIFrameAllocatesNothing(t *testing.T) {
	u := cacheFixture(t)
	allocs := testing.AllocsPerRun(100, func() {
		u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 5, Y: 105}})
		u.HoveredLink()
		u.Draw()
	})
	if allocs != 0 {
		t.Fatalf("steady frame: %g allocations", allocs)
	}
}

// TestRichTooltipCopiesAndReleases checks caller isolation and removal lifetime.
func TestRichTooltipCopiesAndReleases(t *testing.T) {
	u := New(800, 600)
	w := widgets.NewButton("item", core.Rect{W: 100, H: 40}, "item")
	mustAdd(t, u, w)
	segments := []core.RichSegment{{Text: "Original", HasColor: true, Color: core.Color{R: 200, A: 255}}}
	u.SetRichTooltip("item", core.RichTooltip{Title: "Sword", Segments: segments})
	segments[0].Text = "mutated"
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 5, Y: 5}})
	u.Draw()
	if u.richTips["item"].Segments[0].Text != "Original" || !u.richTipCache.Valid() {
		t.Fatal("tooltip data aliased caller storage")
	}
	allocs := testing.AllocsPerRun(100, func() { u.Draw() })
	if allocs != 0 {
		t.Fatalf("tooltip frame: %g allocations", allocs)
	}
	u.Remove("item")
	if len(u.richTips) != 0 || u.richTipCache.Valid() {
		t.Fatal("removed tooltip retained active content")
	}
}

// TestRichCacheInvalidatesOnEdits checks draw and hit-test use the same revision.
func TestRichCacheInvalidatesOnEdits(t *testing.T) {
	u := cacheFixture(t)
	rich := u.Lookup("chat").(*widgets.RichText)
	old := u.richCaches["chat"]
	rich.SetRichSegments([]core.RichSegment{{Text: "New", Link: core.Link{Kind: core.LinkItem, Target: "new"}}})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 5, Y: 105}})
	u.Draw()
	_, link, _, ok := u.HoveredLink()
	if !ok || link.Target != "new" || old.Spans()[0].Text != "New" {
		t.Fatal("hover or drawing used stale text")
	}
	u.Remove("chat")
	if len(u.richCaches) != 0 {
		t.Fatal("removed rich widget retained a cache")
	}
}
