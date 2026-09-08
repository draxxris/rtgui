package render

import (
	"image/color"

	"github.com/draxxris/rtgui/core"
)

// RegisterInlineIcon whitelists one inline icon name for rich text.
// The icon is borrowed metadata; a zero ID still reserves layout space and
// logs a placeholder. Names are matched exactly; empty names are ignored.
func (t *Theme) RegisterInlineIcon(name string, icon core.TooltipIcon) {
	if t == nil || name == "" {
		return
	}
	if t.inlineIcons == nil {
		t.inlineIcons = make(map[string]core.TooltipIcon)
	}
	t.inlineIcons[name] = icon
}

// UnregisterInlineIcon removes one whitelisted inline icon.
func (t *Theme) UnregisterInlineIcon(name string) {
	if t == nil || t.inlineIcons == nil {
		return
	}
	delete(t.inlineIcons, name)
}

// LookupInlineIcon resolves a whitelisted inline icon by exact name.
func (t *Theme) LookupInlineIcon(name string) (core.TooltipIcon, bool) {
	if t == nil || t.inlineIcons == nil {
		return core.TooltipIcon{}, false
	}
	icon, ok := t.inlineIcons[name]
	return icon, ok
}

// SetLinkColor registers the draw-time tint for one link kind.
// Explicit segment colors still win; unset kinds fall back to the fixed
// link blue. LinkNone is ignored.
func (t *Theme) SetLinkColor(kind core.LinkKind, color core.Color) {
	if t == nil || kind == core.LinkNone {
		return
	}
	if t.linkColors == nil {
		t.linkColors = make(map[core.LinkKind]core.Color)
	}
	t.linkColors[kind] = color
}

// ClearLinkColor removes the registered tint for one link kind.
func (t *Theme) ClearLinkColor(kind core.LinkKind) {
	if t == nil || t.linkColors == nil {
		return
	}
	delete(t.linkColors, kind)
}

// LinkColor resolves the registered tint for one link kind.
func (t *Theme) LinkColor(kind core.LinkKind) (core.Color, bool) {
	if t == nil || t.linkColors == nil {
		return core.Color{}, false
	}
	color, ok := t.linkColors[kind]
	return color, ok
}

// richLinkTint resolves body, explicit, registered, and default link tints.
// Explicit segment colors win, then the parent-registered per-kind color,
// then the fixed fallback blue.
func (t *Theme) richLinkTint(hasColor bool, tint core.Color, kind core.LinkKind, fallback color.RGBA) color.RGBA {
	if hasColor {
		return tint.RGBA()
	}
	if kind != core.LinkNone {
		if registered, ok := t.LinkColor(kind); ok {
			return registered.RGBA()
		}
		return fallback
	}
	return color.RGBA{}
}
