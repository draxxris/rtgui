// Package skin defines render skin data and registries.
package skin

import "github.com/draxxris/rtgui/core"

// Texture describes a raylib texture without importing raylib into skin.
type Texture struct {
	// ID is the renderer texture handle; zero means no texture.
	ID uint32
	// Width is the texture width in pixels.
	Width int32
	// Height is the texture height in pixels.
	Height int32
	// Mipmaps is the number of uploaded mip levels.
	Mipmaps int32
	// Format is the renderer pixel format value.
	Format int32
}

// NinePatch stores the border thicknesses for a textured nine-patch.
type NinePatch struct {
	// Left and Top are the source and destination border thicknesses.
	Left, Top int32
	// Right and Bottom are the source and destination border thicknesses.
	Right, Bottom int32
}

// SkinPart identifies one drawable component of a widget skin.
type SkinPart int32

const (
	// PartBackground identifies a widget's main fill.
	PartBackground SkinPart = iota
	// PartBorder identifies a widget border.
	PartBorder
	// PartIcon identifies a general widget icon.
	PartIcon
	// PartTrack identifies a slider or progress track.
	PartTrack
	// PartThumb identifies a slider thumb.
	PartThumb
	// PartArrow identifies a dropdown arrow.
	PartArrow
	// PartCheckmark identifies a checked checkbox icon.
	PartCheckmark
	// PartOverlay identifies a progress-bar fill (ProgressBar::fill) or a
	// popup row highlight (Dropdown::highlight, Menu::highlight). Keys are
	// scoped by widget kind, so the roles never share a registry entry.
	PartOverlay
	// PartText identifies text-oriented draw records.
	PartText
	// PartSpark identifies a progress-bar edge marker at the fill boundary.
	PartSpark
	// PartPopup identifies a popup background (Dropdown::popup, Menu::popup).
	PartPopup
	// PartPopupBorder identifies a popup border (Dropdown::popup, Menu::popup).
	PartPopupBorder
	// PartTab identifies one tab button inside a tab bar.
	PartTab
	// PartCaret identifies a textbox caret line. It is geometry-only and
	// never authored through CSS; unauthored textboxes still show a caret.
	PartCaret
)

// SkinKey selects a descriptor by widget kind, part, and visual state.
type SkinKey struct {
	// Widget selects the widget kind.
	Widget core.WidgetKind
	// Part selects the visual component.
	Part SkinPart
	// State selects the visual state.
	State core.WidgetState
}

// SkinDescriptor contains the texture and geometry for one skin key. Tint is
// exact RGBA data; callers that want an untinted texture must set it to opaque
// white explicitly.
type SkinDescriptor struct {
	// Texture is the borrowed or CSS-owned texture handle.
	Texture Texture
	// AtlasRegion selects the source rectangle within Texture.
	AtlasRegion core.Rect
	// NinePatch contains border thicknesses when HasNinePatch is true.
	NinePatch NinePatch
	// Tint is exact RGBA draw data; opaque white means no tint.
	Tint core.Color
	// Padding values add content insets in logical pixels.
	PaddingLeft, PaddingTop, PaddingRight, PaddingBottom float32
	// HasTexture reports whether Texture should be drawn.
	HasTexture bool
	// HasNinePatch reports whether NinePatch geometry should be used.
	HasNinePatch bool
	// CenterFill controls whether the middle nine-patch tile is drawn.
	CenterFill bool
}
