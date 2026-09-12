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

// TestGalleryCSSParsesTableParts checks the table shell, fixed header, row
// highlight, optional stripe, and scrollbar vocabulary used by the market.
func TestGalleryCSSParsesTableParts(t *testing.T) {
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
		if rule.Kind != core.WidgetTable {
			continue
		}
		switch rule.Part {
		case skin.PartBackground:
			seen["base"] = true
		case skin.PartHeader:
			seen["header"] = true
		case skin.PartOverlay:
			seen["highlight"] = true
		case skin.PartStripe:
			seen["stripe"] = true
		case skin.PartTrack:
			seen["track"] = true
		case skin.PartThumb:
			seen["thumb"] = true
		case skin.PartArrow:
			seen["arrow"] = true
		}
	}
	for _, part := range []string{"base", "header", "highlight", "stripe", "track", "thumb", "arrow"} {
		if !seen[part] {
			t.Fatalf("gallery css misses Table %s (seen=%v)", part, seen)
		}
	}
}

// TestAuctionTableDemo exercises the market popup, stable selection, both
// sortable numeric columns, explicit game activation, invalid-row behavior,
// and the hidden root's input blocking boundary.
func TestAuctionTableDemo(t *testing.T) {
	u := ui.New(1280, 780)
	g := newGallery(u)
	assertMarketStartsHidden(t, u, g)
	clickGalleryButton(t, u, "marketButton")
	assertMarketOpened(t, g)
	exerciseAuctionHeaderSort(t, u, g)
	exerciseAuctionActions(t, u, g)
	clickGalleryButton(t, u, "marketWindow/close")
	if g.marketWindow.IsOpen() || g.status != "Market closed (demo)" {
		t.Fatalf("close: open=%v status=%q", g.marketWindow.IsOpen(), g.status)
	}
}

// assertMarketStartsHidden verifies that a registered hidden popup neither
// opens accidentally nor blocks the visible scroll surface underneath it.
func assertMarketStartsHidden(t *testing.T, u *ui.UI, g *gallery) {
	t.Helper()
	if g.marketWindow.IsOpen() {
		t.Fatal("market must start hidden")
	}
	u.HandleMouse(ui.MouseEvent{Pos: core.Vec2{X: 700, Y: 390}})
	if hovered := u.Hovered(); hovered == nil || hovered.Name() != "scrollPanel" {
		t.Fatalf("hidden market blocked underlying hover: %#v", hovered)
	}
	if g.auctionTable.RowCount() != 8 {
		t.Fatalf("auction row count = %d, want 8", g.auctionTable.RowCount())
	}
	if got, ok := g.auctionTable.CellAt("cinder-amulet", "buyout"); !ok || !got.Invalid {
		t.Fatalf("invalid auction row missing invalid buyout: %+v/%v", got, ok)
	}
}

// assertMarketOpened checks the demo button's popup transition and status.
func assertMarketOpened(t *testing.T, g *gallery) {
	t.Helper()
	if !g.marketWindow.IsOpen() || g.status != "Market opened (demo)" {
		t.Fatalf("open: open=%v status=%q", g.marketWindow.IsOpen(), g.status)
	}
}

// exerciseAuctionHeaderSort drives the actual header press-release path twice
// and verifies the required ascending/descending toggle.
func exerciseAuctionHeaderSort(t *testing.T, u *ui.UI, g *gallery) {
	t.Helper()
	tableBounds := g.auctionTable.Bounds()
	headerBuyout := core.Vec2{X: tableBounds.X + tableBounds.W - 40, Y: tableBounds.Y + 12}
	u.HandleMouse(ui.MouseEvent{Pos: headerBuyout, Pressed: true})
	u.HandleMouse(ui.MouseEvent{Pos: headerBuyout, Released: true})
	if column, direction := g.auctionTable.SortColumn(); column != "buyout" || direction != core.SortAsc {
		t.Fatalf("header buyout asc = %q/%v", column, direction)
	}
	u.HandleMouse(ui.MouseEvent{Pos: headerBuyout, Pressed: true})
	u.HandleMouse(ui.MouseEvent{Pos: headerBuyout, Released: true})
	if column, direction := g.auctionTable.SortColumn(); column != "buyout" || direction != core.SortDesc {
		t.Fatalf("header buyout desc = %q/%v", column, direction)
	}
}

// exerciseAuctionActions covers stable selection, both numeric sort columns,
// and explicit activation of the invalid-but-actionable listing.
func exerciseAuctionActions(t *testing.T, u *ui.UI, g *gallery) {
	t.Helper()
	if !u.SelectTableRow("auctionTable", "cinder-amulet") || g.status != `Auction "cinder-amulet" selected` {
		t.Fatalf("select invalid row: status=%q", g.status)
	}
	if !u.SetTableSort("auctionTable", "buyout", core.SortAsc) || g.status != "Auction sorted by buyout ascending" {
		t.Fatalf("buyout asc: status=%q", g.status)
	}
	if column, direction := g.auctionTable.SortColumn(); column != "buyout" || direction != core.SortAsc {
		t.Fatalf("buyout asc state = %q/%v", column, direction)
	}
	if !u.SetTableSort("auctionTable", "buyout", core.SortDesc) || g.status != "Auction sorted by buyout descending" {
		t.Fatalf("buyout desc: status=%q", g.status)
	}
	if !u.SetTableSort("auctionTable", "level", core.SortAsc) || g.status != "Auction sorted by level ascending" {
		t.Fatalf("level asc: status=%q", g.status)
	}
	if !u.ActivateTableRow("auctionTable", "cinder-amulet") || g.status != `Bid/Buy "cinder-amulet" (demo)` {
		t.Fatalf("semantic activation: status=%q", g.status)
	}
	clickGalleryButton(t, u, "buyButton")
	if g.status != `Bid/Buy "cinder-amulet" (demo)` {
		t.Fatalf("button activation: status=%q", g.status)
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
