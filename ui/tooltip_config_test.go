package ui

import (
	"testing"
	"time"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// tooltipVisible draws one frame and reports whether a tooltip reached the recorder.
func tooltipVisible(t *testing.T, u *UI) bool {
	t.Helper()
	recorder := attachDrawRecorder(t, u)
	u.DrawWidgets()
	u.DrawPopup()
	return hasKindPart(recorder.Calls(), core.WidgetTooltip, skin.PartText)
}

// tooltipTestClock installs a manually advanced clock and returns the advance function.
func tooltipTestClock(u *UI, now time.Time) func(time.Duration) {
	current := now
	u.setTooltipClock(func() time.Time { return current })
	return func(d time.Duration) { current = current.Add(d) }
}

// TestTooltipDelayDefaultsImmediate checks zero-value configuration shows instantly.
func TestTooltipDelayDefaultsImmediate(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltip("slot", "Iron Sword")
	if got := u.TooltipDelay(); got != 0 {
		t.Fatalf("default delay = %v, want 0", got)
	}
	if got := u.TooltipAnchor(); got != (TooltipAnchor{}) {
		t.Fatalf("default anchor = %+v, want zero", got)
	}
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	if !tooltipVisible(t, u) {
		t.Fatal("default configuration must show hover tooltips immediately")
	}
}

// TestTooltipDelayGatesHover checks the dwell hides early draws and shows at expiry.
func TestTooltipDelayGatesHover(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltip("slot", "Iron Sword")
	advance := tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(500 * time.Millisecond)

	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	if tooltipVisible(t, u) {
		t.Fatal("tooltip drew before the dwell elapsed")
	}
	advance(499 * time.Millisecond)
	if tooltipVisible(t, u) {
		t.Fatal("tooltip drew 1ms before the dwell elapsed")
	}
	advance(time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("tooltip did not draw once the dwell elapsed")
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 700, Y: 500}})
	if tooltipVisible(t, u) {
		t.Fatal("tooltip drew after hover left")
	}
	u.SetTooltipDelay(-time.Second)
	if got := u.TooltipDelay(); got != 0 {
		t.Fatalf("negative delay = %v, want clamped 0", got)
	}
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	if !tooltipVisible(t, u) {
		t.Fatal("clamped delay must show immediately")
	}
}

// TestTooltipDelaySurvivesIntraWidgetMotion checks moves within one widget keep the dwell.
func TestTooltipDelaySurvivesIntraWidgetMotion(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 200, H: 60}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltip("slot", "Iron Sword")
	advance := tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(500 * time.Millisecond)

	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	advance(400 * time.Millisecond)
	moved := centerOf(button)
	moved.X += 40
	u.HandleMouse(MouseEvent{Pos: moved})
	advance(100 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("intra-widget motion must not restart the dwell")
	}
}

// TestTooltipDelayRestartsOnTargetChange checks hovering a new widget restarts the dwell.
func TestTooltipDelayRestartsOnTargetChange(t *testing.T) {
	u := New(800, 600)
	first := widgets.NewButton("first", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "First")
	second := widgets.NewButton("second", core.Rect{X: 200, Y: 50, W: 100, H: 30}, "Second")
	mustAdd(t, u, first, second)
	u.SetTooltip("first", "First tip")
	u.SetTooltip("second", "Second tip")
	advance := tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(500 * time.Millisecond)

	u.HandleMouse(MouseEvent{Pos: centerOf(first)})
	advance(400 * time.Millisecond)
	u.HandleMouse(MouseEvent{Pos: centerOf(second)})
	if tooltipVisible(t, u) {
		t.Fatal("new hover target must restart the dwell")
	}
	advance(400 * time.Millisecond)
	if tooltipVisible(t, u) {
		t.Fatal("tooltip drew 100ms into the restarted dwell")
	}
	advance(100 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("tooltip did not draw after the restarted dwell elapsed")
	}
}

// TestTooltipPerWidgetDelayOverride checks per-widget dwells beat the global one.
func TestTooltipPerWidgetDelayOverride(t *testing.T) {
	u := New(800, 600)
	slow := widgets.NewButton("slow", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slow")
	fast := widgets.NewButton("fast", core.Rect{X: 200, Y: 50, W: 100, H: 30}, "Fast")
	mustAdd(t, u, slow, fast)
	u.SetTooltip("slow", "Slow tip")
	u.SetTooltip("fast", "Fast tip")
	advance := tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(500 * time.Millisecond)
	u.SetTooltipOptions("fast", TooltipOptions{Delay: 0, HasDelay: true})

	u.HandleMouse(MouseEvent{Pos: centerOf(fast)})
	if !tooltipVisible(t, u) {
		t.Fatal("per-widget zero delay must show immediately under a global delay")
	}
	u.HandleMouse(MouseEvent{Pos: centerOf(slow)})
	if tooltipVisible(t, u) {
		t.Fatal("global delay must still gate widgets without overrides")
	}
	advance(500 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("global delay did not expire for the unoverridden widget")
	}
	u.SetTooltipOptions("fast", TooltipOptions{})
	if _, ok := u.TooltipOptions("fast"); ok {
		t.Fatal("cleared options survived")
	}
	u.HandleMouse(MouseEvent{Pos: centerOf(fast)})
	if tooltipVisible(t, u) {
		t.Fatal("cleared widget must fall back to the global delay")
	}
	u.SetTooltipOptions("fast", TooltipOptions{})
	if _, ok := u.TooltipOptions("fast"); ok {
		t.Fatal("second clear left an entry")
	}
	if _, ok := u.TooltipOptions("missing"); ok {
		t.Fatal("unknown widget must report no options")
	}
	u.SetTooltipOptions("missing", TooltipOptions{Delay: time.Second, HasDelay: true})
	if _, ok := u.TooltipOptions("missing"); ok {
		t.Fatal("unknown widget must not store options")
	}
	u.SetTooltipOptions("fast", TooltipOptions{})
	if _, ok := u.TooltipOptions("fast"); ok {
		t.Fatal("empty options must clear the entry")
	}
	var nilUI *UI
	nilUI.SetTooltipDelay(time.Second)
	nilUI.SetTooltipAnchor(TooltipAnchor{})
	nilUI.SetTooltipOptions("fast", TooltipOptions{HasDelay: true})
	nilUI.setTooltipClock(nil)
	if nilUI.TooltipDelay() != 0 {
		t.Fatal("nil UI answered tooltip config")
	}
	if _, ok := nilUI.TooltipOptions("fast"); ok {
		t.Fatal("nil UI answered per-widget options")
	}
}

// TestTooltipFixedAnchorPinsHover checks fixed anchors ignore pointer motion.
func TestTooltipFixedAnchorPinsHover(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 200, H: 60}, "Slot")
	mustAdd(t, u, button)
	u.SetRichTooltip("slot", core.RichTooltip{Title: "Iron Sword"})
	fixed := core.Vec2{X: 400, Y: 400}
	u.SetTooltipOptions("slot", TooltipOptions{
		Anchor:    TooltipAnchor{Kind: TooltipAnchorFixed, Point: fixed},
		HasAnchor: true,
	})

	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	u.DrawWidgets()
	u.DrawPopup()
	first := u.richTipCache.Bounds()
	if first.W <= 0 || first.H <= 0 {
		t.Fatal("fixed tooltip did not lay out")
	}
	if first.X != fixed.X+render.TooltipCursorOffset || first.Y != fixed.Y+render.TooltipCursorDrop {
		t.Fatalf("fixed bounds origin = (%v,%v), want point plus cursor gap", first.X, first.Y)
	}
	moved := centerOf(button)
	moved.X += 40
	u.HandleMouse(MouseEvent{Pos: moved})
	u.DrawWidgets()
	u.DrawPopup()
	if second := u.richTipCache.Bounds(); second != first {
		t.Fatalf("fixed tooltip followed the pointer: %+v vs %+v", first, second)
	}
	u.SetTooltipOptions("slot", TooltipOptions{})
	u.DrawWidgets()
	u.DrawPopup()
	before := u.richTipCache.Bounds()
	farther := centerOf(button)
	farther.X -= 40
	u.HandleMouse(MouseEvent{Pos: farther})
	u.DrawWidgets()
	u.DrawPopup()
	if after := u.richTipCache.Bounds(); after == before {
		t.Fatal("cursor anchor must follow the pointer after the override clears")
	}
}

// TestTooltipFixedAnchorServesPlainTips checks plain tips resolve the same anchor.
func TestTooltipFixedAnchorServesPlainTips(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltip("slot", "Iron Sword")
	fixed := core.Vec2{X: 300, Y: 300}
	u.SetTooltipOptions("slot", TooltipOptions{
		Anchor:    TooltipAnchor{Kind: TooltipAnchorFixed, Point: fixed},
		HasAnchor: true,
	})
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	text, anchor, ok := u.derivedTooltip()
	if !ok || text != "Iron Sword" {
		t.Fatalf("plain tip = %q/%v", text, ok)
	}
	if anchor != fixed {
		t.Fatalf("plain tip anchor = %+v, want %+v", anchor, fixed)
	}
	u.SetTooltipAnchor(TooltipAnchor{Kind: TooltipAnchorFixed, Point: fixed})
	u.SetTooltipOptions("slot", TooltipOptions{})
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	if _, anchor, ok := u.derivedTooltip(); !ok || anchor != fixed {
		t.Fatalf("global fixed anchor = %+v/%v, want %+v", anchor, ok, fixed)
	}
}

// TestTooltipExplicitBypassesDelay checks programmatic pins ignore the dwell.
func TestTooltipExplicitBypassesDelay(t *testing.T) {
	u := New(800, 600)
	tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(time.Hour)
	u.ShowTooltip("Pinned note", core.Vec2{X: 400, Y: 400})
	if !tooltipVisible(t, u) {
		t.Fatal("explicit tooltip must bypass the hover dwell")
	}
	u.HideTooltip()
	u.ShowRichTooltip(core.RichTooltip{Title: "Pinned"}, core.Vec2{X: 400, Y: 400})
	if !tooltipVisible(t, u) {
		t.Fatal("explicit rich tooltip must bypass the hover dwell")
	}
	u.HideTooltip()
}

// TestTooltipPressRestartsDwell checks dismissal waits out a fresh dwell.
func TestTooltipPressRestartsDwell(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltip("slot", "Iron Sword")
	advance := tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(500 * time.Millisecond)

	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	advance(600 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("tooltip did not appear after the dwell")
	}
	clickAt(u, centerOf(button))
	if tooltipVisible(t, u) {
		t.Fatal("press must restart the dwell instead of redrawing at once")
	}
	advance(500 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("tooltip did not reappear after the restarted dwell")
	}
}

// TestTooltipOptionsRemovedWithWidget checks disposal clears per-widget overrides.
func TestTooltipOptionsRemovedWithWidget(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltipOptions("slot", TooltipOptions{Delay: time.Second, HasDelay: true})
	if !u.Remove("slot") {
		t.Fatal("Remove did not dispose the widget")
	}
	if _, ok := u.TooltipOptions("slot"); ok {
		t.Fatal("removed widget kept tooltip options")
	}
}

// TestTooltipClockChangeRestartsDwell checks clock swaps do not strand pending dwells.
func TestTooltipClockChangeRestartsDwell(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltip("slot", "Iron Sword")
	u.SetTooltipDelay(500 * time.Millisecond)
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	advance := tooltipTestClock(u, time.Unix(2000, 0))
	advance(499 * time.Millisecond)
	if tooltipVisible(t, u) {
		t.Fatal("clock change must restart the pending dwell")
	}
	advance(time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("tooltip did not draw after the restarted dwell")
	}
	u.setTooltipClock(nil)
	u.SetTooltipDelay(0)
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	if !tooltipVisible(t, u) {
		t.Fatal("restored clock must show immediate tooltips")
	}
}

// TestTooltipLinkDwellSurvivesMenuDismissal checks a menu opened over a link
// does not leak dwell credit: returning to the same link waits out a fresh
// dwell instead of showing immediately.
func TestTooltipLinkDwellSurvivesMenuDismissal(t *testing.T) {
	u := New(800, 600)
	message := richTestMessage(t, u)
	u.OnLinkTooltipRequested("chat", func(core.Link) string { return "dynamic tip" })
	advance := tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(500 * time.Millisecond)

	link := richSegCenter(u, message, 1)
	u.HandleMouse(MouseEvent{Pos: link})
	advance(600 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("link tip did not appear after the dwell")
	}
	var selected string
	openTestMenu(u, &selected)
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 700, Y: 500}, Pressed: true})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 700, Y: 500}, Released: true})
	if u.HasOpenMenu() {
		t.Fatal("outside press must dismiss the menu")
	}
	u.HandleMouse(MouseEvent{Pos: richSegCenter(u, message, 1)})
	if tooltipVisible(t, u) {
		t.Fatal("returning to the same link must wait out a fresh dwell")
	}
	advance(499 * time.Millisecond)
	if tooltipVisible(t, u) {
		t.Fatal("link tip drew 1ms before the fresh dwell elapsed")
	}
	advance(time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("link tip did not draw after the fresh dwell elapsed")
	}
}

// TestTooltipLinkRevisionKeepsDwell checks a text revision under a stationary
// hover refreshes link content without restarting the dwell.
func TestTooltipLinkRevisionKeepsDwell(t *testing.T) {
	u := New(800, 600)
	message := richTestMessage(t, u)
	requests := 0
	u.OnLinkTooltipRequested("chat", func(core.Link) string {
		requests++
		return "dynamic tip"
	})
	advance := tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(500 * time.Millisecond)

	u.HandleMouse(MouseEvent{Pos: richSegCenter(u, message, 1)})
	advance(400 * time.Millisecond)
	revised := []core.RichSegment{
		{Text: "Need! "},
		{Text: "Thunderfury", Link: core.Link{Kind: core.LinkItem, Target: "item:19019"}},
		{Text: " thanks"},
		{Text: "wiki", Link: core.Link{Kind: core.LinkURL, Target: "https://example.com"}},
	}
	if !message.SetRichSegments(revised) {
		t.Fatal("revision setup did not change segments")
	}
	u.HandleMouse(MouseEvent{Pos: richSegCenter(u, message, 1)})
	if requests != 2 {
		t.Fatalf("revision refresh requests = %d, want 2", requests)
	}
	advance(100 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("text revision must not restart the link dwell")
	}
}

// TestTooltipEscapeRestartsHoverDwell checks Escape dismisses a hover tooltip
// without an explicit tip present and requires a fresh dwell.
func TestTooltipEscapeRestartsHoverDwell(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltip("slot", "Iron Sword")
	advance := tooltipTestClock(u, time.Unix(1000, 0))
	u.SetTooltipDelay(500 * time.Millisecond)

	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	advance(600 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("tooltip did not appear after the dwell")
	}
	u.HandleKey(KeyEvent{Escape: true})
	if tooltipVisible(t, u) {
		t.Fatal("Escape must restart the hover dwell")
	}
	advance(500 * time.Millisecond)
	if !tooltipVisible(t, u) {
		t.Fatal("tooltip did not reappear after the restarted dwell")
	}
}
