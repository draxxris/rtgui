// Package core contains the data types shared by the rtgui packages.
//
// The package deliberately has no rendering or window-system dependency.
package core

import "image/color"

// Status is returned by operations that reject an invalid value or cannot
// resolve a requested resource.
type Status int32

const (
	// StatusInvalidArg reports an invalid input value.
	StatusInvalidArg Status = iota + 1
	// StatusMissingSkin reports that no descriptor exists for a skin lookup.
	StatusMissingSkin
)

// Error returns the stable diagnostic text for a status.
func (s Status) Error() string {
	switch s {
	case StatusInvalidArg:
		return "invalid argument"
	case StatusMissingSkin:
		return "missing skin"
	default:
		return "rtgui error"
	}
}

// Rect is an axis-aligned rectangle in logical UI coordinates.
type Rect struct {
	// X and Y are the rectangle's logical origin; W and H are its size.
	X, Y, W, H float32
}

// Contains reports whether p is inside the rectangle, including its edges.
func (r Rect) Contains(p Vec2) bool {
	return p.X >= r.X && p.X <= r.X+r.W && p.Y >= r.Y && p.Y <= r.Y+r.H
}

// Vec2 is a two-dimensional point or size in logical UI coordinates.
type Vec2 struct {
	// X and Y are the horizontal and vertical components.
	X, Y float32
}

// Color is an exact 8-bit RGBA color. A zero Color is transparent black.
type Color struct {
	// R, G, B, and A are the red, green, blue, and alpha channels.
	R, G, B, A uint8
}

// ToColor converts a standard-library RGBA value without changing channels.
func ToColor(c color.RGBA) Color { return Color{R: c.R, G: c.G, B: c.B, A: c.A} }

// RGBA converts c to the standard-library RGBA representation.
func (c Color) RGBA() color.RGBA { return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A} }

// WidgetKind identifies the behavior and rendering contract of a widget.
type WidgetKind int32

const (
	// WidgetButton is an activatable push button.
	WidgetButton WidgetKind = iota
	// WidgetLabel displays non-interactive text.
	WidgetLabel
	// WidgetCheckbox toggles a boolean value when activated.
	WidgetCheckbox
	// WidgetTextbox accepts bounded UTF-8 text edits.
	WidgetTextbox
	// WidgetScrollPanel owns a scroll offset for application content.
	WidgetScrollPanel
	// WidgetDropdown displays and selects one item.
	WidgetDropdown
	// WidgetSlider edits a value in the inclusive range [0, 1].
	WidgetSlider
	// WidgetProgressBar displays a value in the inclusive range [0, 1].
	WidgetProgressBar
	// WidgetFrame is a non-interactive layout and decoration frame.
	WidgetFrame
	// WidgetTabBar displays and selects one tab.
	WidgetTabBar
	// WidgetMenu is an ephemeral context-menu popup owned by the UI.
	WidgetMenu
	// WidgetTooltip is a non-interactive hover popup owned by the UI.
	WidgetTooltip
	// WidgetRichText displays wrapped multi-line segments with clickable links.
	WidgetRichText
	// WidgetCanvas delegates drawing to a custom application callback.
	WidgetCanvas
	// WidgetLineGraph displays one or more numeric data series.
	WidgetLineGraph
	// WidgetList displays a collapsible category list with single leaf selection.
	WidgetList
	// WidgetChatLog displays a bounded read-only log of wrapped rich-text messages.
	WidgetChatLog
	// WidgetAny represents any widget kind for universal CSS rules.
	WidgetAny WidgetKind = -1
)

// WidgetState is the visual state selected by UI interaction ownership.
type WidgetState int32

const (
	// StateNormal is the default visual state.
	StateNormal WidgetState = iota
	// StateFocused marks the focused widget.
	StateFocused
	// StateHovered marks the topmost widget under the pointer.
	StateHovered
	// StatePressed marks the widget owning the active press.
	StatePressed
	// StateDisabled marks a widget that does not accept interaction.
	StateDisabled
	// StateSelected is available for explicit application or skin state.
	StateSelected
)

// WidgetInfo is the renderer-facing snapshot of a widget.
type WidgetInfo struct {
	// Name is the widget's external registry identity.
	Name string
	// Bounds are the widget's resolved logical bounds.
	Bounds Rect
	// Kind identifies the widget behavior.
	Kind WidgetKind
	// State is the visual state computed by the UI owner.
	State WidgetState
	// Class is the authored CSS class variant for styling; empty when none.
	Class string
	// TextColor is the optional explicit text color for widgets with text.
	TextColor Color
	// HasTextColor reports whether TextColor was explicitly configured.
	HasTextColor bool
	// FontSize is the optional explicit font size in pixels (0 means auto-derive from height).
	FontSize float32
	// Italic reports whether to render text with the theme's italic font.
	Italic bool
	// Align is the horizontal alignment for widget text.
	Align TextAlign
}

// TextAlign defines horizontal text alignment within widget bounds.
type TextAlign int32

const (
	// AlignLeft aligns text to the left edge of the content area.
	AlignLeft TextAlign = iota
	// AlignCenter centers text horizontally in the content area.
	AlignCenter
	// AlignRight aligns text to the right edge of the content area.
	AlignRight
)

// Viewport describes the physical viewport and fixed logical design size.
type Viewport struct {
	// Viewport is the physical window rectangle.
	Viewport Rect
	// LogicalSize is the coordinate space used by layout and input.
	LogicalSize Vec2
}

// MenuItem is one context-menu row shared by ui state and render drawing.
// Core owns the shape so render never imports ui and ui never imports render
// for menu data.
type MenuItem struct {
	// ID is the selection value passed to the menu callback.
	ID string
	// Label is the displayed row text; ignored for separators.
	Label string
	// Disabled marks a visible but unselectable row.
	Disabled bool
	// Separator marks a non-selectable rule row.
	Separator bool
}

// LinkKind identifies the domain meaning of a rich-text link target.
// Targets stay opaque strings; each game system interprets its own kinds.
type LinkKind int32

const (
	// LinkNone marks plain text without a link.
	LinkNone LinkKind = iota
	// LinkURL opens an out-of-game address held in Target.
	LinkURL
	// LinkItem references an in-game inventory item held in Target.
	LinkItem
	// LinkLocation references an in-game position held in Target.
	LinkLocation
	// LinkQuest references an in-game quest held in Target.
	LinkQuest
	// LinkPlayer references an in-game character held in Target.
	LinkPlayer
	// LinkCustom carries an opaque app-defined payload held in Target.
	LinkCustom
)

// String returns the stable display name of a link kind.
func (k LinkKind) String() string {
	switch k {
	case LinkURL:
		return "url"
	case LinkItem:
		return "item"
	case LinkLocation:
		return "location"
	case LinkQuest:
		return "quest"
	case LinkPlayer:
		return "player"
	case LinkCustom:
		return "custom"
	default:
		return "none"
	}
}

// Link is one clickable rich-text reference. A zero Link is plain text.
type Link struct {
	// Kind identifies the domain meaning of Target.
	Kind LinkKind
	// Target is the opaque reference interpreted per Kind.
	Target string
	// Tooltip is static hover text; empty defers to OnLinkTooltipRequested.
	Tooltip string
}

// RichSegment is one styled run inside a rich-text widget. Adjacent
// segments flow without forced breaks; wrapping is word-based.
//
// Go code constructs styling directly; only icon and link markup is
// parsed from player strings. An icon run has HasIcon set and Icon naming
// a UI-registered whitelist entry; Text should be empty for icon runs.
// A link run carries Link; its tint resolves at draw time from the
// parent-registered per-kind color unless HasColor overrides it.
type RichSegment struct {
	// Text is the displayed run; newlines force breaks.
	Text string
	// Color is the run color used only when HasColor is true.
	Color Color
	// HasColor selects Color over theme defaults and registered link colors.
	HasColor bool
	// Link makes the run clickable; a zero Link is plain text.
	Link Link
	// Bold requests a bold face for Go-authored emphasis.
	Bold bool
	// FontSize is the run size used only when HasFontSize is true.
	FontSize float32
	// HasFontSize selects FontSize over the fixed message metrics.
	HasFontSize bool
	// Font names a loaded face used only when HasFont is true.
	Font string
	// HasFont selects Font over the theme default face.
	HasFont bool
	// Icon names a UI-whitelisted inline graphic used only when HasIcon is true.
	Icon string
	// HasIcon marks the run as an inline icon placeholder.
	HasIcon bool
}
