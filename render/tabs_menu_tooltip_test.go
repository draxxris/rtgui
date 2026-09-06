package render_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
)

// TestTabGeometryTilesContentExactly checks tab cells invert hit testing.
func TestTabGeometryTilesContentExactly(t *testing.T) {
	content := core.Rect{X: 10, Y: 20, W: 300, H: 36}
	for index := 0; index < 3; index++ {
		cell, ok := render.TabTabRect(content, 3, index)
		if !ok || cell.W != 100 || cell.X != 10+float32(index)*100 || cell.Y != 20 || cell.H != 36 {
			t.Fatalf("tab cell %d = %+v/%v", index, cell, ok)
		}
		middle := core.Vec2{X: cell.X + cell.W/2, Y: cell.Y + cell.H/2}
		if got := render.TabIndexAt(content, 3, middle); got != index {
			t.Fatalf("tab index at %+v = %d", middle, got)
		}
	}
	if got := render.TabIndexAt(content, 3, core.Vec2{X: 0, Y: 0}); got != -1 {
		t.Fatalf("outside tab index = %d", got)
	}
	if _, ok := render.TabTabRect(content, 0, 0); ok {
		t.Fatal("zero-count tab cell must fail")
	}
}

// TestMenuGeometryTilesRowsExactly checks menu rows invert hit testing.
func TestMenuGeometryTilesRowsExactly(t *testing.T) {
	outer := render.MenuOuterBounds(core.Vec2{X: 750, Y: 580}, core.Vec2{X: 800, Y: 600}, 3)
	if outer.X+outer.W > 800 || outer.Y+outer.H > 600 {
		t.Fatalf("menu outer not clamped: %+v", outer)
	}
	if outer.W != render.MenuWidth || outer.H != 3*render.MenuRowHeight {
		t.Fatalf("menu outer size = %+v", outer)
	}
	content := core.Rect{X: outer.X, Y: outer.Y, W: outer.W, H: outer.H}
	for index := 0; index < 3; index++ {
		row, ok := render.MenuRowRect(content, 3, index)
		if !ok || row.H != content.H/3 {
			t.Fatalf("menu row %d = %+v/%v", index, row, ok)
		}
		middle := core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
		if got := render.MenuIndexAt(content, 3, middle); got != index {
			t.Fatalf("menu index at %+v = %d", middle, got)
		}
	}
	if got := render.MenuIndexAt(content, 3, core.Vec2{X: -10, Y: -10}); got != -1 {
		t.Fatalf("outside menu index = %d", got)
	}
}

// TestTooltipLinesWrapAndClamp checks word wrap and viewport clamping.
func TestTooltipLinesWrapAndClamp(t *testing.T) {
	lines := render.TooltipLines("one two three four five six seven eight", 40, 14)
	if len(lines) < 2 {
		t.Fatalf("tooltip lines did not wrap: %q", lines)
	}
	if lines := render.TooltipLines("", 40, 14); len(lines) != 0 {
		t.Fatalf("empty tooltip lines = %q", lines)
	}
	bounds := render.TooltipOuterBounds("hello", core.Vec2{X: 790, Y: 590}, core.Vec2{X: 800, Y: 600}, 0)
	if bounds.W <= 0 || bounds.H <= 0 || bounds.X+bounds.W > 800 || bounds.Y+bounds.H > 600 {
		t.Fatalf("tooltip outer not clamped: %+v", bounds)
	}
	if bounds := render.TooltipOuterBounds("", core.Vec2{}, core.Vec2{X: 800, Y: 600}, 0); bounds != (core.Rect{}) {
		t.Fatalf("empty tooltip bounds = %+v", bounds)
	}
}
