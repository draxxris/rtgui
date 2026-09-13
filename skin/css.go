// Package skin maps widget styling to draw descriptors. This file adds the
// headless half of CSS file skinning: it parses a LOOK-only CSS subset into
// SkinRules without touching GL, so everything here is unit-testable without
// a display. Texture upload and registry writes live in render/css.go.
//
// Supported grammar: Kind[::part][:pseudo] and * selectors with properties
// (border-image-source, border-image-slice, background-image, background-image-tint,
// background-color, border-image-source-tint, border-radius, padding, color, font-size,
// font-family, font-italic-family). Anything else is a hard error. Image sources
// accept url(...) or none; none drops inherited textures and gradients.
//
// Gradient layers for background-image: one to four comma-separated layers,
// first layer on top. Each layer is linear-gradient([to <dir> | <angle>deg,]
// <color> [<pos>%], ...) with 2-4 stops, or radial-gradient([circle
// [at <x>% <y>%],] <color> [<pos>%], ...) with 2-4 stops. A background
// stack may instead start with one url(...) texture followed by up to four
// gradient layers; the URL must be first and none cannot be mixed. Missing
// stop positions follow CSS Images 3 (ends anchor at 0%/100%, runs interpolate).
// Example: radial-gradient(circle at 30% 20%, #3a5a7a80, #00000000 60%),
// linear-gradient(135deg, #17b978 0%, #0ea071 50%, #086972 100%),
// linear-gradient(to bottom, #2b3d54, #0b1524),
// linear-gradient(to top, #00000066, #00000000 30%).
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

	// NoTexture reports an explicit none that drops inherited textures and
	// gradients through Overlay. Background and border rules never share an
	// entry, so one flag serves both properties.
	NoTexture bool

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

	// Gradients holds parsed layers, first layer on top, when declared.
	Gradients [MaxGradientLayers]Gradient
	// GradientCount is the used prefix length of Gradients.
	GradientCount int

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
	"List":        core.WidgetList,
	"ChatLog":     core.WidgetChatLog,
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
	core.WidgetList:        {"highlight": true, "track": true, "thumb": true},
	core.WidgetChatLog:     {"highlight": true, "track": true, "thumb": true},
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
	if (kind == core.WidgetScrollPanel || kind == core.WidgetList || kind == core.WidgetChatLog) && (part == PartTrack || part == PartThumb) {
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
		if lower == "none" {
			entry.Image, entry.HasImage = "", false
			entry.Gradients, entry.GradientCount = [MaxGradientLayers]Gradient{}, 0
			entry.NoTexture = true
			return nil
		}
		if strings.HasPrefix(lower, "url(") {
			return parseBackgroundTextureStack(entry, selector, property, trimmed)
		}
		if strings.Contains(lower, "linear-gradient(") || strings.Contains(lower, "radial-gradient(") {
			layers, count, err := parseBackgroundGradients(selector, property, value)
			if err != nil {
				return err
			}
			entry.Gradients, entry.GradientCount = layers, count
			entry.Image, entry.HasImage = "", false
			entry.NoTexture = false
			return nil
		}
		return fmt.Errorf("skin: %s in %q must be none, url(...), gradients, or a leading url(...) followed by gradients, got %q", property, selector, value)
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

// parseBackgroundTextureStack parses one leading texture and its lower CSS
// layers. Only the first comma-separated item may be url(...); all remaining
// items must be gradients so the renderer can preserve CSS paint order.
func parseBackgroundTextureStack(entry *SkinRule, selector, property, value string) error {
	parts, err := splitParenArgs(value)
	if err != nil {
		return fmt.Errorf("skin: %s in %q: %w", property, selector, err)
	}
	if len(parts) > MaxGradientLayers+1 {
		return fmt.Errorf("skin: %s in %q allows one texture and at most %d gradient layers", property, selector, MaxGradientLayers)
	}
	path, err := extractURL(selector, property, parts[0])
	if err != nil {
		return err
	}
	entry.Image, entry.HasImage = path, true
	entry.Gradients, entry.GradientCount = [MaxGradientLayers]Gradient{}, 0
	entry.NoTexture = false
	for index, raw := range parts[1:] {
		gradient, parseErr := parseGradientLayer(selector, property, raw)
		if parseErr != nil {
			return parseErr
		}
		entry.Gradients[index] = gradient
		entry.GradientCount++
	}
	return nil
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
// A none source drops the inherited ring.
func applyBorderProp(entry *SkinRule, selector, property, value string) error {
	switch property {
	case "border-image-source":
		if strings.ToLower(strings.TrimSpace(value)) == "none" {
			entry.Image, entry.HasImage = "", false
			entry.NoTexture = true
			return nil
		}
		path, err := extractURL(selector, property, value)
		if err != nil {
			return err
		}
		entry.Image, entry.HasImage = path, true
		entry.NoTexture = false
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

// parseBackgroundGradients parses comma-separated gradient layers.
// CSS paints the first layer on top; rendering composites back-to-front.
func parseBackgroundGradients(selector, property, value string) ([MaxGradientLayers]Gradient, int, error) {
	var out [MaxGradientLayers]Gradient
	layers, splitErr := splitParenArgs(strings.TrimSpace(value))
	if splitErr != nil {
		return out, 0, fmt.Errorf("skin: %s in %q: %w", property, selector, splitErr)
	}
	if len(layers) < 1 || len(layers) > MaxGradientLayers {
		return out, 0, fmt.Errorf("skin: %s in %q expects 1-%d gradient layers (got %d)", property, selector, MaxGradientLayers, len(layers))
	}
	for i, layer := range layers {
		grad, parseErr := parseGradientLayer(selector, property, layer)
		if parseErr != nil {
			return out, 0, parseErr
		}
		out[i] = grad
	}
	return out, len(layers), nil
}

// HasGradient reports whether the rule declares any gradient layer.
func (r SkinRule) HasGradient() bool {
	return r.GradientCount > 0
}

// parseGradientLayer dispatches one layer to its linear or radial parser.
func parseGradientLayer(selector, property, layer string) (LinearGradient, error) {
	trimmed := strings.TrimSpace(layer)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "linear-gradient(") {
		return parseLinearGradient(selector, property, trimmed)
	}
	if strings.HasPrefix(lower, "radial-gradient(") {
		return parseRadialGradient(selector, property, trimmed)
	}
	if strings.HasPrefix(lower, "inner-gradient(") {
		return parseInnerGradient(selector, property, trimmed)
	}
	return LinearGradient{}, fmt.Errorf("skin: %s in %q must be linear-gradient(...), radial-gradient(...), or inner-gradient(...), got %q", property, selector, layer)
}

// parseLinearGradient parses a single linear-gradient(...) layer.
// It accepts an optional direction or angle head plus 2-4 color stops.
func parseLinearGradient(selector, property, value string) (LinearGradient, error) {
	inner, err := gradientInner(selector, property, value, "linear-gradient(")
	if err != nil {
		return LinearGradient{}, err
	}
	if inner == "" {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q has empty linear-gradient", property, selector)
	}
	args, err := splitParenArgs(inner)
	if err != nil {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q: %w", property, selector, err)
	}
	return buildLinearGradient(selector, property, args)
}

// gradientInner unwraps the parenthesized body of one gradient function.
func gradientInner(selector, property, value, prefix string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, ")") {
		return "", fmt.Errorf("skin: %s in %q has unclosed %s", property, selector, strings.TrimSuffix(prefix, "("))
	}
	if len(trimmed) <= len(prefix) {
		return "", fmt.Errorf("skin: %s in %q has empty %s", property, selector, strings.TrimSuffix(prefix, "("))
	}
	return strings.TrimSpace(trimmed[len(prefix) : len(trimmed)-1]), nil
}

// buildLinearGradient resolves the optional head and 2-4 stops.
func buildLinearGradient(selector, property string, args []string) (LinearGradient, error) {
	head, stopArgs, err := splitLinearHead(args)
	if err != nil {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q: %w", property, selector, err)
	}
	if len(stopArgs) < 2 || len(stopArgs) > MaxGradientStops {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q expects 2-%d color stops (got %d)", property, selector, MaxGradientStops, len(stopArgs))
	}
	stops := make([]ColorStop, 0, len(stopArgs))
	for _, raw := range stopArgs {
		stop, parseErr := parseColorStopAuto(selector, property, raw)
		if parseErr != nil {
			return LinearGradient{}, parseErr
		}
		stops = append(stops, stop)
	}
	if head.isAngle {
		grad, ok := NewAngleGradient(head.angle, stops...)
		if !ok {
			return LinearGradient{}, fmt.Errorf("skin: %s in %q has invalid stops", property, selector)
		}
		return grad, nil
	}
	grad, ok := NewLinearGradient(head.dir, stops...)
	if !ok {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q has invalid stops", property, selector)
	}
	return grad, nil
}

// linearHead carries a resolved direction or angle prefix.
type linearHead struct {
	dir     GradientDirection
	isAngle bool
	angle   float32
}

// splitLinearHead separates an optional direction or angle from stops.
func splitLinearHead(args []string) (linearHead, []string, error) {
	if len(args) == 0 {
		return linearHead{}, nil, fmt.Errorf("expected at least 2 color stops")
	}
	head, isHead, err := parseGradientHead(args[0])
	if err != nil {
		return linearHead{}, nil, err
	}
	if !isHead {
		return linearHead{dir: GradientToBottom}, args, nil
	}
	return head, args[1:], nil
}

// parseGradientHead detects a direction keyword or CSS angle.
// It reports isHead=false for color stops so callers keep the default.
func parseGradientHead(arg string) (linearHead, bool, error) {
	normalized := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(arg))), " ")
	if dir, ok := gradientDirections[normalized]; ok {
		return linearHead{dir: dir}, true, nil
	}
	if strings.HasSuffix(normalized, "deg") {
		angle, err := parseGradientAngle(strings.TrimSpace(arg))
		if err != nil {
			return linearHead{}, true, err
		}
		return linearHead{isAngle: true, angle: angle}, true, nil
	}
	return linearHead{}, false, nil
}

// parseGradientAngle parses a CSS angle like 135deg.
func parseGradientAngle(arg string) (float32, error) {
	trimmed := strings.TrimSpace(arg)
	lower := strings.ToLower(trimmed)
	if !strings.HasSuffix(lower, "deg") {
		return 0, fmt.Errorf("invalid gradient angle %q", arg)
	}
	numStr := strings.TrimSpace(trimmed[:len(trimmed)-3])
	angle, err := strconv.ParseFloat(numStr, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid gradient angle %q", arg)
	}
	return float32(angle), nil
}

// parseRadialGradient parses a radial-gradient(circle at X% Y%, stops...).
// The center defaults to 50% 50% and the radius reaches the farthest corner.
func parseRadialGradient(selector, property, value string) (LinearGradient, error) {
	inner, err := gradientInner(selector, property, value, "radial-gradient(")
	if err != nil {
		return LinearGradient{}, err
	}
	if inner == "" {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q has empty radial-gradient", property, selector)
	}
	args, err := splitParenArgs(inner)
	if err != nil {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q: %w", property, selector, err)
	}
	return buildRadialGradient(selector, property, args)
}

// parseInnerGradient parses an inner-gradient(stop, ...) layer. Stops run
// from every border (position 0) to the center (position 1) with no head
// argument, so one layer replaces four directional linear gradients.
func parseInnerGradient(selector, property, value string) (LinearGradient, error) {
	inner, err := gradientInner(selector, property, value, "inner-gradient(")
	if err != nil {
		return LinearGradient{}, err
	}
	if inner == "" {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q has empty inner-gradient", property, selector)
	}
	args, err := splitParenArgs(inner)
	if err != nil {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q: %w", property, selector, err)
	}
	if len(args) < 2 || len(args) > MaxGradientStops {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q expects 2-%d color stops (got %d)", property, selector, MaxGradientStops, len(args))
	}
	stops := make([]ColorStop, 0, len(args))
	for _, raw := range args {
		stop, parseErr := parseColorStopAuto(selector, property, raw)
		if parseErr != nil {
			return LinearGradient{}, parseErr
		}
		stops = append(stops, stop)
	}
	grad, ok := NewInnerGradient(stops...)
	if !ok {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q has invalid inner stops", property, selector)
	}
	return grad, nil
}

// buildRadialGradient resolves the optional center and 2-4 stops.
func buildRadialGradient(selector, property string, args []string) (LinearGradient, error) {
	cx, cy := float32(0.5), float32(0.5)
	stopArgs := args
	if len(args) > 0 && isRadialCenterArg(args[0]) {
		parsedX, parsedY, err := parseRadialCenter(args[0])
		if err != nil {
			return LinearGradient{}, fmt.Errorf("skin: %s in %q: %w", property, selector, err)
		}
		cx, cy = parsedX, parsedY
		stopArgs = args[1:]
	}
	if len(stopArgs) < 2 || len(stopArgs) > MaxGradientStops {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q expects 2-%d color stops (got %d)", property, selector, MaxGradientStops, len(stopArgs))
	}
	stops := make([]ColorStop, 0, len(stopArgs))
	for _, raw := range stopArgs {
		stop, parseErr := parseColorStopAuto(selector, property, raw)
		if parseErr != nil {
			return LinearGradient{}, parseErr
		}
		stops = append(stops, stop)
	}
	grad, ok := NewRadialGradient(cx, cy, stops...)
	if !ok {
		return LinearGradient{}, fmt.Errorf("skin: %s in %q has invalid radial stops", property, selector)
	}
	return grad, nil
}

// isRadialCenterArg reports whether an argument selects the circle center.
func isRadialCenterArg(arg string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(arg)), "circle")
}

// parseRadialCenter parses circle or circle at X% Y% into unit coordinates.
func parseRadialCenter(arg string) (float32, float32, error) {
	fields := strings.Fields(strings.TrimSpace(arg))
	if len(fields) == 1 && strings.ToLower(fields[0]) == "circle" {
		return 0.5, 0.5, nil
	}
	if len(fields) == 4 && strings.ToLower(fields[0]) == "circle" && strings.ToLower(fields[1]) == "at" {
		x, err := parsePercent(fields[2])
		if err != nil {
			return 0, 0, err
		}
		y, err := parsePercent(fields[3])
		if err != nil {
			return 0, 0, err
		}
		return x, y, nil
	}
	return 0, 0, fmt.Errorf("invalid radial center %q (want circle or circle at X%% Y%%)", arg)
}

// parsePercent parses a 0-100% token into a unit fraction.
func parsePercent(token string) (float32, error) {
	if !strings.HasSuffix(token, "%") {
		return 0, fmt.Errorf("invalid percentage %q (must end with %%)", token)
	}
	num, err := strconv.ParseFloat(strings.TrimSuffix(token, "%"), 32)
	if err != nil || num < 0 || num > 100 {
		return 0, fmt.Errorf("invalid percentage %q", token)
	}
	return float32(num / 100), nil
}

// parseColorStopAuto parses one stop with an optional percentage position.
// Missing positions return Position -1 so constructors distribute them.
func parseColorStopAuto(selector, property, stopStr string) (ColorStop, error) {
	fields := strings.Fields(stopStr)
	if len(fields) == 0 {
		return ColorStop{}, fmt.Errorf("skin: %s in %q has empty color stop", property, selector)
	}
	col, err := parseTint(selector, property, fields[0])
	if err != nil {
		return ColorStop{}, err
	}
	if len(fields) == 1 {
		return ColorStop{Color: col, Position: -1}, nil
	}
	if len(fields) == 2 {
		pos, err := parsePercent(fields[1])
		if err != nil {
			return ColorStop{}, fmt.Errorf("skin: %s in %q has invalid color stop position %q (must be percentage)", property, selector, fields[1])
		}
		return ColorStop{Color: col, Position: pos}, nil
	}
	return ColorStop{}, fmt.Errorf("skin: %s in %q has malformed color stop %q", property, selector, stopStr)
}
