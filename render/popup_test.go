package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

// TestDropdownPopupRowsTileContent verifies the single row formula shared
// by hit testing and drawing: rows tile content exactly, centers invert,
// and outside or degenerate inputs miss.
func TestDropdownPopupRowsTileContent(t *testing.T) {
	content := core.Rect{X: 10, Y: 66, W: 180, H: 108}
	for index := 0; index < 3; index++ {
		row, ok := DropdownPopupRow(content, 3, index)
		if !ok {
			t.Fatalf("row %d missing", index)
		}
		want := core.Rect{X: 10, Y: 66 + float32(index*36), W: 180, H: 36}
		if row != want {
			t.Fatalf("row %d = %+v, want %+v", index, row, want)
		}
		center := core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
		if got := DropdownPopupIndex(content, 3, center); got != index {
			t.Fatalf("row %d center maps to %d", index, got)
		}
	}
	if _, ok := DropdownPopupRow(content, 3, 3); ok {
		t.Fatal("out-of-range row must miss")
	}
	if _, ok := DropdownPopupRow(content, 0, 0); ok {
		t.Fatal("empty popup must have no rows")
	}
	for _, point := range []core.Vec2{{X: 0, Y: 0}, {X: 20, Y: 65.9}, {X: 20, Y: 174}, {X: 9.9, Y: 100}, {X: 190.1, Y: 100}} {
		if got := DropdownPopupIndex(content, 3, point); got != -1 {
			t.Fatalf("outside point %+v maps to %d", point, got)
		}
	}
	if got := DropdownPopupIndex(content, 0, core.Vec2{X: 20, Y: 100}); got != -1 {
		t.Fatalf("empty popup index = %d", got)
	}
}

// TestDropdownPopupContentAppliesSkinInsets verifies hit testing and
// drawing share one skin-aware content area, and that the drawn highlight
// for a hit-tested row lands exactly on that row.
func TestDropdownPopupContentAppliesSkinInsets(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	popup := core.Rect{X: 10, Y: 66, W: 180, H: 108}
	if got := theme.DropdownPopupContent(popup, core.StatePressed); got != popup {
		t.Fatalf("unskinned content = %+v, want %+v", got, popup)
	}
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetDropdown, Part: skin.PartPopup, State: core.StateNormal}, skin.SkinDescriptor{
		PaddingLeft: 4, PaddingTop: 4, PaddingRight: 4, PaddingBottom: 4,
	})
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetDropdown, Part: skin.PartPopupBorder, State: core.StateNormal}, skin.SkinDescriptor{
		HasNinePatch: true, NinePatch: skin.NinePatch{Left: 8, Top: 8, Right: 8, Bottom: 8},
	})
	content := theme.DropdownPopupContent(popup, core.StatePressed)
	want := core.Rect{X: 18, Y: 74, W: 164, H: 92}
	if content != want {
		t.Fatalf("skinned content = %+v, want %+v", content, want)
	}
	row, _ := DropdownPopupRow(content, 3, 1)
	center := core.Vec2{X: row.X + row.W/2, Y: row.Y + row.H/2}
	hovered := DropdownPopupIndex(content, 3, center)
	if hovered != 1 {
		t.Fatalf("skinned hover maps to %d, want 1", hovered)
	}
	// The old full-bounds formula would map this center to row 1 at a
	// different Y; the shared content formula must agree with the draw.
	recorder := newTestRecorder(t, 16)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	info := core.WidgetInfo{Name: "class", Bounds: popup, Kind: core.WidgetDropdown, State: core.StatePressed}
	theme.DrawDropdownPopup(info, []string{"Warrior", "Ranger", "Mage"}, hovered)
	found := false
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartOverlay && call.State == core.StateHovered {
			found = true
			if call.Bounds != row {
				t.Fatalf("highlight bounds = %+v, want %+v", call.Bounds, row)
			}
		}
	}
	if !found {
		t.Fatal("hovered row drew no highlight")
	}
}

// TestDropdownPopupHelpersAreNilSafeAndAllocationFree protects the input
// hot path: hover and press compute skin-aware rows on every pointer
// event, so content and index resolution must not touch the heap.
func TestDropdownPopupHelpersAreNilSafeAndAllocationFree(t *testing.T) {
	var nilTheme *Theme
	popup := core.Rect{X: 10, Y: 66, W: 180, H: 108}
	if got := nilTheme.DropdownPopupContent(popup, core.StatePressed); got != popup {
		t.Fatalf("nil theme content = %+v, want %+v", got, popup)
	}
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.SetDrawRecorder(nil)
	point := core.Vec2{X: 20, Y: 110}
	allocations := testing.AllocsPerRun(200, func() {
		content := theme.DropdownPopupContent(popup, core.StatePressed)
		_ = DropdownPopupIndex(content, 3, point)
	})
	if allocations != 0 {
		t.Fatalf("popup hit-test allocations = %v, want 0", allocations)
	}
}
