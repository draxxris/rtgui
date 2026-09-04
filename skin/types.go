// Package skin defines render skin data and registries.
package skin

import "rtgui/core"

type Texture struct {
	ID      uint32
	Width   int32
	Height  int32
	Mipmaps int32
	Format  int32
}

type NinePatch struct {
	Source        core.Rect
	Left, Top     int32
	Right, Bottom int32
	Layout        int32
}

type SkinPart int32

const (
	PartBackground SkinPart = iota
	PartBorder
	PartIcon
	PartTrack
	PartThumb
	PartArrow
	PartCheckmark
	PartHighlight
	PartOverlay
	PartText
)

type SkinKey struct {
	Widget core.WidgetKind
	Part   SkinPart
	State  core.WidgetState
}

type SkinDescriptor struct {
	Texture                                              Texture
	AtlasRegion                                          core.Rect
	NinePatch                                            NinePatch
	Tint                                                 core.Color
	Alpha                                                float32
	PaddingLeft, PaddingTop, PaddingRight, PaddingBottom float32
	MinWidth, MinHeight                                  float32
	HasTexture                                           bool
	HasNinePatch                                         bool
	TileMode                                             int32
	CenterFill                                           bool
}
