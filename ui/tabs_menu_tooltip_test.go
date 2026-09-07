package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// tabCellCenter resolves the skin-aware center of one tab cell for tests.
func tabCellCenter(u *UI, bar *widgets.TabBar, index int) core.Vec2 {
	content := u.Theme().TabContent(bar.Bounds(), core.StateNormal)
	cell, _ := render.TabTabRect(content, bar.TabCount(), index)
	return core.Vec2{X: cell.X + cell.W/2, Y: cell.Y + cell.H/2}
}

// menuRowCenter resolves the skin-aware center of one menu row for tests.
func menuRowCenter(u *UI, index int) core.Vec2 {
	bounds, _ := u.MenuBounds()
	content := u.Theme().MenuContent(bounds, core.StateNormal)
	row, _ := render.MenuRowRect(content, len(u.menuItems), index)
	return core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
}

// hasKindPart searches recorder calls for one kind and part.
func hasKindPart(calls []render.DrawCall, kind core.WidgetKind, part skin.SkinPart) bool {
	for _, call := range calls {
		if call.Kind == kind && call.Part == part {
			return true
		}
	}
	return false
}

// menuTestItems returns a stable mixed menu for tests.
func menuTestItems() []MenuItem {
	return []MenuItem{{ID: "a", Label: "Alpha"}, {ID: "b", Label: "Beta", Disabled: true}, {Separator: true}, {ID: "c", Label: "Gamma"}}
}

// openTestMenu shows a menu that selects into got.
func openTestMenu(u *UI, got *string) {
	u.ShowContextMenu(menuTestItems(), core.Vec2{X: 100, Y: 100}, func(id string) { *got = id })
}

// TestTabBarSelectsThroughPressRelease checks physical tab selection.
func TestTabBarSelectsThroughPressRelease(t *testing.T) {
	u := New(400, 200)
	bar := widgets.NewTabBar("tabs", core.Rect{X: 20, Y: 20, W: 300, H: 36}, []string{"A", "B", "C"}, 0)
	mustAdd(t, u, bar)
	var selected []int
	clicks := 0
	u.OnTabSelect("tabs", func(index int) { selected = append(selected, index) })
	u.OnClick("tabs", func() { clicks++ })
	clickAt(u, tabCellCenter(u, bar, 2))
	if bar.SelectedTab() != 2 || len(selected) != 1 || clicks != 1 {
		t.Fatalf("tab click = %d/%v/%d", bar.SelectedTab(), selected, clicks)
	}
	clickAt(u, tabCellCenter(u, bar, 2))
	if len(selected) != 1 || clicks != 2 {
		t.Fatalf("same-tab click fired select: %v/%d", selected, clicks)
	}
}

// TestTabBarSemanticsSelectsWithoutPointer checks SelectTab and Activate.
func TestTabBarSemanticsSelectsWithoutPointer(t *testing.T) {
	u := New(400, 200)
	bar := widgets.NewTabBar("tabs", core.Rect{X: 20, Y: 20, W: 300, H: 36}, []string{"A", "B", "C"}, 0)
	mustAdd(t, u, bar)
	var selected []int
	clicks := 0
	u.OnTabSelect("tabs", func(index int) { selected = append(selected, index) })
	u.OnClick("tabs", func() { clicks++ })
	if !u.SelectTab("tabs", 1) || bar.SelectedTab() != 1 || len(selected) != 1 {
		t.Fatalf("semantic select = %d/%v", bar.SelectedTab(), selected)
	}
	if u.SelectTab("tabs", 9) || u.SelectTab("missing", 0) || u.SelectTab("tabs", -1) {
		t.Fatal("invalid semantic select accepted")
	}
	if !u.Activate("tabs") || clicks != 2 {
		t.Fatalf("semantic activate clicks = %d", clicks)
	}
	bar.SetEnabled(false)
	if u.SelectTab("tabs", 0) || u.Activate("tabs") {
		t.Fatal("disabled tab bar accepted semantics")
	}
}

// TestContextMenuCommitsAndDrawsAboveAppLayers checks selection and layering.
func TestContextMenuCommitsAndDrawsAboveAppLayers(t *testing.T) {
	u := New(800, 600)
	got := ""
	openTestMenu(u, &got)
	if !u.HasOpenMenu() {
		t.Fatal("menu did not open")
	}
	point := menuRowCenter(u, 0)
	if u.HandleMouse(MouseEvent{Pos: point}) {
		t.Fatal("menu hover alone must not consume")
	}
	if !u.HandleMouse(MouseEvent{Pos: point, Wheel: 1}) {
		t.Fatal("menu wheel must be consumed")
	}
	recorder := attachDrawRecorder(t, u)
	u.DrawWidgets()
	for _, call := range recorder.Calls() {
		if call.Kind == core.WidgetMenu {
			t.Fatal("DrawWidgets emitted menu calls")
		}
	}
	u.DrawPopup()
	calls := recorder.Calls()
	if !hasKindPart(calls, core.WidgetMenu, skin.PartPopup) || !hasKindPart(calls, core.WidgetMenu, skin.PartOverlay) {
		t.Fatal("DrawPopup did not render menu shell and hovered row")
	}
	if !u.HandleMouse(MouseEvent{Pos: point, Pressed: true}) || !u.HandleMouse(MouseEvent{Pos: point, Released: true}) {
		t.Fatal("menu row gesture was not consumed")
	}
	if got != "a" || u.HasOpenMenu() {
		t.Fatalf("menu commit = %q open=%v", got, u.HasOpenMenu())
	}
}

// TestContextMenuDismissesWithoutSelection checks disabled rows and Escape.
func TestContextMenuDismissesWithoutSelection(t *testing.T) {
	u := New(800, 600)
	got := ""
	openTestMenu(u, &got)
	disabled := menuRowCenter(u, 1)
	u.HandleMouse(MouseEvent{Pos: disabled, Pressed: true})
	u.HandleMouse(MouseEvent{Pos: disabled, Released: true})
	if !u.HasOpenMenu() || got == "a" {
		t.Fatal("disabled row must keep the menu open without callback")
	}
	if !u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 700, Y: 500}, Pressed: true}) || u.HasOpenMenu() {
		t.Fatal("outside press must dismiss the menu")
	}
	openTestMenu(u, &got)
	if !u.HandleKey(KeyEvent{Escape: true}) || u.HasOpenMenu() {
		t.Fatal("Escape must close the menu")
	}
	u.ShowContextMenu(nil, core.Vec2{}, nil)
	if u.HasOpenMenu() {
		t.Fatal("empty menu must not open")
	}
	var nilUI *UI
	nilUI.ShowContextMenu(menuTestItems(), core.Vec2{}, nil)
	if nilUI.HasOpenMenu() || nilUI.CloseMenu() || nilUI.HideTooltip() {
		t.Fatal("nil UI menu/tooltip answered data")
	}
}

// TestTooltipDrawsOnHover checks hover-derived visibility.
func TestTooltipDrawsOnHover(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetTooltip("slot", "Iron Sword")
	if text, ok := u.TooltipText("slot"); !ok || text != "Iron Sword" {
		t.Fatalf("tooltip mapping = %q/%v", text, ok)
	}
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	recorder := attachDrawRecorder(t, u)
	u.DrawWidgets()
	u.DrawPopup()
	if !hasKindPart(recorder.Calls(), core.WidgetTooltip, skin.PartText) {
		t.Fatal("hovered tooltip did not draw")
	}
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 700, Y: 500}})
	moved := attachDrawRecorder(t, u)
	u.DrawWidgets()
	u.DrawPopup()
	if hasKindPart(moved.Calls(), core.WidgetTooltip, skin.PartText) {
		t.Fatal("tooltip drew without hover")
	}
	u.SetTooltip("slot", "")
	if _, ok := u.TooltipText("slot"); ok {
		t.Fatal("cleared tooltip mapping survived")
	}
}

// TestTooltipExplicitShowsUntilInteraction checks positioned tooltips.
func TestTooltipExplicitShowsUntilInteraction(t *testing.T) {
	u := New(800, 600)
	recorder := attachDrawRecorder(t, u)
	u.ShowTooltip("Pinned note", core.Vec2{X: 400, Y: 400})
	u.DrawWidgets()
	u.DrawPopup()
	if !hasKindPart(recorder.Calls(), core.WidgetTooltip, skin.PartText) {
		t.Fatal("explicit tooltip did not draw")
	}
	if !u.HandleKey(KeyEvent{Escape: true}) || u.tooltipText != "" {
		t.Fatal("Escape must hide the explicit tooltip")
	}
	u.ShowTooltip("Pinned note", core.Vec2{X: 400, Y: 400})
	u.HandleMouse(MouseEvent{Pos: core.Vec2{X: 400, Y: 400}, Pressed: true})
	if u.tooltipText != "" {
		t.Fatal("press must hide the explicit tooltip")
	}
}
