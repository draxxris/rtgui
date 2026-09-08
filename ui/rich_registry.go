package ui

import (
	"github.com/draxxris/rtgui/core"
)

// Rich registries (inline icons, link tints, named rich faces) go through
// UI; general font loading, CSS skins, and viewport setup stay on Theme().

// RegisterInlineIcon whitelists one inline icon name for rich text.
// Gallery registers iron-plate here to prove parent-owned icon data.
func (u *UI) RegisterInlineIcon(name string, icon core.TooltipIcon) {
	if u == nil || u.theme == nil {
		return
	}
	u.theme.RegisterInlineIcon(name, icon)
}

// UnregisterInlineIcon removes one whitelisted inline icon.
func (u *UI) UnregisterInlineIcon(name string) {
	if u == nil || u.theme == nil {
		return
	}
	u.theme.UnregisterInlineIcon(name)
}

// LookupInlineIcon resolves a whitelisted inline icon by exact name.
func (u *UI) LookupInlineIcon(name string) (core.TooltipIcon, bool) {
	if u == nil || u.theme == nil {
		return core.TooltipIcon{}, false
	}
	return u.theme.LookupInlineIcon(name)
}

// SetLinkColor registers the draw-time tint for one link kind.
// Gallery assigns distinct colors to URL, item, and player links.
func (u *UI) SetLinkColor(kind core.LinkKind, color core.Color) {
	if u == nil || u.theme == nil {
		return
	}
	u.theme.SetLinkColor(kind, color)
}

// ClearLinkColor removes the registered tint for one link kind.
func (u *UI) ClearLinkColor(kind core.LinkKind) {
	if u == nil || u.theme == nil {
		return
	}
	u.theme.ClearLinkColor(kind)
}

// LinkColor resolves the registered tint for one link kind.
func (u *UI) LinkColor(kind core.LinkKind) (core.Color, bool) {
	if u == nil || u.theme == nil {
		return core.Color{}, false
	}
	return u.theme.LinkColor(kind)
}

// RegisterFont records a named TTF/OTF face for rich segments.
// Gallery registers ValleySans here to prove parent-owned face data.
func (u *UI) RegisterFont(name, path string) error {
	if u == nil || u.theme == nil {
		return nil
	}
	return u.theme.RegisterFont(name, path)
}
