package render

import (
	"rtgui/core"
	"rtgui/skin"
)

type DrawCall struct {
	Kind     core.WidgetKind
	Part     skin.SkinPart
	State    core.WidgetState
	Bounds   core.Rect
	Src      core.Rect
	Dest     core.Rect
	Tint     core.Color
	Alpha    float32
	Fallback bool
}

type DebugInfo struct {
	Bounds       core.Rect
	PatchBorders NinePatchConfig
	SkinKey      skin.SkinKey
	Fallback     bool
}
