package text_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/text"
)

// allowList builds a whitelist probe from names for parser tests.
func allowList(names ...string) text.IconAllowed {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return func(name string) bool { return set[name] }
}

// TestParsePlayerMarkupPlain passes text without tags through untouched.
func TestParsePlayerMarkupPlain(t *testing.T) {
	segments := text.ParsePlayerMarkup("hello world", nil)
	if len(segments) != 1 || segments[0].Text != "hello world" {
		t.Fatalf("plain = %+v", segments)
	}
	if got := text.ParsePlayerMarkup("", nil); len(got) != 0 {
		t.Fatalf("empty = %+v", got)
	}
}

// TestParsePlayerMarkupIcon whitelists icons and renders misses literally.
func TestParsePlayerMarkupIcon(t *testing.T) {
	allowed := allowList("iron-plate")
	segments := text.ParsePlayerMarkup("a [icon=iron-plate] b", allowed)
	if len(segments) != 3 || !segments[1].HasIcon || segments[1].Icon != "iron-plate" {
		t.Fatalf("icon = %+v", segments)
	}
	denied := text.ParsePlayerMarkup("a [icon=evil] b", allowed)
	if len(denied) != 1 || denied[0].HasIcon {
		t.Fatalf("denied icon parsed = %+v", denied)
	}
	nilAllowed := text.ParsePlayerMarkup("[icon=iron-plate]", nil)
	if len(nilAllowed) != 1 || nilAllowed[0].HasIcon {
		t.Fatalf("nil whitelist parsed = %+v", nilAllowed)
	}
}

// TestParsePlayerMarkupLink validates schemes, bodies, and nesting.
func TestParsePlayerMarkupLink(t *testing.T) {
	segments := text.ParsePlayerMarkup("[link=item:iron-plate]iron[/link]", nil)
	if len(segments) != 1 || segments[0].Link.Kind != core.LinkItem || segments[0].Text != "iron" {
		t.Fatalf("link = %+v", segments)
	}
	multi := text.ParsePlayerMarkup("[link=url:https://example.com]wiki[/link] and [link=player:Bob]Bob[/link]", nil)
	if len(multi) != 3 || multi[0].Link.Kind != core.LinkURL || multi[2].Link.Kind != core.LinkPlayer {
		t.Fatalf("multi = %+v", multi)
	}
	unknown := text.ParsePlayerMarkup("[link=bogus:x]y[/link]", nil)
	if len(unknown) != 1 || unknown[0].Link.Kind != core.LinkNone {
		t.Fatalf("unknown scheme parsed = %+v", unknown)
	}
	unclosed := text.ParsePlayerMarkup("[link=item:x]oops", nil)
	if len(unclosed) != 1 || unclosed[0].Link.Kind != core.LinkNone {
		t.Fatalf("unclosed parsed = %+v", unclosed)
	}
	empty := text.ParsePlayerMarkup("[link=item:x][/link]", nil)
	if len(empty) != 1 || empty[0].Link.Kind != core.LinkNone {
		t.Fatalf("empty body parsed = %+v", empty)
	}
}

// TestParsePlayerMarkupEscapeAndStyle keeps escapes and rejects styling.
func TestParsePlayerMarkupEscapeAndStyle(t *testing.T) {
	escaped := text.ParsePlayerMarkup("[[icon=x]", nil)
	if len(escaped) != 1 || escaped[0].Text != "[icon=x]" {
		t.Fatalf("escape = %+v", escaped)
	}
	styled := text.ParsePlayerMarkup("[color=red]hi[/color] [b]yo[/b]", nil)
	if len(styled) != 1 || styled[0].Link.Kind != core.LinkNone || styled[0].HasIcon {
		t.Fatalf("style parsed = %+v", styled)
	}
}

// TestParsePlayerMarkupIconInLink keeps clickable icons inside links.
func TestParsePlayerMarkupIconInLink(t *testing.T) {
	allowed := allowList("iron-plate")
	segments := text.ParsePlayerMarkup("[link=item:iron-plate]take [icon=iron-plate][/link]", allowed)
	if len(segments) != 2 {
		t.Fatalf("inner icon count = %+v", segments)
	}
	for _, segment := range segments {
		if segment.Link.Kind != core.LinkItem {
			t.Fatalf("inner link lost = %+v", segments)
		}
	}
	if !segments[1].HasIcon {
		t.Fatalf("inner icon lost = %+v", segments)
	}
}
