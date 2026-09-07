package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// ScrollContent returns the skin-aware content area available inside a scroll panel's
// background and border skins. It resolves ScrollPanel's background and border descriptors
// for state (with normal fallback) and snaps when pixel snap is on. It is safe on a nil theme,
// where it returns bounds.
func (t *Theme) ScrollContent(bounds core.Rect, state core.WidgetState) core.Rect {
	var background, border skin.SkinDescriptor
	if t != nil {
		background, _ = t.resolveDescriptor(core.WidgetScrollPanel, skin.PartBackground, state)
		border, _ = t.resolveDescriptor(core.WidgetScrollPanel, skin.PartBorder, state)
	}
	return t.snap(ContentRect(bounds, background, border))
}
