package main

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/dragdrop"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/widgets"
)

// setupGameWidgets demonstrates cached graphs, rich tooltips, and drag ownership.
func (g *gallery) setupGameWidgets() {
	g.lineGraph = widgets.NewLineGraph("marketGraph", g.layout.scroll).SetMaxPoints(128).SetShowLabels(true)
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
func (g *gallery) setupItemDrag() {
	if err := g.facade.SetParent(g.panelText.Name(), g.panel.Name()); err != nil {
		panic(err)
	}
	g.panelText.SetBounds(core.Rect{X: 14, Y: 30, W: g.layout.panelText.W, H: g.layout.panelText.H})
	if err := layout.Arrange(g.panel.Frame(), core.Rect{}); err != nil {
		panic(err)
	}
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
