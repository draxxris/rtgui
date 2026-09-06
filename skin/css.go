// Package skin maps widget styling to draw descriptors. This file adds the
// headless half of CSS file skinning: it parses a LOOK-only CSS subset into
// SkinRules without touching GL, so everything here is unit-testable without
// a display. Texture upload and registry writes live in render/css.go.
//
// Supported grammar: Kind[::part][:pseudo] selectors with a closed property
// set (border-image-source, border-image-slice, background-image,
// background-image-tint, border-image-source-tint, padding). Anything else is
// a hard error, never a silent skip.
package skin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/draxxris/rtgui/core"
	"github.com/vanng822/css"
)

// SkinRule is one parsed CSS rule block broken down per touched part: at most
// a background rule and a border rule per block, in source order. Later rules
// for the same key win at apply time; states inherit missing siblings from
// the same kind+part normal rule at merge time (see render).
type SkinRule struct {
	// Kind is the widget kind selected by the CSS selector.
	Kind core.WidgetKind
	// Part is the visual component selected by the CSS selector.
	Part SkinPart
	// State is the pseudo-class state selected by the CSS selector.
	State core.WidgetState

	// Image is the authored url() path, unresolved; empty when absent.
	Image string
	// HasImage reports whether Image was declared.
	HasImage bool

	// Slice is border-image-slice in pixels when HasSlice is true.
	Slice int32
	// HasSlice reports whether Slice was declared.
	HasSlice bool

	// Tint is the authored *-tint color when HasTint is true.
	Tint core.Color
	// HasTint reports whether Tint was declared.
	HasTint bool

	// Padding contains top, right, bottom, and left values when declared.
	Padding [4]float32
	// HasPadding reports whether Padding was declared.
	HasPadding bool
}

// kindSelectors maps CSS kind spellings to widget kinds. Exact case.
var kindSelectors = map[string]core.WidgetKind{
	"Button":      core.WidgetButton,
	"Checkbox":    core.WidgetCheckbox,
	"Textbox":     core.WidgetTextbox,
	"Dropdown":    core.WidgetDropdown,
	"Slider":      core.WidgetSlider,
	"ProgressBar": core.WidgetProgressBar,
	"Frame":       core.WidgetFrame,
	"Label":       core.WidgetLabel,
	"ScrollPanel": core.WidgetScrollPanel,
	"TabBar":      core.WidgetTabBar,
	"Menu":        core.WidgetMenu,
	"Tooltip":     core.WidgetTooltip,
}

// pseudoSelectors maps pseudo-classes to widget states. "" is normal.
var pseudoSelectors = map[string]core.WidgetState{
	"":         core.StateNormal,
	"hover":    core.StateHovered,
	"active":   core.StatePressed,
	"focus":    core.StateFocused,
	"disabled": core.StateDisabled,
	"selected": core.StateSelected,
}

// partSelectors maps ::part names to skin parts.
var partSelectors = map[string]SkinPart{
	"track":     PartTrack,
	"thumb":     PartThumb,
	"arrow":     PartArrow,
	"checkmark": PartCheckmark,
	"box":       PartIcon,
	"popup":     PartPopup,
	"highlight": PartOverlay,
	"fill":      PartOverlay,
	"spark":     PartSpark,
	"tab":       PartTab,
}

// partAllowlist restricts which parts each kind accepts. Pairings outside it
// are hard errors; plain kind rules keep their whole-widget meaning.
var partAllowlist = map[core.WidgetKind]map[string]bool{
	core.WidgetSlider:      {"track": true, "thumb": true},
	core.WidgetDropdown:    {"arrow": true, "popup": true, "highlight": true},
	core.WidgetCheckbox:    {"checkmark": true, "box": true},
	core.WidgetProgressBar: {"track": true, "fill": true, "spark": true},
	core.WidgetTabBar:      {"tab": true},
	core.WidgetMenu:        {"popup": true, "highlight": true},
}

// ParseCSS parses LOOK-only CSS text into SkinRules in source order.
// Unknown kinds, parts, pseudo-classes, properties, and malformed values fail
// loudly with the offending selector or declaration named.
func ParseCSS(text string) ([]SkinRule, error) {
	sheet := css.Parse(text)
	var rules []SkinRule
	for _, rule := range sheet.GetCSSRuleList() {
		selector := strings.TrimSpace(rule.Style.Selector.Text())
		kind, partName, part, state, err := parseSelector(selector)
		if err != nil {
			return nil, err
		}
		block, err := parseBlock(selector, kind, part, partName != "", state, rule.Style.Styles)
		if err != nil {
			return nil, err
		}
		rules = append(rules, block...)
	}
	return rules, nil
}

// parseSelector splits Kind[::part][:pseudo] against the frozen vocabularies.
// It returns the kind, the ::part name ("" when whole-widget), the resolved
// part for declaration routing, and the state.
func parseSelector(selector string) (core.WidgetKind, string, SkinPart, core.WidgetState, error) {
	rest := selector
	kindName := rest
	partName := ""
	if idx := strings.Index(rest, "::"); idx >= 0 {
		kindName, rest = rest[:idx], rest[idx+2:]
		partName = rest
		if idx := strings.Index(rest, ":"); idx >= 0 {
			partName, rest = rest[:idx], rest[idx+1:]
		} else {
			rest = ""
		}
	} else if idx := strings.Index(rest, ":"); idx >= 0 {
		kindName, rest = rest[:idx], rest[idx+1:]
	} else {
		rest = ""
	}
	kind, ok := kindSelectors[kindName]
	if !ok {
		return 0, "", 0, 0, fmt.Errorf("skin: unknown kind selector %q", selector)
	}
	var part SkinPart
	if partName != "" {
		part, ok = partSelectors[partName]
		if !ok {
			return 0, "", 0, 0, fmt.Errorf("skin: unknown part %q in selector %q", partName, selector)
		}
		allowed, ok := partAllowlist[kind]
		if !ok || !allowed[partName] {
			return 0, "", 0, 0, fmt.Errorf("skin: part %q not allowed on %q", partName, kindName)
		}
	}
	state, ok := pseudoSelectors[rest]
	if !ok {
		return 0, "", 0, 0, fmt.Errorf("skin: unknown pseudo-class %q in selector %q", rest, selector)
	}
	return kind, partName, part, state, nil
}

// parseBlock converts one rule block's declarations into per-part entries.
// Whole-widget rules split into PartBackground and PartBorder entries.
// Popup parts (Dropdown::popup, Menu::popup) are the exceptions that accept
// both looks and fan out to PartPopup and PartPopupBorder entries; other parts
// take only background-image declarations, and only whole widgets and popup
// parts take padding.
func parseBlock(selector string, kind core.WidgetKind, part SkinPart, hasPart bool, state core.WidgetState, styles []*css.CSSStyleDeclaration) ([]SkinRule, error) {
	byPart := map[SkinPart]*SkinRule{}
	order := []SkinPart{}
	take := func(part SkinPart) *SkinRule {
		if entry, ok := byPart[part]; ok {
			return entry
		}
		entry := &SkinRule{Kind: kind, Part: part, State: state}
		byPart[part] = entry
		order = append(order, part)
		return entry
	}
	imageTarget, borderTarget, paddingTarget := PartBackground, PartBorder, PartBackground
	allowBorder, allowPadding := true, true
	if hasPart {
		if part == PartPopup {
			imageTarget, borderTarget, paddingTarget = PartPopup, PartPopupBorder, PartPopup
		} else {
			imageTarget = part
			allowBorder, allowPadding = false, false
		}
	}
	for _, style := range styles {
		property := strings.TrimSpace(style.Property)
		value := strings.TrimSpace(style.Value.Text())
		switch property {
		case "background-image", "background-image-tint":
			if err := applyBackgroundProp(take(imageTarget), selector, property, value); err != nil {
				return nil, err
			}
		case "border-image-source", "border-image-source-tint", "border-image-slice":
			if !allowBorder {
				return nil, fmt.Errorf("skin: %s in %q applies to widgets and ::popup parts, not other parts", property, selector)
			}
			if err := applyBorderProp(take(borderTarget), selector, property, value); err != nil {
				return nil, err
			}
		case "padding":
			if !allowPadding {
				return nil, fmt.Errorf("skin: padding in %q applies to widgets and ::popup parts, not other parts", selector)
			}
			padding, err := expandPadding(selector, value)
			if err != nil {
				return nil, err
			}
			entry := take(paddingTarget)
			entry.Padding, entry.HasPadding = padding, true
		default:
			return nil, fmt.Errorf("skin: unsupported property %q in %q (LOOK-only subset)", property, selector)
		}
	}
	entries := make([]SkinRule, 0, len(order))
	for _, part := range order {
		entries = append(entries, *byPart[part])
	}
	return entries, nil
}

// applyBackgroundProp stores a background-image declaration on the entry.
func applyBackgroundProp(entry *SkinRule, selector, property, value string) error {
	switch property {
	case "background-image":
		path, err := extractURL(selector, property, value)
		if err != nil {
			return err
		}
		entry.Image, entry.HasImage = path, true
		return nil
	default:
		tint, err := parseTint(selector, property, value)
		if err != nil {
			return err
		}
		entry.Tint, entry.HasTint = tint, true
		return nil
	}
}

// applyBorderProp stores a border-image declaration on the entry.
func applyBorderProp(entry *SkinRule, selector, property, value string) error {
	switch property {
	case "border-image-source":
		path, err := extractURL(selector, property, value)
		if err != nil {
			return err
		}
		entry.Image, entry.HasImage = path, true
		return nil
	case "border-image-source-tint":
		tint, err := parseTint(selector, property, value)
		if err != nil {
			return err
		}
		entry.Tint, entry.HasTint = tint, true
		return nil
	default:
		slice, err := parsePixels(selector, property, value, true)
		if err != nil {
			return err
		}
		entry.Slice, entry.HasSlice = slice, true
		return nil
	}
}

// extractURL unwraps url("..."), url('...'), or url(...) and rejects empties.
func extractURL(selector, property, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 5 || !strings.HasPrefix(strings.ToLower(trimmed), "url(") || !strings.HasSuffix(trimmed, ")") {
		return "", fmt.Errorf("skin: %s in %q must be url(...), got %q", property, selector, value)
	}
	inner := strings.TrimSpace(trimmed[4 : len(trimmed)-1])
	inner = strings.Trim(inner, "\"'")
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return "", fmt.Errorf("skin: %s in %q has an empty url", property, selector)
	}
	return inner, nil
}

// parsePixels parses a bare number or px-suffixed integer. whole demands an
// integer value (border slices); otherwise decimals are allowed.
func parsePixels(selector, property, value string, whole bool) (int32, error) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimSuffix(trimmed, "px")
	trimmed = strings.TrimSpace(trimmed)
	if whole {
		number, err := strconv.Atoi(trimmed)
		if err != nil || number < 0 {
			return 0, fmt.Errorf("skin: %s in %q must be a non-negative integer, got %q", property, selector, value)
		}
		return int32(number), nil
	}
	number, err := strconv.ParseFloat(trimmed, 32)
	if err != nil {
		return 0, fmt.Errorf("skin: %s in %q must be numeric, got %q", property, selector, value)
	}
	return int32(number), nil
}

// expandPadding expands CSS 1–4 value padding into top, right, bottom, left.
func expandPadding(selector, value string) ([4]float32, error) {
	fields := strings.Fields(value)
	numbers := make([]float32, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSuffix(field, "px")
		number, err := strconv.ParseFloat(trimmed, 32)
		if err != nil {
			return [4]float32{}, fmt.Errorf("skin: padding in %q must be numeric, got %q", selector, value)
		}
		numbers = append(numbers, float32(number))
	}
	switch len(numbers) {
	case 1:
		return [4]float32{numbers[0], numbers[0], numbers[0], numbers[0]}, nil
	case 2:
		return [4]float32{numbers[0], numbers[1], numbers[0], numbers[1]}, nil
	case 3:
		return [4]float32{numbers[0], numbers[1], numbers[2], numbers[1]}, nil
	case 4:
		return [4]float32{numbers[0], numbers[1], numbers[2], numbers[3]}, nil
	default:
		return [4]float32{}, fmt.Errorf("skin: padding in %q needs 1–4 values, got %q", selector, value)
	}
}

// parseTint parses #rrggbb or #rrggbbaa hex into a color. The leading # is
// required and digits are case-insensitive.
func parseTint(selector, property, value string) (core.Color, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 2 || trimmed[0] != '#' {
		return core.Color{}, fmt.Errorf("skin: %s in %q must be #rrggbb or #rrggbbaa, got %q", property, selector, value)
	}
	digits := trimmed[1:]
	if len(digits) != 6 && len(digits) != 8 {
		return core.Color{}, fmt.Errorf("skin: %s in %q must be #rrggbb or #rrggbbaa, got %q", property, selector, value)
	}
	bytes := make([]uint8, len(digits)/2)
	for i := range bytes {
		number, err := strconv.ParseUint(digits[i*2:i*2+2], 16, 8)
		if err != nil {
			return core.Color{}, fmt.Errorf("skin: %s in %q must be #rrggbb or #rrggbbaa, got %q", property, selector, value)
		}
		bytes[i] = uint8(number)
	}
	tint := core.Color{R: bytes[0], G: bytes[1], B: bytes[2], A: 255}
	if len(bytes) == 4 {
		tint.A = bytes[3]
	}
	return tint, nil
}
