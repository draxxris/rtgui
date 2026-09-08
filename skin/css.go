// Package skin maps widget styling to draw descriptors. This file adds the
// headless half of CSS file skinning: it parses a LOOK-only CSS subset into
// SkinRules without touching GL, so everything here is unit-testable without
// a display. Texture upload and registry writes live in render/css.go.
//
// Supported grammar: Kind[::part][:pseudo] and * selectors with properties
// (border-image-source, border-image-slice, background-image, background-image-tint,
// background-color, border-image-source-tint, border-radius, padding, color, font-size,
// font-family, font-italic-family). Anything else is a hard error.
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
	// Class is the CSS class variant name; empty for base kind rules.
	Class string
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

	// BackgroundColor is the authored background-color when HasBackgroundColor is true.
	BackgroundColor core.Color
	// HasBackgroundColor reports whether BackgroundColor was declared.
	HasBackgroundColor bool

	// Radius is the authored border-radius in pixels when HasRadius is true.
	Radius float32
	// HasRadius reports whether Radius was declared.
	HasRadius bool

	// Gradient is the parsed linear-gradient descriptor when HasGradient is true.
	Gradient LinearGradient
	// HasGradient reports whether Gradient was declared.
	HasGradient bool

	// Font stores the font-family specification (url path or face name).
	Font string
	// HasFont reports whether Font was declared.
	HasFont bool

	// ItalicFont stores the font-italic-family specification (url path or face name).
	ItalicFont string
	// HasItalicFont reports whether ItalicFont was declared.
	HasItalicFont bool

	// FontSize is the authored font size in pixels.
	FontSize float32
	// HasFontSize reports whether FontSize was declared.
	HasFontSize bool

	// TextColor is the authored text color.
	TextColor core.Color
	// HasTextColor reports whether TextColor was declared.
	HasTextColor bool
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
	"RichText":    core.WidgetRichText,
	"LineGraph":   core.WidgetLineGraph,
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
	core.WidgetRichText:    {"highlight": true},
	core.WidgetScrollPanel: {"track": true, "thumb": true},
}

// ParseCSS parses LOOK-only CSS text into SkinRules in source order.
// Unknown kinds, parts, pseudo-classes, properties, and malformed values fail
// loudly with the offending selector or declaration named.
func ParseCSS(text string) ([]SkinRule, error) {
	sheet := css.Parse(text)
	var rules []SkinRule
	for _, rule := range sheet.GetCSSRuleList() {
		selector := strings.TrimSpace(rule.Style.Selector.Text())
		kind, className, partName, part, state, err := parseSelector(selector)
		if err != nil {
			return nil, err
		}
		block, err := parseBlock(selector, kind, className, part, partName != "", state, rule.Style.Styles)
		if err != nil {
			return nil, err
		}
		rules = append(rules, block...)
	}
	return rules, nil
}

// isValidClassName reports whether name conforms to standard CSS class identifier rules.
func isValidClassName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '-' {
			continue
		}
		if i > 0 && (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

// parseKindAndClass extracts optional widget kind and optional class name from kindName.
func parseKindAndClass(kindName, selector string) (core.WidgetKind, string, error) {
	if dot := strings.Index(kindName, "."); dot >= 0 {
		kName := kindName[:dot]
		className := kindName[dot+1:]
		if className == "" {
			return 0, "", fmt.Errorf("skin: empty class in selector %q", selector)
		}
		if !isValidClassName(className) {
			return 0, "", fmt.Errorf("skin: invalid class %q in selector %q", className, selector)
		}
		if kName == "" || kName == "*" {
			return core.WidgetAny, className, nil
		}
		k, ok := kindSelectors[kName]
		if !ok {
			return 0, "", fmt.Errorf("skin: unknown kind selector %q in %q", kName, selector)
		}
		return k, className, nil
	}
	if kindName == "*" {
		return core.WidgetAny, "", nil
	}
	k, ok := kindSelectors[kindName]
	if !ok {
		return 0, "", fmt.Errorf("skin: unknown kind selector %q", selector)
	}
	return k, "", nil
}

// parseSelector splits [Kind][.class][::part][:pseudo] against the frozen vocabularies.
// It returns the kind, the class name ("" when unclassed), the ::part name ("" when whole-widget),
// the resolved part for declaration routing, and the state.
func parseSelector(selector string) (core.WidgetKind, string, string, SkinPart, core.WidgetState, error) {
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
	kind, className, err := parseKindAndClass(kindName, selector)
	if err != nil {
		return 0, "", "", 0, 0, err
	}
	var part SkinPart
	if partName != "" {
		p, ok := partSelectors[partName]
		if !ok {
			return 0, "", "", 0, 0, fmt.Errorf("skin: unknown part %q in selector %q", partName, selector)
		}
		part = p
		if kind == core.WidgetAny {
			return 0, "", "", 0, 0, fmt.Errorf("skin: part %q requires a kind selector in %q", partName, selector)
		}
		allowed, ok := partAllowlist[kind]
		if !ok || !allowed[partName] {
			return 0, "", "", 0, 0, fmt.Errorf("skin: part %q not allowed on %q", partName, kindName)
		}
	}
	state, ok := pseudoSelectors[rest]
	if !ok {
		return 0, "", "", 0, 0, fmt.Errorf("skin: unknown pseudo-class %q in selector %q", rest, selector)
	}
	return kind, className, partName, part, state, nil
}

// resolveRuleTargets determines image, border, and padding target parts and permissions.
func resolveRuleTargets(kind core.WidgetKind, part SkinPart, hasPart bool) (imageTarget, borderTarget, paddingTarget SkinPart, allowBorder, allowPadding bool) {
	if !hasPart {
		return PartBackground, PartBorder, PartBackground, true, true
	}
	if part == PartPopup {
		return PartPopup, PartPopupBorder, PartPopup, true, true
	}
	if kind == core.WidgetScrollPanel && (part == PartTrack || part == PartThumb) {
		return part, part, PartBackground, true, false
	}
	return part, PartBorder, PartBackground, false, false
}

// parseBlock converts one rule block's declarations into per-part entries.
// Whole-widget rules split into PartBackground and PartBorder entries.
// border-radius rides the background entry: it clips background layers to the
// border's rounded shape and never reshapes the border texture itself.
// Popup parts (Dropdown::popup, Menu::popup) are the exceptions that accept
// both looks and fan out to PartPopup and PartPopupBorder entries; other parts
// take only background-image declarations, and only whole widgets and popup
// parts take padding.
func parseBlock(selector string, kind core.WidgetKind, className string, part SkinPart, hasPart bool, state core.WidgetState, styles []*css.CSSStyleDeclaration) ([]SkinRule, error) {
	byPart := map[SkinPart]*SkinRule{}
	order := []SkinPart{}
	take := func(part SkinPart) *SkinRule {
		if entry, ok := byPart[part]; ok {
			return entry
		}
		entry := &SkinRule{Kind: kind, Class: className, Part: part, State: state}
		byPart[part] = entry
		order = append(order, part)
		return entry
	}
	imageTarget, borderTarget, paddingTarget, allowBorder, allowPadding := resolveRuleTargets(kind, part, hasPart)
	for _, style := range styles {
		property := strings.TrimSpace(style.Property)
		value := strings.TrimSpace(style.Value.Text())
		switch property {
		case "background-image", "background-image-tint", "background-color", "border-radius":
			if err := applyBackgroundProp(take(imageTarget), selector, property, value); err != nil {
				return nil, err
			}
		case "border-image-source", "border-image-source-tint", "border-image-slice":
			if !allowBorder {
				return nil, fmt.Errorf("skin: %s in %q applies to widgets, ::popup, ::track, and ::thumb parts, not other parts", property, selector)
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
		case "color", "font-size", "font-family", "font-italic-family":
			if err := applyTextProp(take(imageTarget), selector, property, value); err != nil {
				return nil, err
			}
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

// applyBackgroundProp stores background-image, background-image-tint,
// background-color, or border-radius on entry.
func applyBackgroundProp(entry *SkinRule, selector, property, value string) error {
	switch property {
	case "background-image":
		trimmed := strings.TrimSpace(value)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "url(") {
			path, err := extractURL(selector, property, value)
			if err != nil {
				return err
			}
			entry.Image, entry.HasImage = path, true
			return nil
		}
		if strings.HasPrefix(lower, "linear-gradient(") {
			grad, err := parseLinearGradient(selector, property, value)
			if err != nil {
				return err
			}
			entry.Gradient, entry.HasGradient = grad, true
			return nil
		}
		if lower == "none" {
			entry.Image, entry.HasImage = "", false
			entry.Gradient, entry.HasGradient = LinearGradient{}, false
			return nil
		}
		return fmt.Errorf("skin: %s in %q must be url(...) or linear-gradient(...), got %q", property, selector, value)
	case "border-radius":
		return applyRadiusProp(entry, selector, property, value)
	case "background-color":
		col, err := parseTint(selector, property, value)
		if err != nil {
			return err
		}
		entry.BackgroundColor, entry.HasBackgroundColor = col, true
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

// applyRadiusProp stores a border-radius declaration on the entry. Zero
// clears inherited rounding; negatives fail like other pixel values.
func applyRadiusProp(entry *SkinRule, selector, property, value string) error {
	radius, err := parsePixels(selector, property, value, false)
	if err != nil {
		return err
	}
	if radius < 0 {
		return fmt.Errorf("skin: %s in %q must be non-negative, got %q", property, selector, value)
	}
	entry.Radius, entry.HasRadius = float32(radius), true
	return nil
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

// applyTextProp stores color, font-size, font-family, or font-italic-family on entry.
func applyTextProp(entry *SkinRule, selector, property, value string) error {
	switch property {
	case "color":
		col, err := parseTint(selector, property, value)
		if err != nil {
			return err
		}
		entry.TextColor, entry.HasTextColor = col, true
		return nil
	case "font-size":
		size, err := parsePixels(selector, property, value, false)
		if err != nil {
			return err
		}
		if size <= 0 {
			return fmt.Errorf("skin: font-size in %q must be positive, got %q", selector, value)
		}
		entry.FontSize, entry.HasFontSize = float32(size), true
		return nil
	case "font-family":
		trimmed := strings.TrimSpace(value)
		if strings.HasPrefix(strings.ToLower(trimmed), "url(") {
			urlPath, err := extractURL(selector, property, value)
			if err != nil {
				return err
			}
			entry.Font, entry.HasFont = urlPath, true
			return nil
		}
		name := strings.Trim(trimmed, "\"'")
		if name == "" {
			return fmt.Errorf("skin: %s in %q cannot be empty", property, selector)
		}
		entry.Font, entry.HasFont = name, true
		return nil
	case "font-italic-family":
		trimmed := strings.TrimSpace(value)
		if strings.HasPrefix(strings.ToLower(trimmed), "url(") {
			urlPath, err := extractURL(selector, property, value)
			if err != nil {
				return err
			}
			entry.ItalicFont, entry.HasItalicFont = urlPath, true
			return nil
		}
		name := strings.Trim(trimmed, "\"'")
		if name == "" {
			return fmt.Errorf("skin: %s in %q cannot be empty", property, selector)
		}
		entry.ItalicFont, entry.HasItalicFont = name, true
		return nil
	default:
		return fmt.Errorf("skin: unknown text property %q in %q", property, selector)
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

// gradientDirections maps CSS direction keywords to GradientDirection enum values.
var gradientDirections = map[string]GradientDirection{
	"to bottom":       GradientToBottom,
	"to top":          GradientToTop,
	"to right":        GradientToRight,
	"to left":         GradientToLeft,
	"to bottom right": GradientToBottomRight,
	"to bottom left":  GradientToBottomLeft,
	"to top right":    GradientToTopRight,
	"to top left":     GradientToTopLeft,
}

// parseLinearGradient parses a linear-gradient(...) CSS declaration.
func parseLinearGradient(selector, property, value string) (LinearGradient, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, ")") {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q has unclosed linear-gradient", property, selector)
	}
	prefix := "linear-gradient("
	inner := strings.TrimSpace(trimmed[len(prefix) : len(trimmed)-1])
	if inner == "" {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q has empty linear-gradient", property, selector)
	}
	args, err := splitParenArgs(inner)
	if err != nil {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q: %w", property, selector, err)
	}
	dir, stop0Str, stop1Str, err := resolveGradientArgs(args)
	if err != nil {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q: %w", property, selector, err)
	}
	stop0, err := parseColorStop(selector, property, stop0Str, 0.0)
	if err != nil {
		return LinearGradient{}, err
	}
	stop1, err := parseColorStop(selector, property, stop1Str, 1.0)
	if err != nil {
		return LinearGradient{}, err
	}
	return LinearGradient{
		Direction: dir,
		Stops:     [2]ColorStop{stop0, stop1},
	}, nil
}

// splitParenArgs splits s by comma only when parentheses are balanced.
func splitParenArgs(s string) ([]string, error) {
	var args []string
	start := 0
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unmatched ')' in %q", s)
			}
		case ',':
			if depth == 0 {
				item := strings.TrimSpace(s[start:i])
				if item == "" {
					return nil, fmt.Errorf("empty argument in %q", s)
				}
				args = append(args, item)
				start = i + 1
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unclosed '(' in %q", s)
	}
	last := strings.TrimSpace(s[start:])
	if last == "" {
		return nil, fmt.Errorf("trailing comma or empty argument in %q", s)
	}
	args = append(args, last)
	return args, nil
}

// resolveGradientArgs resolves direction and the two stop strings from parsed arguments.
func resolveGradientArgs(args []string) (GradientDirection, string, string, error) {
	if len(args) == 2 {
		return GradientToBottom, args[0], args[1], nil
	}
	if len(args) == 3 {
		dirStr := strings.Join(strings.Fields(strings.ToLower(args[0])), " ")
		dir, ok := gradientDirections[dirStr]
		if !ok {
			return 0, "", "", fmt.Errorf("invalid gradient direction %q", args[0])
		}
		return dir, args[1], args[2], nil
	}
	return 0, "", "", fmt.Errorf("expected 2 color stops (got %d arguments)", len(args))
}

// parseColorStop parses a single color stop with optional percentage position.
func parseColorStop(selector, property, stopStr string, defaultPos float32) (ColorStop, error) {
	fields := strings.Fields(stopStr)
	if len(fields) == 0 {
		return ColorStop{}, fmt.Errorf("skin: %s in %q has empty color stop", property, selector)
	}
	col, err := parseTint(selector, property, fields[0])
	if err != nil {
		return ColorStop{}, err
	}
	pos := defaultPos
	if len(fields) == 2 {
		posStr := fields[1]
		if !strings.HasSuffix(posStr, "%") {
			return ColorStop{}, fmt.Errorf("skin: %s in %q has invalid color stop position %q (must be percentage)", property, selector, posStr)
		}
		numStr := strings.TrimSuffix(posStr, "%")
		p, err := strconv.ParseFloat(numStr, 32)
		if err != nil || p < 0 || p > 100 {
			return ColorStop{}, fmt.Errorf("skin: %s in %q has invalid color stop position %q", property, selector, posStr)
		}
		pos = float32(p / 100.0)
	} else if len(fields) > 2 {
		return ColorStop{}, fmt.Errorf("skin: %s in %q has malformed color stop %q", property, selector, stopStr)
	}
	return ColorStop{Color: col, Position: pos}, nil
}
