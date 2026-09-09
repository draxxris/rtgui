package main

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/widgets"
)

// gallerySlot is one placement row. Empty owner means a screen root with
// absolute design coordinates and no points. Otherwise rect X/Y is the
// TopLeft offset inside the owner and W/H is the size.
type gallerySlot struct {
	name  string
	owner string
	rect  core.Rect
}

// gallerySlots authors the fixed logical design placement. Children carry
// owner-relative offsets, WoW FrameXML style, so ownership and anchors share
// one table. Window resize only scales; a new logical resolution needs a new
// table plus UI scale, like WoW options.
func gallerySlots(width, height float32) []gallerySlot {
	margin, gap := float32(28), float32(24)
	panelW := (width - 2*margin - gap) / 2
	panelH := height - 98
	rightX := margin + panelW + gap
	colW := panelW - 48
	const captionLift = float32(23)
	return []gallerySlot{
		{name: "leftPanel", rect: core.Rect{X: margin, Y: 74, W: panelW, H: panelH}},
		{name: "rightPanel", rect: core.Rect{X: rightX, Y: 74, W: panelW, H: panelH}},
		{name: "galleryTitle", rect: core.Rect{X: 28, Y: 24, W: 600, H: 26}},
		{name: "gallerySubtitle", rect: core.Rect{X: 30, Y: 51, W: 1200, H: 20}},
		{name: "statusLabel", rect: core.Rect{X: 30, Y: height - 18, W: width - 60, H: 20}},
		{name: "leftTitle", owner: "leftPanel", rect: core.Rect{X: 24, Y: 15, W: colW, H: 24}},
		{name: "primaryButton", owner: "leftPanel", rect: core.Rect{X: 24, Y: 48, W: colW, H: 42}},
		{name: "enableCheckbox", owner: "leftPanel", rect: core.Rect{X: 24, Y: 112, W: colW, H: 38}},
		{name: "textboxCaption", owner: "leftPanel", rect: core.Rect{X: 24, Y: 153, W: colW, H: 20}},
		{name: "inputTextbox", owner: "leftPanel", rect: core.Rect{X: 24, Y: 176, W: colW, H: 42}},
		{name: "dropdownCaption", owner: "leftPanel", rect: core.Rect{X: 24, Y: 217, W: colW, H: 20}},
		{name: "classDropdown", owner: "leftPanel", rect: core.Rect{X: 24, Y: 240, W: colW, H: 42}},
		{name: "sliderCaption", owner: "leftPanel", rect: core.Rect{X: 24, Y: 281, W: colW, H: 20}},
		{name: "valueSlider", owner: "leftPanel", rect: core.Rect{X: 24, Y: 304, W: colW, H: 42}},
		{name: "valueProgress", owner: "leftPanel", rect: core.Rect{X: 24, Y: 368, W: colW, H: 42}},
		{name: "demoPanel", owner: "leftPanel", rect: core.Rect{X: 24, Y: 440, W: colW, H: 70}},
		{name: "panelText", owner: "demoPanel", rect: core.Rect{X: 14, Y: 30, W: colW - 28, H: 24}},
		{name: "tabbarCaption", owner: "leftPanel", rect: core.Rect{X: 24, Y: 513, W: colW, H: 20}},
		{name: "demoTabs", owner: "leftPanel", rect: core.Rect{X: 24, Y: 536, W: colW, H: 36}},
		{name: "chatCaption", owner: "leftPanel", rect: core.Rect{X: 24, Y: 577, W: colW, H: 20}},
		{name: "chatMessage", owner: "leftPanel", rect: core.Rect{X: 24, Y: 600, W: colW, H: 68}},
		{name: "rightTitle", owner: "rightPanel", rect: core.Rect{X: 24, Y: 15, W: colW, H: 24}},
		{name: "demoLabel", owner: "rightPanel", rect: core.Rect{X: 24, Y: 40, W: colW, H: 32}},
		{name: "demoFrame", owner: "rightPanel", rect: core.Rect{X: 24, Y: 92, W: colW, H: 164}},
		{name: "frameCaption", owner: "demoFrame", rect: core.Rect{X: 18, Y: 20, W: colW - 36, H: 22}},
		{name: "frameChildButton", owner: "demoFrame", rect: core.Rect{X: 24, Y: 74, W: colW - 96, H: 42}},
		{name: "scrollCaption", owner: "rightPanel", rect: core.Rect{X: 36, Y: 260, W: colW - 24, H: 20}},
		{name: "scrollPanel", owner: "rightPanel", rect: core.Rect{X: 24, Y: 282, W: colW, H: 232}},
		{name: "marketGraph", owner: "rightPanel", rect: core.Rect{X: 24, Y: 282, W: colW, H: 232}},
		{name: "chatLog", owner: "rightPanel", rect: core.Rect{X: 24, Y: 282, W: colW, H: 232}},
		{name: "menuHint", owner: "rightPanel", rect: core.Rect{X: 36, Y: 524, W: colW - 24, H: 20}},
		{name: "tooltipHint", owner: "rightPanel", rect: core.Rect{X: 36, Y: 542, W: colW - 24, H: 20}},
		{name: "questButton", owner: "rightPanel", rect: core.Rect{X: 24, Y: 572, W: 264, H: 36}},
		{name: "categoryButton", owner: "rightPanel", rect: core.Rect{X: 296, Y: 572, W: 256, H: 36}},
		{name: "stateSamples", owner: "rightPanel", rect: core.Rect{X: 24, Y: panelH - 66, W: colW, H: 34}},
		{name: "questLog", rect: core.Rect{X: (width - 420) / 2, Y: (height - 320) / 2, W: 420, H: 320}},
		{name: "questText", owner: "questLog/content", rect: core.Rect{X: 16, Y: 12, W: 372, H: 150}},
		{name: "questAccept", owner: "questLog/content", rect: core.Rect{X: 16, Y: 190, W: 178, H: 44}},
		{name: "questDecline", owner: "questLog/content", rect: core.Rect{X: 210, Y: 190, W: 178, H: 44}},
		{name: "categoryWindow", rect: core.Rect{X: (width - 300) / 2, Y: (height - 440) / 2, W: 300, H: 440}},
		{name: "categoryList", owner: "categoryWindow/content", rect: core.Rect{X: 16, Y: 12, W: 252, H: 308}},
		{name: "sellButton", owner: "categoryWindow/content", rect: core.Rect{X: 16, Y: 332, W: 252, H: 44}},
	}
}

// applyLayout places every registered widget from the slot table, then
// resolves the ownership roots. Missing widgets are skipped so marketGraph
// lands on the pass after it is added.
func (g *gallery) applyLayout() {
	for _, slot := range g.slots {
		w := g.facade.Lookup(slot.name)
		if w == nil {
			continue
		}
		g.placeSlot(w, slot)
	}
	g.arrangeGalleryRoots()
	g.frameOrigin = core.Vec2{X: g.frame.Bounds().X, Y: g.frame.Bounds().Y}
}

// placeSlot owns, sizes, and anchors one row: roots stay absolute with no
// points, children carry exactly one TopLeft point against their owner.
func (g *gallery) placeSlot(w widgets.Widget, slot gallerySlot) {
	if slot.owner == "" {
		w.Frame().ClearAllPoints()
		w.SetBounds(slot.rect)
		return
	}
	if err := g.facade.SetParent(slot.name, slot.owner); err != nil {
		panic(err)
	}
	w.SetBounds(core.Rect{W: slot.rect.W, H: slot.rect.H})
	if err := w.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: slot.rect.X, Y: slot.rect.Y}); err != nil {
		panic(err)
	}
}

// arrangeGalleryRoots resolves every root that owns children, derived from
// the table instead of naming panels.
func (g *gallery) arrangeGalleryRoots() {
	for _, slot := range g.slots {
		if slot.owner != "" {
			continue
		}
		w := g.facade.Lookup(slot.name)
		if w == nil || w.Frame().ChildCount() == 0 {
			continue
		}
		if err := layout.Arrange(w.Frame(), core.Rect{}); err != nil {
			panic(err)
		}
	}
}
