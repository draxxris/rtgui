package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// TestScrollbarTrackAndThumbDraw verifies that a scroll panel with positive MaxScroll
// draws track and thumb parts with correct geometry.
func TestScrollbarTrackAndThumbDraw(t *testing.T) {
	u := New(400, 300)
	recorder := attachDrawRecorder(t, u)

	sp := widgets.NewScrollPanel("scroll", core.Rect{X: 50, Y: 50, W: 200, H: 100})
	sp.SetMaxScroll(core.Vec2{Y: 100})
	mustAdd(t, u, sp)

	u.Draw()

	foundTrack := false
	foundThumb := false
	for _, call := range recorder.Calls() {
		if call.Kind == core.WidgetScrollPanel && call.Part == skin.PartTrack {
			foundTrack = true
			if call.Dest.W != 16 || call.Dest.H != 100 {
				t.Fatalf("unexpected track dest: %v", call.Dest)
			}
		}
		if call.Kind == core.WidgetScrollPanel && call.Part == skin.PartThumb {
			foundThumb = true
			if call.Dest.W != 16 || call.Dest.H <= 0 {
				t.Fatalf("unexpected thumb dest: %v", call.Dest)
			}
		}
	}
	if !foundTrack {
		t.Fatal("expected PartTrack draw call")
	}
	if !foundThumb {
		t.Fatal("expected PartThumb draw call")
	}
}

// TestScrollbarThumbHoverAndDrag verifies hover pseudo-class state and drag scrolling.
func TestScrollbarThumbHoverAndDrag(t *testing.T) {
	u := New(400, 300)
	sp := widgets.NewScrollPanel("scroll", core.Rect{X: 50, Y: 50, W: 200, H: 100})
	sp.SetMaxScroll(core.Vec2{Y: 100})
	mustAdd(t, u, sp)

	state, err := u.ScrollThumbState("scroll")
	if err != nil || state != core.StateNormal {
		t.Fatalf("expected StateNormal, got %v, err=%v", state, err)
	}

	thumb := scrollThumbRect(sp)
	thumbCenter := core.Vec2{X: thumb.X + thumb.W/2, Y: thumb.Y + thumb.H/2}

	// Move mouse over thumb
	u.HandleMouse(MouseEvent{Pos: thumbCenter})
	state, _ = u.ScrollThumbState("scroll")
	if state != core.StateHovered {
		t.Fatalf("expected StateHovered when mouse over thumb, got %v", state)
	}

	// Move mouse away
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 10, Y: 10}})
	state, _ = u.ScrollThumbState("scroll")
	if state != core.StateNormal {
		t.Fatalf("expected StateNormal when mouse moves away, got %v", state)
	}

	// Press mouse on thumb -> starts dragging
	if !u.HandleMouse(MouseEvent{Pos: thumbCenter, Pressed: true, Down: true}) {
		t.Fatal("expected thumb press to be handled")
	}
	state, _ = u.ScrollThumbState("scroll")
	if state != core.StatePressed {
		t.Fatalf("expected StatePressed during thumb drag, got %v", state)
	}

	// Drag mouse down
	dragTarget := core.Vec2{X: thumbCenter.X, Y: thumbCenter.Y + 30}
	if !u.HandleMouse(MouseEvent{Pos: dragTarget, Down: true}) {
		t.Fatal("expected drag to be handled")
	}
	if sp.Scroll().Y <= 0 {
		t.Fatalf("expected scroll.Y to increase after dragging down, got %f", sp.Scroll().Y)
	}

	// Release mouse
	if !u.HandleMouse(MouseEvent{Pos: dragTarget, Released: true}) {
		t.Fatal("expected release to be handled")
	}
	state, _ = u.ScrollThumbState("scroll")
	if state != core.StateHovered {
		t.Fatalf("expected StateHovered after release over thumb, got %v", state)
	}

	// Move mouse away from thumb
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 10, Y: 10}})
	state, _ = u.ScrollThumbState("scroll")
	if state != core.StateNormal {
		t.Fatalf("expected StateNormal when mouse moves away, got %v", state)
	}
}

// TestScrollbarTrackClickJumpsScroll verifies clicking the track outside the thumb jumps scroll.
func TestScrollbarTrackClickJumpsScroll(t *testing.T) {
	u := New(400, 300)
	sp := widgets.NewScrollPanel("scroll", core.Rect{X: 50, Y: 50, W: 200, H: 100})
	sp.SetMaxScroll(core.Vec2{Y: 100})
	mustAdd(t, u, sp)

	track := scrollTrackRect(sp)
	// Click near the bottom of the track (below thumb)
	clickPos := core.Vec2{X: track.X + track.W/2, Y: track.Y + track.H - 10}

	if !u.HandleMouse(MouseEvent{Pos: clickPos, Pressed: true}) {
		t.Fatal("expected track click to be handled")
	}
	if sp.Scroll().Y <= 0 {
		t.Fatalf("expected scroll offset to increase after track click, got %f", sp.Scroll().Y)
	}
	u.HandleMouse(MouseEvent{Pos: clickPos, Released: true})
}
