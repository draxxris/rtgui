package text

import (
	"strings"

	"github.com/draxxris/rtgui/core"
)

// IconAllowed reports whether an inline icon name is whitelisted.
// The parser calls it for every [icon=name] candidate; a false result
// renders the tag as literal text.
type IconAllowed func(name string) bool

// ParsePlayerMarkup parses untrusted player markup into display segments.
// It recognizes only [icon=name] and [link=scheme:target]text[/link];
// every other bracket form, including color, bold, size, and font markup,
// renders as literal text. Go code constructs those styles directly.
//
// Unknown icons, unknown link schemes, malformed tags, and empty link
// bodies render literally so abuse stays visible. A nil allowed treats
// every icon as unknown. It returns nil for empty input.
func ParsePlayerMarkup(input string, allowed IconAllowed) []core.RichSegment {
	if input == "" {
		return nil
	}
	parser := markupParser{allowed: allowed}
	parser.parseTop(input)
	return parser.finish()
}

// markupParser accumulates merged plain runs and tagged segments.
type markupParser struct {
	allowed IconAllowed
	out     []core.RichSegment
	plain   strings.Builder
}

// finish flushes trailing plain text and reports the segments.
func (p *markupParser) finish() []core.RichSegment {
	p.flushPlain()
	if len(p.out) == 0 {
		return nil
	}
	return p.out
}

// flushPlain merges buffered literal text into one plain segment.
func (p *markupParser) flushPlain() {
	if p == nil || p.plain.Len() == 0 {
		return
	}
	p.out = append(p.out, core.RichSegment{Text: p.plain.String()})
	p.plain.Reset()
}

// appendIconSegment emits one whitelisted icon run.
func (p *markupParser) appendIconSegment(name string, link core.Link) {
	p.flushPlain()
	p.out = append(p.out, core.RichSegment{Icon: name, HasIcon: true, Link: link})
}

// parseTop scans one message allowing icons and links.
func (p *markupParser) parseTop(input string) {
	pos := 0
	for pos < len(input) {
		if input[pos] != '[' {
			p.plain.WriteByte(input[pos])
			pos++
			continue
		}
		if pos+1 < len(input) && input[pos+1] == '[' {
			p.plain.WriteByte('[')
			pos += 2
			continue
		}
		if name, next, ok := scanIconTag(input, pos); ok {
			if validIconName(name) && p.iconAllowed(name) {
				p.appendIconSegment(name, core.Link{})
				pos = next
				continue
			}
			p.plain.WriteString(input[pos:next])
			pos = next
			continue
		}
		if link, next, ok := scanLinkOpen(input, pos); ok {
			body, after, found := scanLinkBody(input, next)
			if !found {
				p.plain.WriteString(input[pos:next])
				pos = next
				continue
			}
			inner := parseLinkInner(body, p.allowed, link)
			if len(inner) == 0 {
				p.plain.WriteString(input[pos:after])
				pos = after
				continue
			}
			p.flushPlain()
			p.out = append(p.out, inner...)
			pos = after
			continue
		}
		p.plain.WriteByte('[')
		pos++
	}
}

// parseLinkInner parses a link body allowing icons and plain text only.
// Nested link openers render literally; the zero link tags every run.
func parseLinkInner(body string, allowed IconAllowed, link core.Link) []core.RichSegment {
	inner := &markupParser{allowed: allowed}
	pos := 0
	for pos < len(body) {
		if body[pos] != '[' {
			inner.plain.WriteByte(body[pos])
			pos++
			continue
		}
		if pos+1 < len(body) && body[pos+1] == '[' {
			inner.plain.WriteByte('[')
			pos += 2
			continue
		}
		if name, next, ok := scanIconTag(body, pos); ok {
			if validIconName(name) && inner.iconAllowed(name) {
				inner.flushPlain()
				inner.out = append(inner.out, core.RichSegment{Icon: name, HasIcon: true, Link: link})
				pos = next
				continue
			}
			inner.plain.WriteString(body[pos:next])
			pos = next
			continue
		}
		inner.plain.WriteByte('[')
		pos++
	}
	inner.flushPlain()
	for i := range inner.out {
		if !inner.out[i].HasIcon {
			inner.out[i].Link = link
		}
	}
	return nonEmptySegments(inner.out)
}

// iconAllowed probes the parser whitelist without allocating.
func (p *markupParser) iconAllowed(name string) bool {
	if p == nil || p.allowed == nil {
		return false
	}
	return p.allowed(name)
}

// scanIconTag matches [icon=name] at pos and reports the name and end.
func scanIconTag(input string, pos int) (string, int, bool) {
	const prefix = "[icon="
	if !hasPrefixAt(input, pos, prefix) {
		return "", 0, false
	}
	start := pos + len(prefix)
	end := indexByteAt(input, ']', start)
	if end < 0 {
		return "", 0, false
	}
	return input[start:end], end + 1, true
}

// scanLinkOpen matches [link=scheme:target] at pos and reports the link.
func scanLinkOpen(input string, pos int) (core.Link, int, bool) {
	const prefix = "[link="
	if !hasPrefixAt(input, pos, prefix) {
		return core.Link{}, 0, false
	}
	start := pos + len(prefix)
	end := indexByteAt(input, ']', start)
	if end < 0 {
		return core.Link{}, 0, false
	}
	link, ok := parseLinkRef(input[start:end])
	if !ok {
		return core.Link{}, 0, false
	}
	return link, end + 1, true
}

// scanLinkBody finds the first [/link] closer after open and splits body.
func scanLinkBody(input string, open int) (string, int, bool) {
	const closer = "[/link]"
	rel := indexOfAt(input, closer, open)
	if rel < 0 {
		return "", 0, false
	}
	return input[open:rel], rel + len(closer), true
}

// parseLinkRef splits scheme:target and maps the scheme to a link kind.
func parseLinkRef(ref string) (core.Link, bool) {
	colon := strings.IndexByte(ref, ':')
	if colon <= 0 || colon+1 >= len(ref) {
		return core.Link{}, false
	}
	kind, ok := linkKindForScheme(ref[:colon])
	if !ok {
		return core.Link{}, false
	}
	target := strings.TrimSpace(ref[colon+1:])
	if !validLinkTarget(target) {
		return core.Link{}, false
	}
	return core.Link{Kind: kind, Target: target}, true
}

// linkKindForScheme maps a markup scheme to its link kind.
func linkKindForScheme(scheme string) (core.LinkKind, bool) {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "url":
		return core.LinkURL, true
	case "item":
		return core.LinkItem, true
	case "player":
		return core.LinkPlayer, true
	case "location":
		return core.LinkLocation, true
	case "quest":
		return core.LinkQuest, true
	case "custom":
		return core.LinkCustom, true
	default:
		return core.LinkNone, false
	}
}

// validIconName constrains whitelist keys to portable path characters.
func validIconName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for i := 0; i < len(name); i++ {
		value := name[i]
		switch {
		case value >= 'a' && value <= 'z':
		case value >= 'A' && value <= 'Z':
		case value >= '0' && value <= '9':
		case value == '-' || value == '_' || value == '/' || value == '.' || value == '+':
		default:
			return false
		}
	}
	return true
}

// validLinkTarget constrains opaque targets without interpreting them.
func validLinkTarget(target string) bool {
	if len(target) == 0 || len(target) > 128 {
		return false
	}
	for i := 0; i < len(target); i++ {
		value := target[i]
		if value == '[' || value == ']' || value < 0x20 || value == 0x7f {
			return false
		}
	}
	return true
}

// nonEmptySegments drops empty text runs while keeping every icon run.
func nonEmptySegments(segments []core.RichSegment) []core.RichSegment {
	kept := segments[:0]
	hasContent := false
	for _, segment := range segments {
		if segment.HasIcon {
			kept = append(kept, segment)
			hasContent = true
			continue
		}
		if segment.Text == "" {
			continue
		}
		kept = append(kept, segment)
		if strings.TrimSpace(segment.Text) != "" {
			hasContent = true
		}
	}
	if !hasContent {
		return nil
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// hasPrefixAt reports whether prefix matches input at pos.
func hasPrefixAt(input string, pos int, prefix string) bool {
	if pos < 0 || pos+len(prefix) > len(input) {
		return false
	}
	return input[pos:pos+len(prefix)] == prefix
}

// indexByteAt finds value at or after start, or -1.
func indexByteAt(input string, value byte, start int) int {
	if start < 0 {
		start = 0
	}
	for i := start; i < len(input); i++ {
		if input[i] == value {
			return i
		}
	}
	return -1
}

// indexOfAt finds token at or after start, or -1.
func indexOfAt(input, token string, start int) int {
	if start < 0 {
		start = 0
	}
	if token == "" || start+len(token) > len(input) {
		if token == "" {
			return start
		}
		return -1
	}
	for i := start; i+len(token) <= len(input); i++ {
		if input[i:i+len(token)] == token {
			return i
		}
	}
	return -1
}
