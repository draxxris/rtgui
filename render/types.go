// Package render draws widget snapshots with raylib and owns skin, font, and
// optional bounded diagnostic-recording state.
package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// DrawCall is one recorded render operation.
type DrawCall struct {
	// Kind identifies the widget that produced the operation.
	Kind core.WidgetKind
	// Part identifies the rendered skin component.
	Part skin.SkinPart
	// State is the visual state used for lookup.
	State core.WidgetState
	// Bounds are the widget or part's logical bounds before snapping.
	Bounds core.Rect
	// Src is the selected atlas source rectangle.
	Src core.Rect
	// Dest is the logical destination rectangle after snapping.
	Dest core.Rect
	// Tint is the exact RGBA color passed to raylib.
	Tint core.Color
	// Fallback reports that no matching skin descriptor was found.
	Fallback bool
}
