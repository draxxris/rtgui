package main

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/ui"
	"testing"
)

// TestGameWidgetGalleryHeadless exercises the actual gallery integration without GL.
func TestGameWidgetGalleryHeadless(t *testing.T) {
	u := ui.New(1280, 780)
	g := newGallery(u)
	if g.lineGraph.Visible() {
		t.Fatal("graph should start on the Style page only")
	}
	u.SelectTab("demoTabs", 1)
	u.Draw()
	if !g.lineGraph.Visible() || g.scroll.Visible() {
		t.Fatal("Style page did not swap its graph")
	}
	from, to := g.frameButton.Bounds(), g.panel.Bounds()
	a := core.Vec2{X: from.X + 10, Y: from.Y + 10}
	b := core.Vec2{X: to.X + 20, Y: to.Y + 40}
	u.HandleMouse(ui.MouseEvent{Pos: a, Pressed: true, Down: true})
	u.HandleMouse(ui.MouseEvent{Pos: b, Down: true})
	u.Draw()
	u.HandleMouse(ui.MouseEvent{Pos: b, Released: true})
	if g.status != "Item drop delivered once; the game owns inventory validation" {
		t.Fatalf("gallery drop: %q", g.status)
	}
}
