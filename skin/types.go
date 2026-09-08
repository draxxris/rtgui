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

// ThreePatch stores cap thicknesses for a textured vertical 3-patch.
type ThreePatch struct {
	// Top and Bottom are the source and destination cap thicknesses.
	Top, Bottom int32
}

// GradientDirection specifies the orientation of a linear gradient.
type GradientDirection uint8

const (
	// GradientToBottom renders vertically from top to bottom (CSS default).
	GradientToBottom GradientDirection = iota
	// GradientToTop renders vertically from bottom to top.
	GradientToTop
	// GradientToRight renders horizontally from left to right.
	GradientToRight
	// GradientToLeft renders horizontally from right to left.
	GradientToLeft
	// GradientToBottomRight renders diagonally from top-left to bottom-right.
	GradientToBottomRight
	// GradientToBottomLeft renders diagonally from top-right to bottom-left.
	GradientToBottomLeft
	// GradientToTopRight renders diagonally from bottom-left to top-right.
	GradientToTopRight
	// GradientToTopLeft renders diagonally from bottom-right to top-left.
	GradientToTopLeft
)

// ColorStop specifies one color stop in a linear gradient.
type ColorStop struct {
	// Color is the exact RGBA color of the stop.
	Color core.Color
	// Position is the normalized position in [0, 1], or -1 for default.
	Position float32
}

// LinearGradient stores a 2-stop linear gradient specification as a value type.
type LinearGradient struct {
	// Direction selects the orientation of the gradient.
	Direction GradientDirection
	// Stops stores the start and end color stops.
	Stops [2]ColorStop
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

// SkinKey selects a descriptor by widget kind, class, part, and visual state.
type SkinKey struct {
	// Widget selects the widget kind.
	Widget core.WidgetKind
	// Class selects the CSS class variant; empty for base kind rules.
	Class string
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
	// ThreePatch contains cap thicknesses when HasThreePatch is true.
	ThreePatch ThreePatch
	// Tint is exact RGBA draw data; opaque white means no tint.
	Tint core.Color
	// Padding values add content insets in logical pixels.
	PaddingLeft, PaddingTop, PaddingRight, PaddingBottom float32
	// HasPadding reports whether padding values were explicitly declared.
	HasPadding bool
	// HasTexture reports whether Texture should be drawn.
	HasTexture bool
	// HasNinePatch reports whether NinePatch geometry should be used.
	HasNinePatch bool
	// HasThreePatch reports whether ThreePatch geometry should be used.
	HasThreePatch bool
	// CenterFill controls whether the middle nine-patch tile is drawn.
	CenterFill bool
	// BackgroundColor stores the solid fill color when HasBackgroundColor is true.
	BackgroundColor core.Color
	// HasBackgroundColor reports whether BackgroundColor should be drawn.
	HasBackgroundColor bool
	// Radius clips background layers (color, gradient, texture) to a rounded
	// rectangle of this pixel radius when HasRadius is true. Zero draws square.
	Radius float32
	// HasRadius reports whether Radius was explicitly declared.
	HasRadius bool
	// Gradient stores the linear gradient fill when HasGradient is true.
	Gradient LinearGradient
	// HasGradient reports whether Gradient should be drawn.
	HasGradient bool
	// TextColor stores the authored text color when HasTextColor is true.
	TextColor core.Color
	// HasTextColor reports whether TextColor was explicitly declared.
	HasTextColor bool
	// FontSize stores the authored font size in pixels when HasFontSize is true.
	FontSize float32
	// HasFontSize reports whether FontSize was explicitly declared.
	HasFontSize bool
	// Font stores the font face name or font file path when HasFont is true.
	Font string
	// HasFont reports whether Font was explicitly declared.
	HasFont bool
	// ItalicFont stores the accent/italic font path when HasItalicFont is true.
	ItalicFont string
	// HasItalicFont reports whether ItalicFont was explicitly declared.
	HasItalicFont bool
}

// Overlay returns a copy of d with visual properties declared in other applied on top.
func (d SkinDescriptor) Overlay(other SkinDescriptor) SkinDescriptor {
	if other.HasTexture {
		d.Texture = other.Texture
		d.AtlasRegion = other.AtlasRegion
		d.Tint = other.Tint
		d.HasTexture = true
	}
	if other.HasNinePatch {
		d.NinePatch = other.NinePatch
		d.HasNinePatch = true
		d.CenterFill = other.CenterFill
	}
	if other.HasThreePatch {
		d.ThreePatch = other.ThreePatch
		d.HasThreePatch = true
	}
	if other.HasBackgroundColor {
		d.BackgroundColor = other.BackgroundColor
		d.HasBackgroundColor = true
	}
	if other.HasRadius {
		d.Radius = other.Radius
		d.HasRadius = true
	}
	if other.HasGradient {
		d.Gradient = other.Gradient
		d.HasGradient = true
	}
	if other.HasPadding {
		d.PaddingLeft = other.PaddingLeft
		d.PaddingTop = other.PaddingTop
		d.PaddingRight = other.PaddingRight
		d.PaddingBottom = other.PaddingBottom
		d.HasPadding = true
	}
	if other.HasTextColor {
		d.TextColor = other.TextColor
		d.HasTextColor = true
	}
	if other.HasFontSize {
		d.FontSize = other.FontSize
		d.HasFontSize = true
	}
	if other.HasFont {
		d.Font = other.Font
		d.HasFont = true
	}
	if other.HasItalicFont {
		d.ItalicFont = other.ItalicFont
		d.HasItalicFont = true
	}
	return d
}
