package main

import (
	"fmt"
	"image/color"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/dragdrop"
	"github.com/draxxris/rtgui/text"
	"github.com/draxxris/rtgui/widgets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// setupGameWidgets demonstrates cached graphs, rich tooltips, and drag ownership.
// marketGraph placement comes from the slot table on the next applyLayout pass.
func (g *gallery) setupGameWidgets() {
	g.setupRichDemo()
	g.lineGraph = widgets.NewLineGraph("marketGraph", core.Rect{}).SetMaxPoints(128).SetShowGrid(false).SetShowLabels(false)
	g.lineGraph.SetTextColor(core.Color{R: 220, G: 230, B: 240, A: 255})
	series := g.lineGraph.AddSeries(core.Color{R: 120, G: 210, B: 130, A: 255}, 2)
	g.lineGraph.SetSeriesFX(series, widgets.LineSeriesFX{FillEnabled: true, GlowEnabled: true})
	g.lineGraph.SetSeriesData(series, []core.Vec2{{X: 0, Y: 12}, {X: 1, Y: 16}, {X: 2, Y: 13}, {X: 3, Y: 21}, {X: 4, Y: 19}, {X: 5, Y: 25}})
	if err := g.facade.Add(g.lineGraph); err != nil {
		panic(err)
	}
	g.facade.SetRichTooltip("marketGraph", core.RichTooltip{Title: "Market history", Subtitle: "Cached line graph", Segments: []core.RichSegment{{Text: "The Style tab shows numeric series with cached ticks and labels."}}})
	g.facade.SetRichTooltip(g.frameButton.Name(), core.RichTooltip{
		Title: "Thunderfury", Subtitle: "Legendary sword", Class: "item",
		HasTitleColor: true, TitleColor: core.Color{R: 255, G: 180, B: 70, A: 255},
		Segments: []core.RichSegment{{Text: "+12 Strength\n", HasColor: true, Color: core.Color{R: 120, G: 220, B: 130, A: 255}}, {Text: "Drag this button onto the left decoration panel. Escape cancels."}},
	})
	g.setupItemDrag()
	g.setupChatLog()
}

// categoryDemoItems returns the auction-house category tree for the floating
// Categories window. Materials starts expanded with Essences preselected;
// every other category starts collapsed. Icons reuse the demo whitelist.
func categoryDemoItems() []widgets.ListItem {
	icon := "iron-plate"
	return []widgets.ListItem{
		{ID: "fav", Label: "Favorites", Icon: icon},
		{ID: "weapons", Label: "Weapons", Icon: icon, Children: []widgets.ListItem{
			{ID: "blades", Label: "Blades"},
			{ID: "blunts", Label: "Blunts"},
		}},
		{ID: "armor", Label: "Armor", Icon: icon, Children: []widgets.ListItem{
			{ID: "plate", Label: "Plate"},
			{ID: "mail", Label: "Mail"},
		}},
		{ID: "consumables", Label: "Consumables", Icon: icon, Children: []widgets.ListItem{
			{ID: "potions", Label: "Potions"},
			{ID: "food", Label: "Food"},
		}},
		{ID: "mats", Label: "Materials", Icon: icon, Expanded: true, Children: []widgets.ListItem{
			{ID: "herbs", Label: "Herbs"},
			{ID: "ore", Label: "Ore"},
			{ID: "leather", Label: "Leather"},
			{ID: "cloth", Label: "Cloth"},
			{ID: "essences", Label: "Essences"},
			{ID: "gems", Label: "Gems"},
			{ID: "enchants", Label: "Enchants"},
		}},
		{ID: "recipes", Label: "Recipes", Icon: icon, Children: []widgets.ListItem{
			{ID: "weapon-plans", Label: "Weapon Plans"},
			{ID: "armor-patterns", Label: "Armor Patterns"},
		}},
		{ID: "pets", Label: "Pets", Icon: icon, Children: []widgets.ListItem{
			{ID: "companions", Label: "Companions"},
			{ID: "mounts", Label: "Mounts"},
		}},
		{ID: "misc", Label: "Miscellaneous", Icon: icon, Children: []widgets.ListItem{
			{ID: "quest-items", Label: "Quest Items"},
			{ID: "junk", Label: "Junk"},
		}},
	}
}

// setupItemDrag binds a source and target without application-side pointer routing.
// Placement stays in the slot table; this only owns drag intent.
func (g *gallery) setupItemDrag() {
	g.facade.OnDrag(g.frameButton.Name(), func() dragdrop.Payload { return dragdrop.NewPayload("sword", "item", 1) })
	g.facade.OnDrop(g.panel.Name(), func(p dragdrop.Payload) bool { return p.Kind == "item" }, func(dragdrop.Payload) { g.setStatus("Item drop delivered once; the game owns inventory validation") })
	g.facade.SetDragGhostDrawer(func(ghost dragdrop.Ghost) {
		info := g.frameButton.Snapshot(core.StatePressed)
		info.Bounds = core.Rect{X: ghost.Pos.X + 12, Y: ghost.Pos.Y + 12, W: 120, H: 32}
		g.facade.Theme().DrawWidget(info, "Sword", 0, false)
	})
}

// selectSharedRegion swaps the right-panel region between scroll rows,
// the graph, and the chat log. Index-aligned with demoTabs labels:
// Widgets shows the scroll panel, Style the line graph, About the chat log.
func (g *gallery) selectSharedRegion(index int) {
	if g.lineGraph == nil || g.chatLog == nil {
		return
	}
	g.scroll.SetVisible(index == 0)
	g.lineGraph.SetVisible(index == 1)
	g.chatLog.SetVisible(index == 2)
	text := "Scroll panel — wheel over this area"
	if index == 1 {
		text = "Line graph — cached price history"
	}
	if index == 2 {
		text = "Chat log — wheel to scroll, links clickable"
	}
	g.scrollCaption.SetText(text)
}

// setupChatLog feeds the About-tab chat log through the same player markup
// and icon whitelist as the chat message. The gallery acts as the game: it
// owns appends, link clicks, and tooltips while the log owns history.
func (g *gallery) setupChatLog() {
	if g.chatLog == nil {
		return
	}
	allowed := func(name string) bool {
		_, ok := g.facade.LookupInlineIcon(name)
		return ok
	}
	feed := []string{
		"Guild: need [link=item:iron-plate]iron plate[/link] [icon=iron-plate] — whisper [link=player:Mor'nor]Mor'nor[/link].",
		"[link=player:Mor'nor]Mor'nor[/link]: crafting [link=item:iron-plate]iron plate[/link] x20, meet at the forge.",
		"System: welcome to the gallery — scroll up to break the stick, scroll down to re-glue.",
		"Party: see [link=url:https://example.com/guide]the wiki[/link] for tonight's route.",
		"Guild: Thunderfury bindings drop in Molten Core — roll need.",
		"[link=player:Mor'nor]Mor'nor[/link]: [icon=iron-plate] sold out, farming more.",
	}
	for _, line := range feed {
		g.chatLog.AddMessage(text.ParsePlayerMarkup(line, allowed))
	}
	g.chatLog.OnLinkClick(func(link core.Link) {
		g.setStatus(fmt.Sprintf("ChatLog link %s %q", link.Kind, link.Target))
	}).OnLinkTooltipRequested(func(link core.Link) string {
		switch link.Kind {
		case core.LinkItem:
			return "Item — " + link.Target
		case core.LinkPlayer:
			return link.Target + " — Level 60 Warrior"
		case core.LinkURL:
			return "Open " + link.Target + " in browser"
		default:
			return ""
		}
	}).SetTooltip("Chat log — wheel to scroll, click a link")
}

// setupRichDemo registers the parent-owned icon whitelist, named faces, and
// per-kind link colors, then drives chat, button, and dropdown rows through
// one parser plus Go-authored style runs. The gallery acts as the game: it
// owns OnLinkClick, tooltips, and colors.
func (g *gallery) setupRichDemo() {
	g.facade.SetLinkColor(core.LinkURL, core.Color{R: 100, G: 170, B: 255, A: 255})
	g.facade.SetLinkColor(core.LinkItem, core.Color{R: 255, G: 180, B: 70, A: 255})
	g.facade.SetLinkColor(core.LinkPlayer, core.Color{R: 120, G: 220, B: 130, A: 255})
	icon := makeIronPlateIcon()
	g.facade.RegisterInlineIcon("iron-plate", icon)
	g.facade.RegisterInlineIcon("item/iron-plate", icon)
	if path := findFontFile("ValleySans-Regular.ttf"); path != "" {
		if err := g.facade.RegisterFont("ValleySans", path); err != nil {
			g.setStatus("ValleySans registration failed; font demo falls back")
		}
	}
	if path := findFontFile("ValleySans-Italic.ttf"); path != "" {
		if err := g.facade.RegisterFont("ValleySans-Italic", path); err != nil {
			g.setStatus("ValleySans-Italic registration failed; italic demo falls back")
		}
	}
	if g.chat != nil {
		allowed := func(name string) bool {
			_, ok := g.facade.LookupInlineIcon(name)
			return ok
		}
		g.chat.SetRichSegments(append(text.ParsePlayerMarkup("Guild: need [link=item:iron-plate]iron plate[/link] [icon=iron-plate] — whisper [link=player:Mor'nor]Mor'nor[/link] or see [link=url:https://example.com/guide]the wiki[/link].", allowed),
			core.RichSegment{Text: " BOLD", Bold: true},
			core.RichSegment{Text: " big", HasFontSize: true, FontSize: 26},
			core.RichSegment{Text: " valley", HasFont: true, Font: "ValleySans"},
			core.RichSegment{Text: " italic", HasFont: true, Font: "ValleySans-Italic"},
		))
	}
	if g.frameButton != nil {
		g.frameButton.SetRichSegments([]core.RichSegment{{Icon: "iron-plate", HasIcon: true}, {Text: " Frame child", Bold: true}})
	}
	if g.dropdown != nil {
		g.dropdown.SetRichDropdownItem(0, []core.RichSegment{{Icon: "iron-plate", HasIcon: true}, {Text: " Warrior", HasFont: true, Font: "ValleySans"}})
	}
}

// makeIronPlateIcon builds a procedural placeholder plate for the demo.
// Factorio art is proprietary, so the gallery never commits it; this gray
// plate proves whitelist lookup, layout, and draw without licensed pixels.
func makeIronPlateIcon() core.TooltipIcon {
	const size = int32(32)
	if !rl.IsWindowReady() {
		return core.TooltipIcon{Width: size, Height: size}
	}
	img := rl.GenImageColor(int(size), int(size), color.RGBA{R: 178, G: 188, B: 198, A: 255})
	if img == nil {
		return core.TooltipIcon{Width: size, Height: size}
	}
	defer rl.UnloadImage(img)
	rl.ImageDrawRectangle(img, 0, 0, size, 4, color.RGBA{R: 120, G: 130, B: 140, A: 255})
	rl.ImageDrawRectangle(img, 0, size-4, size, 4, color.RGBA{R: 120, G: 130, B: 140, A: 255})
	rl.ImageDrawRectangle(img, 0, 0, 4, size, color.RGBA{R: 120, G: 130, B: 140, A: 255})
	rl.ImageDrawRectangle(img, size-4, 0, 4, size, color.RGBA{R: 120, G: 130, B: 140, A: 255})
	rl.ImageDrawRectangle(img, 8, 12, size-16, 8, color.RGBA{R: 225, G: 232, B: 238, A: 255})
	texture := rl.LoadTextureFromImage(img)
	return core.TooltipIcon{ID: texture.ID, Width: texture.Width, Height: texture.Height}
}
