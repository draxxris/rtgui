package main

import (
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
	g.lineGraph = widgets.NewLineGraph("marketGraph", core.Rect{}).SetMaxPoints(128).SetShowLabels(true)
	g.lineGraph.SetTextColor(core.Color{R: 220, G: 230, B: 240, A: 255})
	series := g.lineGraph.AddSeries(core.Color{R: 120, G: 210, B: 130, A: 255}, 2)
	g.lineGraph.SetSeriesData(series, []core.Vec2{{X: 0, Y: 12}, {X: 1, Y: 16}, {X: 2, Y: 13}, {X: 3, Y: 21}, {X: 4, Y: 19}, {X: 5, Y: 25}})
	if err := g.facade.Add(g.lineGraph); err != nil {
		panic(err)
	}
	g.facade.SetRichTooltip("marketGraph", core.RichTooltip{Title: "Market history", Subtitle: "Cached line graph", Segments: []core.RichSegment{{Text: "The Style tab shows numeric series with cached ticks and labels."}}})
	g.facade.SetRichTooltip(g.frameButton.Name(), core.RichTooltip{
		Title: "Thunderfury", Subtitle: "Legendary sword",
		HasTitleColor: true, TitleColor: core.Color{R: 255, G: 180, B: 70, A: 255},
		Segments: []core.RichSegment{{Text: "+12 Strength\n", HasColor: true, Color: core.Color{R: 120, G: 220, B: 130, A: 255}}, {Text: "Drag this button onto the left decoration panel. Escape cancels."}},
	})
	g.setupItemDrag()
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

// selectGraphPage shares the scroll region between ordinary rows and the graph.
func (g *gallery) selectGraphPage(index int) {
	if g.lineGraph == nil {
		return
	}
	g.lineGraph.SetVisible(index == 1)
	g.scroll.SetVisible(index != 1)
	text := "Scroll panel — wheel over this area"
	if index == 1 {
		text = "Line graph — cached price history"
	}
	g.scrollCaption.SetText(text)
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
