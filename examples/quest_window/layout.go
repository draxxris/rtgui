// Package main implements the MMORPG quest window example.
package main

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/widgets"
)

// questSlot is one placement row. Empty owner means a screen root with
// absolute design coordinates and no points. Otherwise rect X/Y is the
// TopLeft offset inside the owner and W/H is the size.
type questSlot struct {
	name  string
	owner string
	rect  core.Rect
}

// questSlots authors the fixed 800x800 logical placement matching the
// reference journal. The root carries the synthetic 24px margin, so the
// viewport stays full-window. Children carry owner-relative offsets, WoW
// FrameXML style, so ownership and anchors share one table. TitledFrame
// chrome comes from its own insets (full-bleed 7px titlebar sides under
// the root moldings, 11px top, 42px titlebar, 11px divider gap, 15px
// content sides), so content children start below the v6 title band at
// content-relative coordinates in a 722x674 well. The root border paints
// after children, so the moldings cover the titlebar and content edges;
// the close button stays inside the titlebar and fills its height.
func questSlots() []questSlot {
	return []questSlot{
		{name: "questWindow", rect: core.Rect{X: 24, Y: 24, W: 752, H: 752}},
		{name: "questTabs", owner: "questWindow/content", rect: core.Rect{X: 15, Y: 9, W: 272, H: 35}},
		{name: "questList", owner: "questWindow/content", rect: core.Rect{X: 9, Y: 57, W: 283, H: 531}},
		{name: "questSheet", owner: "questWindow/content", rect: core.Rect{X: 300, Y: 9, W: 411, H: 583}},
		{name: "questRules", owner: "questSheet", rect: core.Rect{W: 411, H: 583}},
		{name: "questTitle", owner: "questSheet", rect: core.Rect{X: 26, Y: 6, W: 372, H: 29}},
		{name: "questSubtitle", owner: "questSheet", rect: core.Rect{X: 26, Y: 38, W: 372, H: 18}},
		{name: "questBody", owner: "questSheet", rect: core.Rect{X: 26, Y: 62, W: 372, H: 340}},
		{name: "questImage", owner: "questSheet", rect: core.Rect{X: 26, Y: 408, W: 372, H: 64}},
		{name: "questRewards", owner: "questSheet", rect: core.Rect{X: 26, Y: 478, W: 372, H: 92}},
		{name: "questRewardArt", owner: "questSheet", rect: core.Rect{W: 411, H: 583}},
		{name: "questReward1", owner: "questSheet", rect: core.Rect{X: 79, Y: 526, W: 73, H: 51}},
		{name: "questReward2", owner: "questSheet", rect: core.Rect{X: 205, Y: 526, W: 59, H: 51}},
		{name: "questReward3", owner: "questSheet", rect: core.Rect{X: 314, Y: 526, W: 85, H: 51}},
		{name: "showMapButton", owner: "questWindow/content", rect: core.Rect{X: 73, Y: 616, W: 154, H: 37}},
		{name: "shareButton", owner: "questWindow/content", rect: core.Rect{X: 241, Y: 616, W: 110, H: 37}},
		{name: "abandonButton", owner: "questWindow/content", rect: core.Rect{X: 365, Y: 616, W: 120, H: 37}},
		{name: "trackButton", owner: "questWindow/content", rect: core.Rect{X: 499, Y: 616, W: 150, H: 37}},
		{name: "reopenButton", rect: core.Rect{X: 300, Y: 376, W: 200, H: 48}},
	}
}

// applyQuestLayout places every registered widget from the slot table, then
// resolves the ownership roots. Missing widgets are skipped so construction
// order never matters.
func (q *questWindow) applyQuestLayout() {
	if q == nil || q.facade == nil {
		return
	}
	for _, slot := range questSlots() {
		w := q.facade.Lookup(slot.name)
		if w == nil {
			continue
		}
		q.placeQuestSlot(w, slot)
	}
	q.arrangeQuestRoots()
}

// placeQuestSlot owns, sizes, and anchors one row: roots stay absolute with
// no points, children carry exactly one TopLeft point against their owner.
func (q *questWindow) placeQuestSlot(w widgets.Widget, slot questSlot) {
	if q == nil || q.facade == nil || w == nil {
		return
	}
	if slot.owner == "" {
		w.Frame().ClearAllPoints()
		w.SetBounds(slot.rect)
		return
	}
	if err := q.facade.SetParent(slot.name, slot.owner); err != nil {
		panic(err)
	}
	w.SetBounds(core.Rect{W: slot.rect.W, H: slot.rect.H})
	if err := w.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: slot.rect.X, Y: slot.rect.Y}); err != nil {
		panic(err)
	}
}

// arrangeQuestRoots resolves the titled window tree plus every root that
// owns children, derived from the table instead of naming panels.
func (q *questWindow) arrangeQuestRoots() {
	if q == nil || q.facade == nil {
		return
	}
	if q.titled != nil {
		if err := q.titled.Layout(); err != nil {
			panic(err)
		}
	}
	q.arrangeExtraRoots()
	if q.titled != nil {
		if err := q.titled.Arrange(); err != nil {
			panic(err)
		}
	}
}

// arrangeExtraRoots resolves every non-titled root that owns children.
// The titled window arranges separately so its chrome and content stay in
// one Layout/Arrange pair.
func (q *questWindow) arrangeExtraRoots() {
	if q == nil || q.facade == nil {
		return
	}
	seen := map[string]bool{}
	for _, slot := range questSlots() {
		if slot.owner != "" || seen[slot.name] {
			continue
		}
		seen[slot.name] = true
		q.arrangeOneRoot(slot.name)
	}
}

// arrangeOneRoot resolves one ownership root unless it is the titled
// window itself, which owns its chrome through TitledFrame layout.
func (q *questWindow) arrangeOneRoot(name string) {
	if q == nil || q.facade == nil {
		return
	}
	if q.titled != nil && name == "questWindow" {
		return
	}
	w := q.facade.Lookup(name)
	if w == nil || w.Frame().ChildCount() == 0 {
		return
	}
	if err := layout.Arrange(w.Frame(), core.Rect{}); err != nil {
		panic(err)
	}
}
