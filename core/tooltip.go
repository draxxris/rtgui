// Package core contains the data types shared by the rtgui packages.
//
// The package deliberately has no rendering or window-system dependency.
package core

// TooltipIcon carries renderer-free image metadata for a rich tooltip.
//
// Core cannot import skin without creating a cycle, so the icon is plain
// data: a texture handle plus an atlas source rectangle. Render resolves
// the handle; a zero ID means no uploaded texture.
type TooltipIcon struct {
	// ID is the renderer texture handle; zero means no texture.
	ID uint32
	// Width and Height are the texture dimensions in pixels.
	Width, Height int32
	// Source selects the atlas rectangle within the texture.
	Source Rect
	// Tint is the exact RGBA draw tint.
	Tint Color
	// HasTint selects Tint over opaque white.
	HasTint bool
}

// SourceRect returns the atlas rectangle used for drawing. An empty
// source selects the full texture dimensions; a zero texture with an
// explicit source still returns that source.
func (icon TooltipIcon) SourceRect() Rect {
	if icon.Source.W > 0 && icon.Source.H > 0 {
		return icon.Source
	}
	if icon.Width > 0 && icon.Height > 0 {
		return Rect{W: float32(icon.Width), H: float32(icon.Height)}
	}
	return icon.Source
}

// RichTooltip is the renderer-facing data model for a rich hover popup.
//
// Title is the item name, Subtitle is optional flavor text, and Segments
// is the colored multiline body. Width caps the content width; non-positive
// selects the renderer default. The icon is borrowed metadata; HasIcon
// selects Icon. The UI layer defensive-copies data before handing it to
// render, so render may retain strings without copying.
type RichTooltip struct {
	// Title is the item name drawn first.
	Title string
	// TitleColor is the item-rarity color when HasTitleColor is true.
	TitleColor    Color
	HasTitleColor bool
	// Subtitle is optional secondary text drawn under the title.
	Subtitle string
	// Segments is the colored multiline body.
	Segments []RichSegment
	// Width caps the content width; non-positive selects the default.
	Width float32
	// Icon is borrowed image metadata used only when HasIcon is true.
	Icon TooltipIcon
	// HasIcon selects Icon.
	HasIcon bool
}

// HasTitle reports whether a title row should be laid out.
func (d RichTooltip) HasTitle() bool { return d.Title != "" }

// HasBody reports whether any body segment carries visible text or an icon.
func (d RichTooltip) HasBody() bool {
	for _, segment := range d.Segments {
		if segment.Text != "" || segment.HasIcon {
			return true
		}
	}
	return false
}

// HasContent reports whether the tooltip draws anything at all.
func (d RichTooltip) HasContent() bool {
	return d.Title != "" || d.Subtitle != "" || d.HasBody() || d.HasIcon
}

// DesiredWidth returns the content-width cap, falling back when empty.
func (d RichTooltip) DesiredWidth(fallback float32) float32 {
	if d.Width > 0 {
		return d.Width
	}
	return fallback
}

// RichTooltipDataEqual reports whether two tooltip payloads match exactly,
// including icon metadata and every body segment field.
func RichTooltipDataEqual(a, b RichTooltip) bool {
	if a.Title != b.Title || a.Subtitle != b.Subtitle {
		return false
	}
	if a.HasTitleColor != b.HasTitleColor || a.HasTitleColor && a.TitleColor != b.TitleColor {
		return false
	}
	if a.Width != b.Width || a.HasIcon != b.HasIcon {
		return false
	}
	if a.HasIcon && !tooltipIconEqual(a.Icon, b.Icon) {
		return false
	}
	return richSegmentsEqual(a.Segments, b.Segments)
}

// tooltipIconEqual compares borrowed icon metadata field by field.
// Tint is only compared when selected so unused defaults never mismatch.
func tooltipIconEqual(a, b TooltipIcon) bool {
	if a.ID != b.ID || a.Width != b.Width || a.Height != b.Height {
		return false
	}
	if a.Source != b.Source || a.HasTint != b.HasTint {
		return false
	}
	return !a.HasTint || a.Tint == b.Tint
}

// EqualRichSegments reports whether two segment slices match under the
// canonical unset-tolerant semantics: optional colors, sizes, faces, and
// icons compare only when selected, so unused defaults never mismatch.
// Widgets and render caches share this definition so change detection
// agrees across packages.
func EqualRichSegments(a, b []RichSegment) bool {
	return richSegmentsEqual(a, b)
}

// richSegmentsEqual compares body runs including style, color, icon, and link payloads.
// Color is only compared when selected so unused defaults never mismatch.
func richSegmentsEqual(a, b []RichSegment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !richSegmentEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// richSegmentEqual compares one styled run field by field.
func richSegmentEqual(x, y RichSegment) bool {
	if x.Text != y.Text || x.Bold != y.Bold {
		return false
	}
	return richColorEqual(x, y) && richFontEqual(x, y) && richIconEqual(x, y) && tooltipLinkEqual(x.Link, y.Link)
}

// richColorEqual compares optional run colors.
func richColorEqual(x, y RichSegment) bool {
	if x.HasColor != y.HasColor {
		return false
	}
	return !x.HasColor || x.Color == y.Color
}

// richFontEqual compares optional size and face selections.
func richFontEqual(x, y RichSegment) bool {
	if x.HasFontSize != y.HasFontSize {
		return false
	}
	if x.HasFontSize && x.FontSize != y.FontSize {
		return false
	}
	if x.HasFont != y.HasFont {
		return false
	}
	return !x.HasFont || x.Font == y.Font
}

// richIconEqual compares optional inline icon names.
func richIconEqual(x, y RichSegment) bool {
	if x.HasIcon != y.HasIcon {
		return false
	}
	return !x.HasIcon || x.Icon == y.Icon
}

// tooltipLinkEqual compares one clickable reference field by field.
func tooltipLinkEqual(a, b Link) bool {
	return a.Kind == b.Kind && a.Target == b.Target && a.Tooltip == b.Tooltip
}
