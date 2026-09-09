package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/ui"
	"github.com/draxxris/rtgui/widgets"
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
	u.SelectTab("demoTabs", 2)
	u.Draw()
	if !g.chatLog.Visible() || g.scroll.Visible() || g.lineGraph.Visible() {
		t.Fatal("About page did not swap its chat log")
	}
	if g.chatLog.MessageCount() == 0 {
		t.Fatal("chat log feed is empty")
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

// TestGalleryCSSParsesScrollableParts checks the gallery skin carries the
// shell, highlight, and scrollbar parts for the collapsible list (with gold
// selection) and the chat log.
func TestGalleryCSSParsesScrollableParts(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "skins", "gallery.css"))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := skin.ParseCSS(string(text))
	if err != nil {
		t.Fatal(err)
	}
	want := map[core.WidgetKind][]string{
		core.WidgetList:    {"base", "highlight", "selected", "track", "thumb"},
		core.WidgetChatLog: {"base", "highlight", "track", "thumb"},
	}
	for kind, parts := range want {
		seen := map[string]bool{}
		for _, rule := range rules {
			if rule.Kind != kind {
				continue
			}
			seen[cssPartBucket(rule)] = true
		}
		for _, part := range parts {
			if !seen[part] {
				t.Fatalf("gallery css misses %v %s (seen=%v)", kind, part, seen)
			}
		}
	}
}

// cssPartBucket classifies one gallery rule for coverage checks.
func cssPartBucket(rule skin.SkinRule) string {
	switch {
	case rule.Part == skin.PartBackground:
		return "base"
	case rule.Part == skin.PartOverlay && rule.State == core.StateSelected:
		return "selected"
	case rule.Part == skin.PartOverlay:
		return "highlight"
	case rule.Part == skin.PartTrack:
		return "track"
	case rule.Part == skin.PartThumb:
		return "thumb"
	default:
		return "other"
	}
}

// TestCategoryWindowDemo exercises the floating Categories window: hidden by
// default, opened by its button, leaf select and category toggle report
// status, Sell reports without closing, and X closes the window.
func TestCategoryWindowDemo(t *testing.T) {
	u := ui.New(1280, 780)
	g := newGallery(u)
	if g.categories.IsOpen() {
		t.Fatal("categories must start hidden")
	}
	clickGalleryButton(t, u, "categoryButton")
	if !g.categories.IsOpen() || g.status != "Categories opened (demo)" {
		t.Fatalf("open: open=%v status=%q", g.categories.IsOpen(), g.status)
	}
	if id, ok := u.Lookup("categoryList").(*widgets.List).Selected(); !ok || id != "essences" {
		t.Fatalf("preselected = %q/%v", id, ok)
	}
	if !u.SelectListItem("categoryList", "herbs") || g.status != `Category "herbs" selected` {
		t.Fatalf("select: status=%q", g.status)
	}
	if !u.SetListExpanded("categoryList", "weapons", true) || g.status != `Category "weapons" expanded=true` {
		t.Fatalf("expand: status=%q", g.status)
	}
	clickGalleryButton(t, u, "sellButton")
	if !g.categories.IsOpen() || g.status != "Item listed for sale (demo)" {
		t.Fatalf("sell: open=%v status=%q", g.categories.IsOpen(), g.status)
	}
}

// TestCategoryWindowClose exercises the X button close path and reopen.
func TestCategoryWindowClose(t *testing.T) {
	u := ui.New(1280, 780)
	g := newGallery(u)
	clickGalleryButton(t, u, "categoryButton")
	if !g.categories.IsOpen() {
		t.Fatal("categories must open")
	}
	clickGalleryButton(t, u, "categoryWindow/close")
	if g.categories.IsOpen() || g.status != "Categories closed (demo)" {
		t.Fatalf("close: open=%v status=%q", g.categories.IsOpen(), g.status)
	}
	clickGalleryButton(t, u, "categoryButton")
	if !g.categories.IsOpen() || g.status != "Categories opened (demo)" {
		t.Fatalf("reopen: open=%v status=%q", g.categories.IsOpen(), g.status)
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
