package ui

import (
	"time"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// TooltipAnchorKind selects how a hover tooltip derives its anchor point.
type TooltipAnchorKind int

const (
	// TooltipAnchorCursor follows the pointer with the cursor gap, flipped
	// and clamped into the viewport. It is the zero value and preserves
	// the historical behavior.
	TooltipAnchorCursor TooltipAnchorKind = iota
	// TooltipAnchorFixed pins the popup to Point in logical coordinates.
	// Pointer motion never moves it. The point substitutes for the pointer
	// in the shared gap, flip, and clamp placement.
	TooltipAnchorFixed
)

// TooltipAnchor positions one hover tooltip. The zero value follows the
// pointer. Explicit tooltips from ShowTooltip and ShowRichTooltip carry
// their own point and never consult this configuration.
type TooltipAnchor struct {
	// Kind selects cursor-following versus fixed placement.
	Kind TooltipAnchorKind
	// Point is the logical anchor; Fixed only, ignored for Cursor.
	Point core.Vec2
}

// TooltipOptions overrides global tooltip behavior for one registered widget.
// Unset fields inherit the global default, following the HasTitleColor and
// HasIcon pattern. An option value with neither field set clears the entry.
type TooltipOptions struct {
	// Delay is the hover dwell before this widget's tooltip may draw.
	// Non-positive means immediate. Only Delay with HasDelay set applies.
	Delay time.Duration
	// HasDelay selects Delay over the global dwell.
	HasDelay bool
	// Anchor positions this widget's hover tooltip. Only Anchor with
	// HasAnchor set applies.
	Anchor TooltipAnchor
	// HasAnchor selects Anchor over the global anchor.
	HasAnchor bool
}

// SetTooltipDelay replaces the global hover dwell before hover-derived
// tooltips may draw. Non-positive means immediate, preserving the
// historical behavior. Explicit tooltips bypass the dwell. Negative values
// clamp to zero.
func (u *UI) SetTooltipDelay(d time.Duration) {
	if u == nil {
		return
	}
	if d < 0 {
		d = 0
	}
	u.tooltipDelay = d
}

// TooltipDelay returns the global hover dwell. Non-positive means immediate.
func (u *UI) TooltipDelay() time.Duration {
	if u == nil {
		return 0
	}
	return u.tooltipDelay
}

// SetTooltipAnchor replaces the global hover tooltip anchor. The zero value
// follows the pointer. Explicit tooltips carry their own point and never
// consult this configuration.
func (u *UI) SetTooltipAnchor(anchor TooltipAnchor) {
	if u != nil {
		u.tooltipAnchor = anchor
	}
}

// TooltipAnchor returns the global hover tooltip anchor.
func (u *UI) TooltipAnchor() TooltipAnchor {
	if u == nil {
		return TooltipAnchor{}
	}
	return u.tooltipAnchor
}

// SetTooltipOptions replaces the delay and anchor overrides for a registered
// widget. Unknown names are ignored, matching SetRichTooltip. An option
// value with neither HasDelay nor HasAnchor set clears the entry, so there
// is no separate clear method. Negative delays clamp to zero.
func (u *UI) SetTooltipOptions(name string, opts TooltipOptions) {
	if u == nil || u.Lookup(name) == nil {
		return
	}
	if !opts.HasDelay && !opts.HasAnchor {
		delete(u.tooltipOpts, name)
		return
	}
	if opts.HasDelay && opts.Delay < 0 {
		opts.Delay = 0
	}
	if u.tooltipOpts == nil {
		u.tooltipOpts = make(map[string]TooltipOptions)
	}
	u.tooltipOpts[name] = opts
}

// TooltipOptions returns the delay and anchor overrides for name, or false
// when the widget carries none.
func (u *UI) TooltipOptions(name string) (TooltipOptions, bool) {
	if u == nil {
		return TooltipOptions{}, false
	}
	opts, ok := u.tooltipOpts[name]
	return opts, ok
}

// setTooltipClock replaces the hover-dwell time source for deterministic
// headless tests; production code always uses wall time. A nil clock
// restores time.Now. Changing the clock restarts any pending hover dwell
// under the new source so swapped eras never strand a pending tooltip.
func (u *UI) setTooltipClock(now func() time.Time) {
	if u == nil {
		return
	}
	u.tooltipClock = now
	u.resetHoverDwell()
}

// tooltipNow returns the configured hover-dwell time source.
func (u *UI) tooltipNow() time.Time {
	if u != nil && u.tooltipClock != nil {
		return u.tooltipClock()
	}
	return time.Now()
}

// setHovered records a new hover owner and restarts the hover dwell when the
// owner changes. Pointer identity distinguishes owners, so motion within one
// widget never restarts the dwell. Clearing the owner releases the dwell
// timestamp. All hovered mutations route here so dwell tracking can never
// desynchronize from the hover slot.
func (u *UI) setHovered(widget widgets.Widget) {
	if u == nil || widget == u.hovered {
		return
	}
	u.hovered = widget
	if widget == nil {
		u.hoverSince = time.Time{}
	} else {
		u.hoverSince = u.tooltipNow()
	}
}

// resetHoverDwell restarts the dwell for the current hover owner without
// changing the owner. Dismissal paths call it so a dismissed tooltip does
// not instantly reappear while the pointer stays put.
func (u *UI) resetHoverDwell() {
	if u == nil || u.hovered == nil {
		return
	}
	u.hoverSince = u.tooltipNow()
}

// resolveHoverAnchor maps the pointer through the effective anchor for the
// hovered widget: the per-widget anchor wins when selected, otherwise the
// global anchor. Fixed anchors substitute their point; cursor anchors keep
// following the pointer. The result feeds the shared gap, flip, and clamp
// placement, so edge behavior matches everywhere.
func (u *UI) resolveHoverAnchor() core.Vec2 {
	if u == nil {
		return core.Vec2{}
	}
	anchor := u.tooltipAnchor
	if u.hovered != nil {
		if opts, ok := u.tooltipOpts[u.hovered.Name()]; ok && opts.HasAnchor {
			anchor = opts.Anchor
		}
	}
	if anchor.Kind == TooltipAnchorFixed {
		return anchor.Point
	}
	return u.pointer
}

// hoverDwellElapsed reports whether the hover tooltip for the current hover
// target may draw. The per-widget dwell wins when selected, otherwise the
// global dwell applies. Explicit tooltips bypass the dwell; callers check
// those first. Missing tracking state means immediate, preserving behavior
// for hover owners established before any dwell was configured.
func (u *UI) hoverDwellElapsed() bool {
	if u == nil || u.hovered == nil {
		return true
	}
	delay := u.tooltipDelay
	if opts, ok := u.tooltipOpts[u.hovered.Name()]; ok && opts.HasDelay {
		delay = opts.Delay
	}
	if delay > 0 {
		return !u.hoverSince.IsZero() && u.tooltipNow().Sub(u.hoverSince) >= delay
	}
	return true
}
