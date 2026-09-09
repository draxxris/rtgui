package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/ui"
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

// clickGalleryButton presses and releases one gallery button by center.
func clickGalleryButton(t *testing.T, u *ui.UI, name string) {
	t.Helper()
	w := u.Lookup(name)
	if w == nil {
		t.Fatalf("missing widget %q", name)
	}
	bounds := w.Bounds()
	center := core.Vec2{X: bounds.X + bounds.W/2, Y: bounds.Y + bounds.H/2}
	u.HandleMouse(ui.MouseEvent{Pos: center, Pressed: true})
	u.HandleMouse(ui.MouseEvent{Pos: center, Released: true})
}

// TestGalleryCSSParsesTitledVariant checks the gallery skin carries the
// TitledFrame variant with ring opt-outs the overlay additive merge needs.
func TestGalleryCSSParsesTitledVariant(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "skins", "gallery.css"))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := skin.ParseCSS(string(text))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, rule := range rules {
		if rule.Class == "" || len(rule.Class) < 7 || rule.Class[:7] != "titled-" {
			continue
		}
		seen[rule.Class] = true
		if (rule.Class == "titled-titlebar" || rule.Class == "titled-content") && rule.Part == skin.PartBorder && !rule.NoTexture {
			t.Fatalf("%s border must opt out with none", rule.Class)
		}
	}
	for _, class := range []string{"titled-frame", "titled-titlebar", "titled-title", "titled-close", "titled-content"} {
		if !seen[class] {
			t.Fatalf("gallery css misses .%s", class)
		}
	}
}

// TestQuestLogDemo exercises the TitledFrame quest window: hidden by
// default, opened by its button, Accept/Decline report without closing,
// and the X button closes through the shared Close path.
func TestQuestLogDemo(t *testing.T) {
	u := ui.New(1280, 780)
	g := newGallery(u)
	if g.questLog.IsOpen() {
		t.Fatal("quest log must start hidden")
	}
	clickGalleryButton(t, u, "questButton")
	if !g.questLog.IsOpen() || g.status != "Quest log opened (demo)" {
		t.Fatalf("open: open=%v status=%q", g.questLog.IsOpen(), g.status)
	}
	clickGalleryButton(t, u, "questAccept")
	if !g.questLog.IsOpen() || g.status != "Quest accepted (demo)" {
		t.Fatalf("accept: open=%v status=%q", g.questLog.IsOpen(), g.status)
	}
	clickGalleryButton(t, u, "questDecline")
	if !g.questLog.IsOpen() || g.status != "Quest declined (demo)" {
		t.Fatalf("decline: open=%v status=%q", g.questLog.IsOpen(), g.status)
	}
	clickGalleryButton(t, u, "questLog/close")
	if g.questLog.IsOpen() || g.status != "Quest log closed (demo)" {
		t.Fatalf("close: open=%v status=%q", g.questLog.IsOpen(), g.status)
	}
	clickGalleryButton(t, u, "questButton")
	if !g.questLog.IsOpen() || g.status != "Quest log opened (demo)" {
		t.Fatalf("reopen: open=%v status=%q", g.questLog.IsOpen(), g.status)
	}
}
