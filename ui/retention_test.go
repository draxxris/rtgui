package ui

import (
	"fmt"
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
	"testing"
)

// TestClearWidgetsReleasesActiveScrollAndClosures exercises disposal during capture.
func TestClearWidgetsReleasesActiveScrollAndClosures(t *testing.T) {
	u := New(200, 200)
	panel := widgets.NewScrollPanel("scroll", core.Rect{W: 100, H: 100}).SetMaxScroll(core.Vec2{Y: 300})
	mustAdd(t, u, panel)
	u.OnHotkey("scoped", 'R', HotkeyOpts{Scope: "scroll"}, func() {})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 90, Y: 5}, Pressed: true, Down: true})
	u.ClearWidgets()
	before := panel.Scroll()
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 90, Y: 90}, Down: true})
	if panel.Scroll() != before || len(u.hotkeys) != 0 || u.scrollThumbDragging != nil || u.mouseCaptured {
		t.Fatal("clear retained widget interaction")
	}
}

// TestDiagnosticsAreBoundedAndDeduplicated prevents repeated fallback retention.
func TestDiagnosticsAreBoundedAndDeduplicated(t *testing.T) {
	u := New(100, 100)
	for i := 0; i < 300; i++ {
		u.diagnose("warning %d", i)
	}
	if len(u.diagnostics) != 128 {
		t.Fatalf("unbounded diagnostics: %d", len(u.diagnostics))
	}
	u.diagnose("warning %d", 299)
	if len(u.diagnostics) != 128 || u.diagnostics[0] != fmt.Sprintf("warning %d", 172) {
		t.Fatal("diagnostic deduplication failed")
	}
}

// TestHotkeySelfRemovalPreservesConsumption guards synchronous callback mutation.
func TestHotkeySelfRemovalPreservesConsumption(t *testing.T) {
	u := New(100, 100)
	u.OnHotkey("self", 'R', HotkeyOpts{Consume: true}, func() { u.RemoveHotkey("self") })
	if !u.HandleKey(KeyEvent{Hotkeys: []rune{'R'}}) {
		t.Fatal("self-removal changed current input consumption")
	}
	if len(u.hotkeys) != 0 {
		t.Fatal("self-removing callback retained")
	}
}

// TestRightClickConsumesContextAndCancelsDrag checks both manual and polled semantics.
func TestRightClickConsumesContextAndCancelsDrag(t *testing.T) {
	u, _, _ := dragFixture(t)
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 10, Y: 10}, Pressed: true})
	if !u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 10, Y: 10}, RightPressed: true}) || u.dragSource != nil {
		t.Fatal("right click did not cancel drag")
	}
	u.HandleMouse(MouseEvent{Released: true})
	called := false
	u.OnContextMenu(func(core.Vec2) { called = true })
	if !u.HandleMouse(MouseEvent{RightPressed: true}) || !called {
		t.Fatal("context click leaked to world input")
	}
}
